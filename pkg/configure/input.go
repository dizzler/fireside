package fireside

import (
    "os"

    log "github.com/sirupsen/logrus"
)

// InputConfig is a struct to store configuration for pipeline output processors
type InputConfig struct {
    BufferSize    int
    EventCategory string
    EventType     string
    Gzipped      bool
    LineByLine    bool
    SrcID         string
    SrcPath       string
    SrcType       string
}

func NewInputConfig(bufferSize int, eventCat string, eventType string, gzbool bool,linebool bool,
                    srcID string, srcPath string, srcType string) *InputConfig {
    if srcID == "" {
        hostnm, err := os.Hostname()
        if err != nil {
            log.Error("failed to set SrcID for InputConfig")
        }
        if len(hostnm) > 0 {
            srcID = hostnm
        }
    }
    return &InputConfig{
        BufferSize: bufferSize,
        EventCategory: eventCat,
        EventType: eventType,
        Gzipped: gzbool,
        LineByLine: linebool,
        SrcID: srcID,
        SrcPath: srcPath,
        SrcType: srcType,
    }
}
