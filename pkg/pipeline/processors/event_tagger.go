package processors

import (
    "context"

    "fireside/pkg/configure"
    "fireside/pkg/opa/policy"
    "fireside/pkg/pipeline/data"

    log "github.com/sirupsen/logrus"
)

// EventTagger is a wrapper around *policy.QueryStream
type EventTagger struct {
    ConcurrentProcs int
    PipelineState   *configure.PipelineConfigState
    QS              *policy.QueryStream
}

// NewEventTagger instantiates a new EventTagger object
func NewEventTagger(config *configure.QueryStreamPolicyConfig, pipestate *configure.PipelineConfigState) *EventTagger {
    log.Trace("running NewEventTagger function")

    // set the context for the QueryStream
    ctx := context.Background()

    // create the new QueryStream object
    qs := policy.NewQueryStream(config, ctx)

    // default concurrency to the value of configure.DefaultConcurrencyEventTagger
    var conc int = configure.DefaultConcurrencyEventTagger
    if config.Concurrency > 0 {
        conc = config.Concurrency
    } else if pipestate.DefaultConcurrency > 0 {
        conc = pipestate.DefaultConcurrency
    }

    // prepare queries for evaluation via QueryStream method call
    if err := qs.PrepareQueries(); err != nil {
        log.WithError(err).Fatal("failed to PrepareQueries for type *QueryStream")
    }

    // return a pointer to the EventTagger struct
    return &EventTagger{
        ConcurrentProcs: conc,
        PipelineState: pipestate,
        QS: qs,
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

    // evaluate the stream of (all) queries for the input document (JSON)
    et.QS.EvaluatePreparedQueries(d, configure.EventDataField, outputChan, killChan)
}

// method required by processor (conduit) interface
func (et *EventTagger) Finish(outputChan chan data.JSON, killChan chan error) {
    log.Trace("running Finish method for type EventTagger")
}

// method required by processor (conduit) interface
func (et *EventTagger) String() string {
    return "EventTagger"
}
