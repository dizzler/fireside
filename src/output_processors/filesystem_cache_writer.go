package output_processors

import (
	"os"
	"time"

	"github.com/dailyburn/ratchet/data"
	"github.com/dailyburn/ratchet/logger"
	"github.com/dailyburn/ratchet/util"
)

// FsCacheWriter wraps any io.Writer object.
// It can be used to write data out to a File, os.Stdout, or
// any other task that can be supported via io.Writer.
type FsCacheWriter struct {
	OutputDir    string
	OutputPrefix string
}

// NewFsCacheWriter returns a new FsCacheWriter wrapping the given io.Writer object
func NewFsCacheWriter(outdir string, prefix string) *FsCacheWriter {
	return &FsCacheWriter{OutputDir: outdir, OutputPrefix: prefix}
}

// ProcessData writes the data
func (w *FsCacheWriter) ProcessData(d data.JSON, outputChan chan data.JSON, killChan chan error) {
	var (
	        bytesWritten int
		outfile      string
		outpath      string
	)
	const layout = "2006-01-02MST15-04"
	now := time.Now()
	outfile = "fireside-event-cache-" + now.Format(layout) + ".out"
	outpath = w.OutputDir + "/" + outfile
	// Create the output directory as needed
	merr := os.MkdirAll(w.OutputDir, 0750)
	util.KillPipelineIfErr(merr, killChan)

	// Create/Open the file at path 'outpath' as needed
        writer, ferr := os.OpenFile(outpath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0640)
        //writer, ferr := os.OpenFile(outpath, os.O_RDWR|os.O_CREATE, 0660)
	util.KillPipelineIfErr(ferr, killChan)
	defer writer.Close()

	// Append the string to the output file for the current minute
	bytesWritten, ferr = writer.WriteString(string(d) + "\n")

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
