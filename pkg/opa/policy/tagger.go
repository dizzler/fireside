package policy

import (
    "context"
    "encoding/json"
    "errors"

    "fireside/pkg/configure"
    "fireside/pkg/opa/query"
    "fireside/pkg/pipeline/data"

    log "github.com/sirupsen/logrus"
)

type Tagger struct {
    // configuration for the instantiated Tagger object
    Config        *configure.TaggerPolicyConfig

    // Context for the Tagger
    Ctx           context.Context

    // list of tagger queries to run in a separate goroutine;
    // all queries operate on the same EventType
    TaggerQueries []*query.TaggerQuery
}

func NewTagger(config *configure.TaggerPolicyConfig, ctx context.Context) *Tagger {
    log.Trace("running NewTagger function for Tagger type")
    // add tags for queries returning error and/or undefined status
    if config.TagsErrorField == "" {
        config.TagsErrorField = configure.DefaultTagErrorField
    }
    // add tags for queries that evaluate to boolean false
    if config.TagsFalseField == "" {
        config.TagsFalseField = configure.DefaultTagFalseField
    }
    // add tags for queries that evaluate to boolean true
    if config.TagsTrueField == "" {
        config.TagsTrueField = configure.DefaultTagTrueField
    }

    // populate the list of taggerQueries from configs
    var taggerQueries []*query.TaggerQuery
    for _, taggerQuery := range config.QueryConfigs {
        taggerQueries = append(taggerQueries, query.NewTaggerQuery(&taggerQuery))
    }

    // return a pointer to the Tagger struct
    return &Tagger{
        Config: config,
        Ctx: ctx,
        TaggerQueries: taggerQueries,
    }
}

// evaluate the PreparedQuery for each query.TaggerQuery;
// this method must be run after PrepareQueries() method
func (t *Tagger) EvaluatePreparedQueries(in data.JSON, outputChan chan data.JSON, killChan chan error) {
    log.Trace("running EvaluatePreparedQueries method on Tagger type")
    var (
        oerr      error
        out       data.JSON
        tagsError []string
        tagsFalse []string
        tagsTrue  []string
        useNum    bool
    )
    // decode the source JSON prior to evaluating the data with OPA
    dd, err := data.DecodeJSON(in, useNum)
    if err != nil {
        killChan <- err
    }

    // extract the top-level `data` field into a unique JSON object;
    // allows OPA to evaluate the event data, not the event wrapper
    eventData := dd["data"]
    // loop through the list of tagger queries to evaluate
    for _, tq := range t.TaggerQueries {
        // evaluate the query
        eTags, fTags, tTags := tq.PreparedQueryEval(eventData)

        // append tags to appropriate lists; worry about dedup later
        for _, eTag := range eTags {
            tagsError = append(tagsError, eTag)
        }
        for _, fTag := range fTags {
            tagsFalse = append(tagsFalse, fTag)
        }
        for _, tTag := range tTags {
            tagsTrue = append(tagsTrue, tTag)
        }
    }

    // insert each non-empty, de-duplicated list of tags in the appropriate JSON field
    if len(tagsError) > 0 {
        field := t.Config.TagsErrorField
        if _, fieldExists := dd[field]; fieldExists {
            oerr = errors.New("cannot insert tags into existing field = " + field)
            killChan <- oerr
        }
        // insert the list of tags into specified field
        dd[field] = data.DedupStringSlice(tagsError)
        log.Debug("inserted list of tags into field = " + field)
    }
    if len(tagsFalse) > 0 {
        field := t.Config.TagsFalseField
        if _, fieldExists := dd[field]; fieldExists {
            oerr = errors.New("cannot insert tags into existing field = " + field)
            killChan <- oerr
        }
        // insert the list of tags into specified field
        dd[field] = data.DedupStringSlice(tagsFalse)
        log.Debug("inserted list of tags into field = " + field)
    }
    if len(tagsTrue) > 0 {
        field := t.Config.TagsTrueField
        if _, fieldExists := dd[field]; fieldExists {
            oerr = errors.New("cannot insert tags into existing field = " + field)
            killChan <- oerr
        }
        // insert the list of tags into specified field
        dd[field] = data.DedupStringSlice(tagsTrue)
        log.Debug("inserted list of tags into field = " + field)
    }

    // marshal the enriched event data back to JSON bytes
    out, oerr = json.Marshal(dd)

    if oerr != nil {
        // send errors to the kill channel for the pipeline
        killChan <- oerr
    } else {
        // send the completed JSON document to the output channel
        outputChan <- out
    }
}

// prepares the list of TaggerQueries for evaluation;
// this method must be run before EvaluatePreparedQueries() method
func (t *Tagger) PrepareQueries() (preperr error) {
    log.Trace("running PrepareQueries method on Tagger type")
    for _, tq := range t.TaggerQueries {
        if preperr = tq.PrepareQuery(t.Ctx); preperr != nil {
            return
        }
    }
    return
}
