package fireside

import (
    "bufio"
    //"compress/gzip"
    "io"
    "os"

    "github.com/dailyburn/ratchet/data"
    "github.com/dailyburn/ratchet/util"

    configure "fireside/pkg/configure"
    log "github.com/sirupsen/logrus"
)

// FileReader wraps an io.Reader and reads it.
type FileReader struct {
    ActiveBuf  *bufio.Writer
    ActiveFile *os.File
    Config     *configure.InputConfigFileReader
}

// NewFileReader returns a new FileReader wrapping the given io.Reader object.
func NewFileReader(config *configure.InputConfigFileReader) *FileReader {
    // open a file in order to create a reader
    f, err := os.Open(config.Src.Path)
    if err != nil {
        log.WithError(err).Fatal("failed to open file for reading : " + config.Src.Path)
    } else {
	log.Infof("reading input events of type=%s from file at path=%s", config.Event.Type, config.Src.Path)
    }
    return &FileReader{
        ActiveFile: f,
        Config: config,
    }
}

// ProcessData overwrites the reader if the content is Gzipped, then defers to ForEachData
func (r *FileReader) ProcessData(d data.JSON, outputChan chan data.JSON, killChan chan error) {
    /*
    if r.Config.Gzipped {
	gzReader, err := gzip.NewReader(r.ActiveFile)
	util.KillPipelineIfErr(err, killChan)
	r.ActiveFile = gzReader
    }
    */
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
    // read the data from the active file reader
    r.ForEachData(killChan, func(d data.JSON) {
	// create a JSON event wrapper to provide a common format for dowstream event processing
	eventWrapper := configure.NewFiresideEvent(
	    r.Config.Event.Category,
	    r.Config.Event.Type,
	    logID,
	    r.Config.Src.Path,
	    r.Config.Src.Type)
	// insert the JSON data into a field within the event wrapper JSON
	eventJson, err := configure.InsertFiresideEventData(d, eventWrapper)
	if err != nil {
	    log.WithError(err).Error(
		"failed to marshal Fireside Event JSON for event source Path = %s : Type = %s",
	        r.Config.Src.Path,
	        r.Config.Src.Type)
	}
	// send the wrapped JSON event to the output channel
	outputChan <- eventJson
    })
}

// Finish - see interface for documentation.
func (r *FileReader) Finish(outputChan chan data.JSON, killChan chan error) {
}

// ForEachData either reads by line or by buffered stream, sending the data
// back to the anonymous func that ultimately shoves it onto the outputChan
func (r *FileReader) ForEachData(killChan chan error, foo func(d data.JSON)) {
    if r.Config.Line.ByLine {
	r.scanLines(killChan, foo)
    } else {
	r.bufferedRead(killChan, foo)
    }
}

func (r *FileReader) scanLines(killChan chan error, forEach func(d data.JSON)) {
    scanner := bufio.NewScanner(r.ActiveFile)
    for scanner.Scan() {
	forEach(data.JSON(scanner.Text()))
    }
    err := scanner.Err()
    util.KillPipelineIfErr(err, killChan)
}

func (r *FileReader) bufferedRead(killChan chan error, forEach func(d data.JSON)) {
    reader := bufio.NewReader(r.ActiveFile)
    d := make([]byte, r.Config.BufferSize)
    for {
	n, err := reader.Read(d)
	if err != nil && err != io.EOF {
	    killChan <- err
	}
	if n == 0 {
	    break
	}
	forEach(d)
    }
}

func (r *FileReader) String() string {
    return "FileReader"
}
