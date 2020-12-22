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

type QueryStream struct {
    // configuration for the instantiated QueryStream object
    Config        *configure.QueryStreamPolicyConfig

    // Context for the QueryStream
    Ctx           context.Context

    // list of queries to run in a separate goroutine;
    // all queries operate on the same EventType
    StreamQueries []*query.QueryStream
}

func NewQueryStream(config *configure.QueryStreamPolicyConfig, ctx context.Context) *QueryStream {
    log.Trace("running NewQueryStream function for QueryStream type")
    // add tags for queries returning error and/or undefined status
    if config.TagsIssueField == "" {
        config.TagsIssueField = configure.DefaultTagIssueField
    }
    // add tags for queries that evaluate to boolean false
    if config.TagsMissField == "" {
        config.TagsMissField = configure.DefaultTagMissField
    }
    // add tags for queries that evaluate to boolean true
    if config.TagsMatchField == "" {
        config.TagsMatchField = configure.DefaultTagMatchField
    }

    // populate the list of streamQueries from configs
    var streamQueries []*query.QueryStream
    for _, streamQuery := range config.QueryConfigs {
        streamQueries = append(streamQueries, query.NewQueryStream(&streamQuery))
    }

    // return a pointer to the QueryStream struct
    return &QueryStream{
        Config: config,
        Ctx: ctx,
        StreamQueries: streamQueries,
    }
}

// evaluate the PreparedQuery for each query.QueryStream;
// this method must be run after PrepareQueries() method
func (qs *QueryStream) EvaluatePreparedQueries(in data.JSON, extractField string, outputChan chan data.JSON, killChan chan error) {
    log.Trace("running EvaluatePreparedQueries method on QueryStream type")

    if len(qs.StreamQueries) == 0 {
        killChan <- errors.New("cannot evaluate an empty list of StreamQueries")
    }

    var (
        docData   interface{}
	docMap    map[string]interface{}
        oerr      error
        out       data.JSON
        tagsIssue []string
        tagsMiss  []string
        tagsMatch []string
        useNum    bool
    )

    // decode the source JSON into a map of keys; each key is a string
    // representing a top-level field in the source document; the value
    // for each (string) key is a go interface{} which can (optionall)
    // be used for OPA query evaluation
    docMap, oerr = data.DecodeJsonAsMap(in, useNum)
    if oerr != nil {
        killChan <- oerr
    }

    if extractField == configure.EventDataField {
        // extract the top-level `data` field into a unique JSON object;
        // allows OPA to evaluate the event data, not the event wrapper
        docData = docMap[extractField]
        log.Trace("extracted decoded data from field : " + extractField)
    } else {
        log.Trace("decoding entire source document as prep for query eval")
        // decode the source JSON prior to evaluating the entire document/object with OPA;
        // pass the entire document as an interface{} for OPA query evaluation
        docData, oerr = data.DecodeJsonAsInterface(in, useNum)
        if oerr != nil {
            killChan <- oerr
        }
    }

    // loop through the list (i.e. "stream") of queries to evaluate
    for _, tq := range qs.StreamQueries {
        // evaluate the query
        eTags, fTags, tTags := tq.PreparedQueryEval(docData)

        // append tags to appropriate lists; worry about dedup later
        for _, eTag := range eTags {
            tagsIssue = append(tagsIssue, eTag)
        }
        for _, fTag := range fTags {
            tagsMiss = append(tagsMiss, fTag)
        }
        for _, tTag := range tTags {
            tagsMatch = append(tagsMatch, tTag)
        }
    }

    // insert each non-empty, de-duplicated list of tags in the appropriate JSON field
    field := qs.Config.TagsIssueField
    if len(tagsIssue) > 0 {
        if _, fieldExists := docMap[field]; fieldExists {
            oerr = errors.New("cannot insert tags into existing field = " + field)
            killChan <- oerr
        }
        // insert the list of tags into specified field
        docMap[field] = data.DedupStringSlice(tagsIssue)
        log.Trace("inserted list of tags into field = " + field)
    } else {
        log.Trace("no tags to insert for field = " + field)
    }
    field = qs.Config.TagsMatchField
    if len(tagsMatch) > 0 {
        if _, fieldExists := docMap[field]; fieldExists {
            oerr = errors.New("cannot insert tags into existing field = " + field)
            killChan <- oerr
        }
        // insert the list of tags into specified field
        docMap[field] = data.DedupStringSlice(tagsMatch)
        log.Trace("inserted list of tags into field = " + field)
    } else {
        log.Trace("no tags to insert for field = " + field)
    }
    field = qs.Config.TagsMissField
    if len(tagsMiss) > 0 {
        if _, fieldExists := docMap[field]; fieldExists {
            oerr = errors.New("cannot insert tags into existing field = " + field)
            killChan <- oerr
        }
        // insert the list of tags into specified field
        docMap[field] = data.DedupStringSlice(tagsMiss)
        log.Trace("inserted list of tags into field = " + field)
    } else {
        log.Trace("no tags to insert for field = " + field)
    }

    // marshal the enriched event data back to JSON bytes
    out, oerr = json.Marshal(docMap)

    if oerr != nil {
        // send errors to the kill channel for the pipeline
        killChan <- oerr
    } else {
        // send the completed JSON document to the output channel
        outputChan <- out
    }
}

// prepares the list of StreamQueries for evaluation;
// this method must be run before EvaluatePreparedQueries() method
func (qs *QueryStream) PrepareQueries() (preperr error) {
    log.Trace("running PrepareQueries method on QueryStream type")

    if len(qs.StreamQueries) == 0 {
        preperr = errors.New("cannot prepare an empty list of StreamQueries")
        return
    }

    for _, tq := range qs.StreamQueries {
        if preperr = tq.PrepareQuery(qs.Ctx); preperr != nil {
            return
        }
    }
    return
}
