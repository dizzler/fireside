package processors

import (
    "context"

    "fireside/pkg/configure"
    "fireside/pkg/opa/policy"
    "fireside/pkg/pipeline/data"

    log "github.com/sirupsen/logrus"
)

// EventTagger is a wrapper around *policy.Tagger
type EventTagger struct {
    T *policy.Tagger
}

// NewEventTagger instantiates a new EventTagger object
func NewEventTagger(config *configure.TaggerPolicyConfig) *EventTagger {
    log.Trace("running NewEventTagger function")

    ctx := context.Background()
    t := policy.NewTagger(config, ctx)
    return &EventTagger{T: t}
}

// processes data from a given event, adding tags for matches, misses and issues;
// method required by processor (conduit) interface
func (et *EventTagger) ProcessData(d data.JSON, outputChan chan data.JSON, killChan chan error) {
    log.Trace("running ProcessData method for type EventTagger")

    dx, err := et.T.EvaluateQueries(d)
    if err != nil {
        killChan <- err
    } else {
        outputChan <- dx
    }
}

// method required by processor (conduit) interface
func (et *EventTagger) Finish(outputChan chan data.JSON, killChan chan error) {
    log.Trace("running Finish method for type EventTagger")
}

// method required by processor (conduit) interface
func (et *EventTagger) String() string {
    return "EventTagger"
}
