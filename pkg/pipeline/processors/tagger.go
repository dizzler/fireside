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

    // set the context for the Tagger
    ctx := context.Background()

    // create the new Tagger object
    t := policy.NewTagger(config, ctx)

    // prepare queries for evaluation via Tagger method call
    if err := t.PrepareQueries(); err != nil {
        log.WithError(err).Fatal("failed to PrepareQueries for type *Tagger")
    }

    // return a pointer to the EventTagger struct
    return &EventTagger{T: t}
}

// processes data from a given event, adding tags for matches, misses and issues;
// method required by processor (conduit) interface
func (et *EventTagger) ProcessData(d data.JSON, outputChan chan data.JSON, killChan chan error) {
    log.Trace("running ProcessData method for type EventTagger")

    var (
        dx  data.JSON
        err error
    )

    dx, err = et.T.EvaluatePreparedQueries(d)

    if err != nil {
        // send the error to the kill channel
        killChan <- err
    } else {
        // send the enriched event to the output channel
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
