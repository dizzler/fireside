package fireside

import (
	"archive/tar"
	"bufio"
	"compress/gzip"
	"fmt"
	"errors"
	"io"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"time"

	configure "fireside/pkg/configure"
        log "github.com/sirupsen/logrus"

        "github.com/aws/aws-sdk-go/aws"
        "github.com/aws/aws-sdk-go/aws/session"
        "github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/dailyburn/ratchet/data"
	"github.com/dailyburn/ratchet/logger"
)

// FsCacheWriter struct provides configuration for a new filesystem cache writer.
type FsCacheWriter struct {
	ActiveBuf    *bufio.Writer
	ActiveFile   *os.File
	ActivePath   string
	ArchivePaths []string
	BaseDir      string
	FilePrefix   string
	OutputConfig *OutputConfig
}

// NewFsCacheWriter returns a new FsCacheWriter wrapping the given io.Writer object
func NewFsCacheWriter(outdir string, prefix string, outconf *OutputConfig) *FsCacheWriter {
	// Initialize the filesystem cache
	p, f, b, err := OpenCacheFile(outdir, prefix)
	if err != nil {
            log.WithError(err).Fatal("error initializing filesystem cache")
	}
	var paths []string
	cacheWriter := &FsCacheWriter{ActiveBuf: b, ActiveFile: f, ActivePath: p, ArchivePaths: paths, BaseDir: outdir, FilePrefix: prefix, OutputConfig: outconf}
	// Run and manage the filesystem cache in a separate goroutine
	go RunFsCache(cacheWriter)
	return cacheWriter
}

// ProcessData writes the data
func (w *FsCacheWriter) ProcessData(d data.JSON, outputChan chan data.JSON, killChan chan error) {
	var bytesWritten int = 0
	// Send the JSON data to the filesystem cache
	// Append the string to the output file for the current minute
	bytesWritten, _ = w.ActiveBuf.WriteString(string(d) + "\n")

	logger.Debug("FsCacheWriter:", bytesWritten, "bytes written")
}

// Finish - see interface for documentation.
func (w *FsCacheWriter) Finish(outputChan chan data.JSON, killChan chan error) {
}

func (w *FsCacheWriter) String() string {
	return "FsCacheWriter"
}

func check(e error) {
       if e != nil {
	   panic(e)
       }
}

// Generate a name for a given cache or archive file based on the current time
func NameCacheFile(baseDir string, filePrefix string, fileSuffix string) (string) {
	const layout = "2006-01-02MST15-04"
	now := time.Now()
        return baseDir + "/" + filePrefix + "-" + now.Format(layout) + "." + fileSuffix
}

// Create the active cache file and return its details as pointers
func OpenCacheFile(baseDir string, filePrefix string) (string, *os.File, *bufio.Writer, error) {
	p := NameCacheFile(baseDir, filePrefix, OutputSuffix)
	var f *os.File
	var b *bufio.Writer

	// Create the output directory as needed
	merr := os.MkdirAll(baseDir, 0750)
	if merr != nil {
		log.Error(merr)
		return p, f, b, merr
	}

	// Create/open the file at path 'p' as needed
        f, err := os.OpenFile(p, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0640)
	if err != nil {
		log.Error(err)
		return p, f, b, merr
	}

	// Create/open the buffered Writer using the file we just opened
	b = bufio.NewWriterSize(f, 4096)
        return p, f, b, err
}

func RunFsCache(cacher *FsCacheWriter) {
	log.Debug("running RunFsCache function")
	// Create the output directory as needed
	merr := os.MkdirAll(cacher.BaseDir, 0750)
	if merr != nil {
		log.Error(merr)
	}

	// Use a ticker to trigger rotation of the active cache file and buffer
	var rotateSeconds int = 10
	rotateInterval := time.Duration(rotateSeconds)
	rotateTicker := time.NewTicker(rotateInterval * time.Second)
	// Use another ticker to trigger cache cleanup
	var cleanSeconds int = 60
	cleanInterval := time.Duration(cleanSeconds)
	cleanTicker := time.NewTicker(cleanInterval * time.Second)
	// Use another ticker to trigger archive upload
	var archiverSeconds int = 5
	archiverInterval := time.Duration(archiverSeconds)
	archiverTicker := time.NewTicker(archiverInterval * time.Second)
	// Use a common 'quit' channel to stop both tickers as needed
	quit := make(chan struct{})
	go func() {
		for {
			select {
			case <- rotateTicker.C:
				go RotateFsCache(cacher)
			case <- cleanTicker.C:
				go CleanFsCache(cacher)
			case <- archiverTicker.C:
				if err := UploadArchives(cacher); err != nil {
					log.Error(err)
				}
			case <- quit:
				rotateTicker.Stop()
				cleanTicker.Stop()
				archiverTicker.Stop()
				return
		        }
		}
	}()
}

func RotateFsCache(cacher *FsCacheWriter) {
	log.Debug("running RotateFsCache function")
	oldBuf := cacher.ActiveBuf
	oldCacheFile := cacher.ActiveFile
	cachePath, cacheFile, cacheBuf, err := OpenCacheFile(cacher.BaseDir, cacher.FilePrefix)
	if err != nil { log.Error(err) }

	// Rotate old and new cache pointers if paths do not match
	if cacher.ActivePath != cachePath {
		cacher.ActiveBuf  = cacheBuf
		cacher.ActivePath = cachePath
		cacher.ActiveFile = cacheFile

		// Flush the old IO buffer
		err := oldBuf.Flush()
		if err != nil { log.Error(err) }

		// Close the old cache file
		err = oldCacheFile.Close()
		if err != nil { log.Error(err) }
	}
}

