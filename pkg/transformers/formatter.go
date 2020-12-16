package transformers

import (
    "github.com/dailyburn/ratchet/data"
    kz "gopkg.in/qntfy/kazaam.v3"
    log "github.com/sirupsen/logrus"
)

// Formatter is a struct that wraps the required inputs for a 'New' instance of kazaam
// kazaam ia a library for arbitrary reformatting of JSON data
//   https://github.com/qntfy/kazaam
type Formatter struct {
    Kz  *kz.Kazaam
}

// NewFormatter instantiates a new instance of Formatter
func NewFormatter(spec string) *Formatter {
    k, _ := kz.New(spec, kz.NewDefaultConfig())
    return &Formatter{Kz: k}
}

// ProcessData converts input JSON to a standard format / schema
func (k *Formatter) ProcessData(d data.JSON, outputChan chan data.JSON, killChan chan error) {
    dx, err := k.Kz.Transform(d)
    if err != nil {
        log.Error(err)
    }
    outputChan <- dx
}

// Finish - see interface for documentation.
func (k *Formatter) Finish(outputChan chan data.JSON, killChan chan error) {
}

func (k *Formatter) String() string {
    return "Formatter"
}
