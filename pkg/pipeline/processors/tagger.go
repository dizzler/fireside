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

    // default concurrency to the value of configure.DefaultConcurrencyEventTagger
    if t.Config.Concurrency == 0 {
        t.Config.Concurrency = configure.DefaultConcurrencyEventTagger
    }

    // prepare queries for evaluation via Tagger method call
    if err := t.PrepareQueries(); err != nil {
        log.WithError(err).Fatal("failed to PrepareQueries for type *Tagger")
    }

    // return a pointer to the EventTagger struct
    return &EventTagger{T: t}
}

// implements the Concurrency() method of the ConcurrentDataProcessor interface
func (et *EventTagger) Concurrency() int {
    return et.T.Config.Concurrency
}

// processes data from a given event, adding tags for matches, misses and issues;
// method required by processor (conduit) interface
func (et *EventTagger) ProcessData(d data.JSON, outputChan chan data.JSON, killChan chan error) {
    log.Trace("running ProcessData method for type EventTagger")

    // evaluate all tagger queries for the input document (JSON)
    et.T.EvaluatePreparedQueries(d, outputChan, killChan)
}

// method required by processor (conduit) interface
func (et *EventTagger) Finish(outputChan chan data.JSON, killChan chan error) {
    log.Trace("running Finish method for type EventTagger")
}

// method required by processor (conduit) interface
func (et *EventTagger) String() string {
    return "EventTagger"
}
