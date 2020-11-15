package output_processors

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"time"

        log "github.com/sirupsen/logrus"

	"github.com/dailyburn/ratchet/data"
	"github.com/dailyburn/ratchet/logger"
)

const (
	ArchiveSuffix string = "tar.gz"
	OutputSuffix  string = "out"
)

// FsCacheWriter struct provides configuration for a new filesystem cache writer.
type FsCacheWriter struct {
	ActiveFile  *os.File
	ActivePath  string
	BaseDir     string
	FilePrefix  string
}

// NewFsCacheWriter returns a new FsCacheWriter wrapping the given io.Writer object
func NewFsCacheWriter(outdir string, prefix string) *FsCacheWriter {
	// Initialize the filesystem cache
	p, f, ferr := OpenCacheFile(outdir, prefix)
	if ferr != nil {
            log.WithError(ferr).Fatal("error initializing filesystem cache")
	}
	cacheWriter := &FsCacheWriter{ActiveFile: f, ActivePath: p, BaseDir: outdir, FilePrefix: prefix}
	// Run and manage the filesystem cache in a separate goroutine
	go RunFsCache(cacheWriter)
	return cacheWriter
}

// ProcessData writes the data
func (w *FsCacheWriter) ProcessData(d data.JSON, outputChan chan data.JSON, killChan chan error) {
	// Send the JSON data to the filesystem cache
	var bytesWritten int = 0
	// Append the string to the output file for the current minute
	bytesWritten, _ = w.ActiveFile.WriteString(string(d) + "\n")

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
func OpenCacheFile(baseDir string, filePrefix string) (string, *os.File, error) {
	p := NameCacheFile(baseDir, filePrefix, OutputSuffix)
	var f *os.File

	// Create the output directory as needed
	merr := os.MkdirAll(baseDir, 0750)
	if merr != nil {
		log.Error(merr)
		return p, f, merr
	}

	// Create/open the file at path 'p' as needed
        f, err := os.OpenFile(p, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0640)
	if err != nil {
		log.Error(err)
		log.WithError(err).Fatal("failed to open cache file")
	}
        return p, f, err
}

func RunFsCache(cacher *FsCacheWriter) {
	// Create the output directory as needed
	merr := os.MkdirAll(cacher.BaseDir, 0750)
	if merr != nil {
		log.Error(merr)
	}

	// Use a ticker to trigger rotation of the active cache file every 60 seconds
	ticker := time.NewTicker(60 * time.Second)
	quit := make(chan struct{})
	go func() {
		for {
			select {
			case <- ticker.C:
				oldCacheFile := cacher.ActiveFile
				cachePath, cacheFile, err := OpenCacheFile(cacher.BaseDir, cacher.FilePrefix)
				if err != nil {
					log.Error(err)
				} else if cacher.ActivePath != cachePath {
					cacher.ActivePath = cachePath
					cacher.ActiveFile = cacheFile
					// Run a goroutine to close the old file and cleanup the cache files
					go CleanFsCache(cacher, oldCacheFile)
				}
			case <- quit:
				ticker.Stop()
				return
		        }
		}
	}()
}

func CleanFsCache(cacher *FsCacheWriter, oldFile *os.File) {
	// Close the old cache file
	oldFile.Close()
	// Clean the filesystem cache by removing empty and/or old files
	files, err := ioutil.ReadDir(cacher.BaseDir)
	if err != nil {
		log.Error(err)
	}
	for _, file := range files {
		filePath := path.Join([]string{cacher.BaseDir, file.Name()}...)
		// Avoid archiving data which is either (1) active OR (2) already archived
		if filePath != cacher.ActivePath && strings.HasSuffix(filePath, OutputSuffix) {
			log.Info("checking cache file for archive and/or cleanup : " + filePath)
			fileInfo, ferr := os.Stat(filePath)
			if ferr != nil {
				log.Error(ferr)
			}
			switch mode := fileInfo.Mode(); {
			case mode.IsDir():
				log.Info("skipping removal of directory : " + filePath)
			case mode.IsRegular():
				if fileInfo.Size() == 0 {
					log.Info("removing empty cache file : " + filePath)
					rerr := os.RemoveAll(filePath)
					if rerr != nil {
						log.Error(rerr)
					}
				} else {
					log.Info("archiving cache file : " + filePath)
					a_err := ArchiveCacheFile(cacher, filePath, fileInfo)
					if a_err != nil {
						log.Error(a_err)
					}
				}
			}
		}
	}
}

func ArchiveCacheFile(cacher *FsCacheWriter, filepath string, fileinfo os.FileInfo) error {
	// Generate the filepath for the archive file, based on the current time.Now()
	acf_name := NameCacheFile(cacher.BaseDir, cacher.FilePrefix, ArchiveSuffix)
	acf, err := os.OpenFile(acf_name, os.O_RDWR|os.O_CREATE, 0640)
	defer acf.Close()
	if err != nil { return err }

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

	// Delete old file after adding its contents to archive
	log.Info("removing cache file after adding to archive : " + filepath)
	err = os.RemoveAll(filepath)
	if err != nil { return err }

	return nil
}
