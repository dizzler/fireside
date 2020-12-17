package query

import (
    "context"

    //"fireside/pkg/configure"

    log "github.com/sirupsen/logrus"

    "github.com/open-policy-agent/opa/rego"
    "github.com/open-policy-agent/opa/storage"
    "github.com/open-policy-agent/opa/storage/inmem"
)

type TaggerQuery struct {
    // name of the boolan "binding" (var) to process after query eval;
    // the value of the binding / variable determines which set of
    // tags return (and later insert) for a given source document/event
    BindingName   string

    // filesystem path of OPA bundle to load into memory; expects to find
    // either a compressed bundle file or a directory to be loaded as a bundle;
    // a "bundle" is a set of directories, .rego modules and .json|.yaml data
    // files under BundlePath that conform to OPA's "Bundle File Format":
    //   https://www.openpolicyagent.org/docs/latest/management/#bundle-file-format
    BundlePath    string

    // PreparedQuery is set when PrepareQuery() method is invoked
    PreparedQuery *rego.PreparedEvalQuery

    // unique ID of for this query; used for tagging / tracking;
    // the value of QueryID is the base/default tag for a given eval result;
    // for example, a query that evaluates to true will generate a list of
    // tags with at least one element, with the QueryID appended to the
    // TagsTrue list of tags yielding a final list with at least one element
    QueryID       string

    // query string to use / evaluate for a given input document / event
    QueryString   string

    // list of extra tags to return when the query evaluates to error and/or undefined
    TagsError     []string

    // list of extra tags to return when the query evaluates to boolean false
    TagsFalse     []string

    // list of extra tags to return when the query evaluates to boolean true
    TagsTrue      []string
}

// evaluates the prepared query
func (tq *TaggerQuery) EvaluateQuery(input interface{}, ctx context.Context) (tagsE []string, tagsF []string, tagsT []string) {
    log.Trace("running EvaluateQuery method on type TaggerQuery")

    // append the QueryID to each list of possible tags
    tagsError := append(tq.TagsError, tq.QueryID)
    tagsFalse := append(tq.TagsFalse, tq.QueryID)
    tagsTrue := append(tq.TagsTrue, tq.QueryID)

    // evaluate the (previously) prepared query
    results, err := tq.PreparedQuery.Eval(ctx, rego.EvalInput(input))
    if err != nil {
        log.WithError(err).Error("rego (OPA) query returned error : " + tq.QueryString)
        tagsE = tagsError
        return
    } else if len(results) == 0 {
        log.WithError(err).Info("rego (OPA) query returned undefined result : " + tq.QueryString)
        tagsE = tagsError
        return
    } else if len(results) > 1 {
        log.WithError(err).Warning("rego (OPA) query returned more results than expected ; evaluating first result only for query : " + tq.QueryString)
        tagsE = tagsError
    }

    // we evaluate a single document, so we should only have one result;
    // assert that the result type boolean
    result, ok := results[0].Bindings[tq.BindingName].(bool)
    if !ok {
        log.WithError(err).Warning("unexpected (non-bool) type returned from rego (OPA) query : " + tq.QueryString)
        tagsE = tagsError
    }

    // debug logging
    log.Debugf("bindings.%s: %t", tq.BindingName, result)

    // set tags based on true||false value of binding var
    if result {
        tagsT = tagsTrue
    } else {
        tagsF = tagsFalse
    }

    return
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
	rego.Query(tq.QueryString),
        rego.LoadBundle(tq.BundlePath),
        rego.Store(store),
        rego.Transaction(txn),
	).PrepareForEval(ctx)
    if qerr != nil {
        return qerr
    }

    if cerr := store.Commit(ctx, txn); cerr != nil {
        return cerr
    }

    tq.PreparedQuery = &query

    return nil
}
