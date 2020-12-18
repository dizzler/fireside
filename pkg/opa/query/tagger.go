package query

import (
    "context"

    "fireside/pkg/configure"

    log "github.com/sirupsen/logrus"

    "github.com/open-policy-agent/opa/rego"
    "github.com/open-policy-agent/opa/storage"
    "github.com/open-policy-agent/opa/storage/inmem"
)

type TaggerQuery struct {
    // configuration for the instantiated TaggerQuery object
    Config        *configure.TaggerQueryConfig

    // PreparedQuery is set when PrepareQuery() method is invoked
    PreparedQuery rego.PreparedEvalQuery
}

// creates a new TaggerQuery from config and provides a wrapper for a PreparedQuery
func NewTaggerQuery(config *configure.TaggerQueryConfig) *TaggerQuery {
    log.Trace("running NewTaggerQuery function")
    // return a pointer to a TaggerQuery struct
    return &TaggerQuery{
        Config: config,
    }
}

// prepares an OPA query for performant eval at a later stage
func (tq *TaggerQuery) PrepareQuery(ctx context.Context) error {
    log.Trace("running PrepareQuery method on type TaggerQuery")

    // create the empty storage.Store layer in memory
    store := inmem.New()

    // Open a write transaction on the store that will perform write operations.
    txn, err := store.NewTransaction(ctx, storage.WriteParams)
    if err != nil {
        return err
    }

    // prepare the rego query, resulting in better performance plus the ability
    // to share the prepared query across an arbitrary number of goroutines
    query, qerr := rego.New(
	rego.Query(tq.Config.QueryString),
        rego.LoadBundle(tq.Config.BundlePath),
	rego.Package(tq.Config.Package),
        rego.Store(store),
        rego.Transaction(txn),
	).PrepareForEval(ctx)
    if qerr != nil {
        return qerr
    }

    if cerr := store.Commit(ctx, txn); cerr != nil {
        return cerr
    }

    tq.PreparedQuery = query

    return nil
}

// evaluates the prepared query
func (tq *TaggerQuery) PreparedQueryEval(input interface{}) (tagsE []string, tagsF []string, tagsT []string) {
    log.Trace("running PreparedQueryEval method on type TaggerQuery")

    // append the QueryID to each list of possible tags
    tagsError := append(tq.Config.TagsError, tq.Config.QueryID)
    tagsFalse := append(tq.Config.TagsFalse, tq.Config.QueryID)
    tagsTrue := append(tq.Config.TagsTrue, tq.Config.QueryID)

    // set the context for query eval
    ctx := context.Background()

    // set the list of EvalOptions before evaluating prepared query
    var options []rego.EvalOption
    options = append(options, rego.EvalInput(input))

    // evaluate the (previously) prepared query
    results, err := tq.PreparedQuery.Eval(ctx, options...)
    if err != nil {
        log.WithError(err).Error("rego (OPA) query returned error : " + tq.Config.QueryString)
        tagsE = tagsError
        return
    } else if len(results) == 0 {
        log.WithError(err).Info("rego (OPA) query returned undefined result : " + tq.Config.QueryString)
        tagsE = tagsError
        return
    } else if len(results) > 1 {
        log.WithError(err).Warning("rego (OPA) query returned more results than expected ; evaluating first result only for query : " + tq.Config.QueryString)
        tagsE = tagsError
    }

    // we evaluate a single document, so we should only have one result;
    // assert that the result type boolean
    result, ok := results[0].Bindings[tq.Config.BindingName].(bool)
    if !ok {
        log.WithError(err).Warning("unexpected (non-bool) type returned from rego (OPA) query : " + tq.Config.QueryString)
        tagsE = tagsError
    }

    // debug logging
    log.Debugf("bindings.%s: %t", tq.Config.BindingName, result)

    // set tags based on true||false value of binding var
    if result {
        tagsT = tagsTrue
    } else {
        tagsF = tagsFalse
    }

    return
}
