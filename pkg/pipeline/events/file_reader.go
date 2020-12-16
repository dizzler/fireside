package events

import (
    "bufio"
    "io"
    "os"
    "time"

    "fireside/pkg/configure"
    "fireside/pkg/pipeline/data"

    log "github.com/sirupsen/logrus"
)

// FileReader provides a config wrapper around a specific type
type FileReader struct {
    Config *configure.InputConfigFileReader
    Tail   *TailReader
}

// NewFileReader returns a new FileReader wrapping the given io.Reader object.
func NewFileReader(config *configure.InputConfigFileReader) *FileReader {
    // open a file in order to create a reader
    tf, err := os.Open(config.Src.Path)
    if err != nil {
        log.WithError(err).Fatal("failed to open file for reading : " + config.Src.Path)
    }
    log.Infof("reading input events of type=%s from file at path=%s", config.Event.Type, config.Src.Path)

    dataChan := make(chan []byte)
    errChan := make(chan error)
    // create a read buffer for in-memory storage of events read from file
    br := bufio.NewReader(tf)
    tr := &TailReader{
        Buf: br,
        DataChan: dataChan,
        ErrChan: errChan,
        File: tf,
        Path: config.Src.Path,
    }
    if err := tr.setFileSize(); err != nil {
        log.WithError(err).Fatal("failed to get/set size for file : " + tr.Path)
    }
    // tail the active file (buffer) for new data / lines
    go tr.tailFile()

    // return a pointer to the new FileReader struct
    return &FileReader{
        Config: config,
        Tail: tr,
    }
}

// ProcessData tails lines from a file, processes the data as JSON,
// then sends the FiresideEvent JSON to the output channel
func (r *FileReader) ProcessData(d data.JSON, outputChan chan data.JSON, killChan chan error) {
    log.Debug("running ProcessData method for FileReader")
    var (
        logID string
        rerr   error
    )
    if r.Config.Src.ID != "" {
        logID = r.Config.Src.ID
    } else if r.Config.Name != ""{
        logID = r.Config.Name
    } else {
        logID, rerr = os.Hostname()
        if rerr != nil {
            log.WithError(rerr).Error("failed to get a value for src.id ; failed to get hostname")
            logID = ""
        }
    }
    // read raw event data from one channel, then format as FiresideEvent JSON and send to outputChan
    for byteData := range r.Tail.DataChan {
        // create a JSON event wrapper to provide a common format for downstream event processing
        eventWrapper := configure.NewFiresideEvent(
            r.Config.Event.Category,
            r.Config.Event.Type,
            logID,
            r.Config.Src.Path,
            r.Config.Src.Type)
        // insert the JSON data into a field within the event wrapper JSON
        eventJson, err := configure.InsertFiresideEventData(byteData, eventWrapper)
        if err != nil {
            log.WithError(err).Errorf(
                "failed to create Fireside Event JSON for event source Path = %s : Type = %s",
                r.Config.Src.Path,
                r.Config.Src.Type)
        }
        // send the wrapped JSON event to the output channel
        outputChan <- eventJson
    }
}

func (r *FileReader) Finish(outputChan chan data.JSON, killChan chan error) {
    for err := range r.Tail.ErrChan {
        killChan <- err
    }
}

func (r *FileReader) String() string {
    return "FileReader"
}

// TailReader is a sub-type of FileReader, containing data needed for
// tailing JSON events from a file
type TailReader struct {
    Buf      *bufio.Reader
    DataChan chan []byte
    ErrChan  chan error
    File     *os.File
    Path     string
    Size     int64
}

// getFileSize returns the FileInfo from os.stat() for the File
func (tr *TailReader) getFileSize() (int64, error) {
    var sz int64
    info, err := tr.File.Stat()
    if err != nil {
        return sz, err
    }
    return info.Size(), nil
}

// setFileSize saves the updated "Size" of the File
func (tr *TailReader) setFileSize() error {
    fsize, err := tr.getFileSize()
    if err != nil {
        return err
    }
    tr.Size = fsize
    return nil
}

// tailFile reads new lines from the end of the File for the TailReader
func (tr *TailReader) tailFile() {
    for {
        for line, prefix, err := tr.Buf.ReadLine(); err != io.EOF; line, prefix, err = tr.Buf.ReadLine() {
            if prefix {
                log.Warning("read line contains a prefix ; cannot convert to JSON : " + string(line))
            } else {
                tr.DataChan <- line
            }
        }
        pos, err := tr.File.Seek(0, io.SeekCurrent)
        if err != nil {
            tr.ErrChan <- err
        }
        for {
            time.Sleep(time.Second)
            newSize, err := tr.getFileSize()
            if err != nil {
                tr.ErrChan <- err
            }
            if newSize != tr.Size {
                if newSize < tr.Size {
                    tr.File.Seek(0, 0)
                } else {
                    tr.File.Seek(pos, io.SeekStart)
                }
                tr.Buf = bufio.NewReader(tr.File)
                tr.Size = newSize
                break
            }
        }
    }
}
