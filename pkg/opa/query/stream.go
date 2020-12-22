package query

import (
    "context"
    "encoding/json"
    "os"

    "fireside/pkg/configure"

    log "github.com/sirupsen/logrus"

    "github.com/open-policy-agent/opa/rego"
    "github.com/open-policy-agent/opa/storage"
    "github.com/open-policy-agent/opa/storage/inmem"
    "github.com/open-policy-agent/opa/topdown"
)

type QueryStream struct {
    // configuration for the instantiated QueryStream object
    Config        *configure.QueryStreamConfig

    // PreparedQuery is set when PrepareQuery() method is invoked
    PreparedQuery rego.PreparedEvalQuery

    // storage layer to prepare and use
    Store         storage.Store
}

// creates a new QueryStream from config and provides a wrapper for a PreparedQuery
func NewQueryStream(config *configure.QueryStreamConfig) *QueryStream {
    log.Trace("running NewQueryStream function")
    // return a pointer to a QueryStream struct
    return &QueryStream{
        Config: config,
    }
}

// prepares an OPA query for performant eval at a later stage
func (tq *QueryStream) PrepareQuery(ctx context.Context) error {
    log.Trace("running PrepareQuery method on type QueryStream")

    // create the empty storage.Store layer in memory
    tq.Store = inmem.New()

    // Open a write transaction on the store that will perform write operations.
    txn, err := tq.Store.NewTransaction(ctx, storage.WriteParams)
    if err != nil {
        return err
    }

    // prepare the rego query, resulting in better performance plus the ability
    // to share the prepared query across an arbitrary number of goroutines
    query, qerr := rego.New(
	rego.Query(tq.Config.QueryString),
        rego.LoadBundle(tq.Config.BundlePath),
        rego.Package(tq.Config.Package),
        rego.Store(tq.Store),
        rego.Transaction(txn),
	).PrepareForEval(ctx)
    if qerr != nil {
        return qerr
    }

    if cerr := tq.Store.Commit(ctx, txn); cerr != nil {
        return cerr
    }

    tq.PreparedQuery = query

    return nil
}

// evaluates the prepared query
func (tq *QueryStream) PreparedQueryEval(input interface{}) (tagsE []string, tagsF []string, tagsT []string) {
    log.Trace("running PreparedQueryEval method on type QueryStream")

    // append the QueryID to each list of possible tags
    tagsIssue := append(tq.Config.TagsIssue, tq.Config.QueryID)
    tagsMatch := append(tq.Config.TagsMatch, tq.Config.QueryID)
    tagsMiss  := append(tq.Config.TagsMiss, tq.Config.QueryID)

    // set the context for query eval
    ctx := context.Background()

    // set the list of EvalOptions before evaluating prepared query
    var options []rego.EvalOption
    options = append(options, rego.EvalInput(input))
    // DEBUG logging
    inJson, _ := json.Marshal(input)
    log.Trace("debugging input interface{} document as JSON... " + string(inJson))

    // enable query tracing
    var buf *topdown.BufferTracer
    if tq.Config.TraceQuery {
        log.Info("enabling trace for OPA query")
        buf = topdown.NewBufferTracer()
        options = append(options, rego.EvalQueryTracer(topdown.QueryTracer(buf)))
    }

    /*
    // DEBUG : list the modules from prepared query state
    mods := tq.PreparedQuery.Modules()
    modsJson, _ := json.Marshal(mods)
    log.Debug("listing prepared query modules... " + string(modsJson))
    */

    // Create a new transaction for eval of prepared query
    var readParams = storage.TransactionParams{Write: false}
    txn, err := tq.Store.NewTransaction(ctx, readParams)
    if err != nil {
        log.WithError(err).Fatal("failed to get new transaction for query eval")
    }
    options = append(options, rego.EvalTransaction(txn))

    // evaluate the (previously) prepared query with eval "options"
    results, err := tq.PreparedQuery.Eval(ctx, options...)
    if err != nil {
        log.WithError(err).Error("rego (OPA) query returned error : " + tq.Config.QueryString)
        tagsE = tagsIssue
        return
    } else {
        // if tracing is enabled, pretty print the query trace
        if tq.Config.TraceQuery {
            log.Info("printing trace for OPA query")
            topdown.PrettyTraceWithLocation(os.Stdout, *buf)
            resultsJson, _ := json.Marshal(results)
            log.Info("rego (OPA) query returned one result to process... " + string(resultsJson))
        }

        if len(results) == 0 {
            log.Error("rego (OPA) query returned undefined result for query:  " + tq.Config.QueryString)
            tagsE = tagsIssue
            return
        } else if len(results) > 1 {
            log.WithError(err).Warning("rego (OPA) query returned more results than expected ; processing first query result ONLY : " + tq.Config.QueryString)
            tagsE = tagsIssue
        } else {
            log.Trace("rego (OPA) query returned one result to process")
        }
    }

    // we evaluate a single document, so we should only have one result;
    // assert that the result type boolean
    if result, ok := results[0].Bindings[tq.Config.BindingName].(bool); ok {
        // trace logging
        log.Tracef("bindings.%s: %t", tq.Config.BindingName, result)

        // set tags based on true||false value of binding var
        if result {
            tagsT = tagsMatch
            log.Debugf("found a match for target result : %s = %t", tq.Config.BindingName, result)
        } else {
            tagsF = tagsMiss
            log.Debugf("no match; binding %s = %t", tq.Config.BindingName, result)
        }
    } else {
        log.WithError(err).Warning("unexpected (non-bool) type returned from rego (OPA) query : " + tq.Config.QueryString)
        tagsE = tagsIssue
    }

    return
}