func CleanFsCache(cacher *FsCacheWriter) {
	log.Debug("running CleanFsCache function")
	// Clean the filesystem cache by removing empty and/or old files
	files, err := ioutil.ReadDir(cacher.BaseDir)
	if err != nil {
		log.Error(err)
	}
	for _, file := range files {
		filePath := path.Join([]string{cacher.BaseDir, file.Name()}...)
		// Avoid archiving data which is either (1) active OR (2) already archived
		if filePath != cacher.ActivePath && strings.HasSuffix(filePath, configure.OutputSuffix) {
			log.Debug("checking cache file for archive and/or cleanup : " + filePath)
			fileInfo, ferr := os.Stat(filePath)
			if ferr != nil {
				log.Error(ferr)
			}
			switch mode := fileInfo.Mode(); {
			case mode.IsDir():
				log.Debug("skipping removal of directory : " + filePath)
			case mode.IsRegular():
				if fileInfo.Size() == 0 {
					log.Debug("removing empty cache file : " + filePath)
					rerr := os.RemoveAll(filePath)
					if rerr != nil {
						log.Error(rerr)
					}
				} else {
					a_err := ArchiveCacheFile(cacher, filePath, fileInfo)
					if a_err != nil {
						log.Error(a_err)
					}
				}
			}
		} else {
			log.Debug("skipping archive/cleanup task for file : " + filePath)
		}
	}
}

func ArchiveCacheFile(cacher *FsCacheWriter, filepath string, fileinfo os.FileInfo) error {
	log.Debug("running ArchiveCacheFile function")
	// Generate the filepath for the archive file, based on the current time.Now()
	acf_name := NameCacheFile(cacher.BaseDir, cacher.FilePrefix, configure.ArchiveSuffix)
	acf, err := os.OpenFile(acf_name, os.O_RDWR|os.O_CREATE, 0640)
	defer acf.Close()
	if err != nil { return err }

	log.Debug("archiving cache data from :" + filepath +": to path :" + acf_name)

	// Create new Writers for gzip and tar. These writers are chained such that
	// writing to the tar writer will write to the gzip writer, which will write
	// to the "archive" writer
	gw := gzip.NewWriter(acf)
	defer gw.Close()
	tw := tar.NewWriter(gw)
	defer tw.Close()

	// Create a tar header from source FileInfo data
	header, err := tar.FileInfoHeader(fileinfo, fileinfo.Name())
	if err != nil { return err }
	header.Name = filepath

	// Write file header to tar archive
	err = tw.WriteHeader(header)
	if err != nil { return err }

	// Open source file as read-only
	var src *os.File
	src, err = os.OpenFile(filepath, os.O_RDONLY, 0640)
	if err != nil { return err }

	// Copy file contents to tar archive
	_, err = io.Copy(tw, src)
	if err != nil { return err }

	// Append the archive to the list of cacher ArchivePaths
	cacher.ArchivePaths = append(cacher.ArchivePaths, acf_name)

	// Delete old file after adding its contents to archive
	log.Info("removing cache file after adding to archive : " + filepath)
	err = os.RemoveAll(filepath)
	if err != nil { return err }

	return nil
}

func UploadArchives(cacher *FsCacheWriter) error {
	log.Debug("running UploadArchives function")
        var (
                bucket string
                region string
        )
        if len(cacher.OutputConfig.AWS.S3Bucket) > 0 {
                bucket = cacher.OutputConfig.AWS.S3Bucket

                if region = cacher.OutputConfig.AWS.Region ; len(region) == 0 {
                        return errors.New("cannot upload logs ; region is undefined for bucket " + bucket)
                }
		log.Debug("Configured to upload archives to AWS region " + region)

                archivePaths := cacher.ArchivePaths
                if len(archivePaths) == 0 {
                        log.Debug("empty list of archives to upload ; skipping...")
                        return nil
                }
                iter := NewS3UploadIterator(bucket, cacher.ArchivePaths)
                uploader := s3manager.NewUploader(session.New(&aws.Config{
                        Region: &region,
                }))
		log.Info(fmt.Printf("Uploading archive files to AWS Region %s : S3 Bucket %s", bucket, region))

                // Upload the list of archive files
                if err := uploader.UploadWithIterator(aws.BackgroundContext(), iter); err != nil {
                        return err
                }

                // Delete the archive files and remove from archive list after successful upload
                if delete_err := DeleteArchives(cacher, archivePaths); delete_err != nil {
                        return delete_err
                }

        } else {
	        return errors.New("cannot upload logs ; target S3 bucket is not set")
        }

	return nil
}

func DeleteArchives(cacher *FsCacheWriter, filePaths []string) error {
	for _, file := range filePaths {
		log.Debug("removing archive file : " + file)
		if err := os.RemoveAll(file); err != nil {
			return err
		}
		newitems := []string{}
		for _, af := range cacher.ArchivePaths {
			// remove uploaded file(s) from list of ArchivePaths
			if file != af {
				newitems = append(newitems, af)
			}
		}
		cacher.ArchivePaths = newitems
	}
	return nil
}
