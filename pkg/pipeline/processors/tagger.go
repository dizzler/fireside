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
    ConcurrentProcs int
    T               *policy.Tagger
    PipelineState   *configure.PipelineConfigState
}

// NewEventTagger instantiates a new EventTagger object
func NewEventTagger(config *configure.TaggerPolicyConfig, pipestate *configure.PipelineConfigState) *EventTagger {
    log.Trace("running NewEventTagger function")

    // set the context for the Tagger
    ctx := context.Background()

    // create the new Tagger object
    t := policy.NewTagger(config, ctx)

    // default concurrency to the value of configure.DefaultConcurrencyEventTagger
    var conc int = configure.DefaultConcurrencyEventTagger
    if config.Concurrency > 0 {
        conc = config.Concurrency
    } else if pipestate.DefaultConcurrency > 0 {
        conc = pipestate.DefaultConcurrency
    }

    // prepare queries for evaluation via Tagger method call
    if err := t.PrepareQueries(); err != nil {
        log.WithError(err).Fatal("failed to PrepareQueries for type *Tagger")
    }

    // return a pointer to the EventTagger struct
    return &EventTagger{
        ConcurrentProcs: conc,
        T: t,
        PipelineState: pipestate,
    }
}

// implements the Concurrency() method of the ConcurrentDataProcessor interface
func (et *EventTagger) Concurrency() int {
    return et.ConcurrentProcs
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
