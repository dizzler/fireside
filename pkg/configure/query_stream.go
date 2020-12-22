package configure

type QueryStreamPolicyConfig struct {
    // number of "ProcessData" method calls that may run in parallel threads
    Concurrency    int                 `yaml:"concurrency"`

    // QueryStream type can be used for different types of events, but a given
    // QueryStream object must process a single EventType as queries/policies
    // expect/require a specific schema for a given EventType
    EventType      string              `yaml:"event_type"`

    // list of configurations for the list (i.e. "stream") of queries that
    // implement QueryStreamPolicy logic
    QueryConfigs   []QueryStreamConfig `yaml:"query_configs"`

    // event.<field> where tags will be added for policies that
    // return some error and/or evaluate to "undefined"
    TagsIssueField string              `yaml:"tags_issue_field"`

    // event.<field> where tags will be added for policies that
    // evaluate to boolan true
    TagsMatchField string              `yaml:"tags_match_field"`

    // event.<field> where tags will be added for policies that
    // evaluate to boolan false
    TagsMissField  string              `yaml:"tags_miss_field"`
}

type QueryStreamConfig struct {
    // name of the boolan "binding" (var) to process after query eval;
    // the value of the binding / variable determines which set of
    // tags return (and later insert) for a given source document/event
    BindingName string   `yaml:"binding_name"`

    // filesystem path of OPA bundle to load into memory; expects to find
    // either a compressed bundle file or a directory to be loaded as a bundle;
    // a "bundle" is a set of directories, .rego modules and .json|.yaml data
    // files under BundlePath that conform to OPA's "Bundle File Format":
    //     https://www.openpolicyagent.org/docs/latest/management/#bundle-file-format
    BundlePath  string   `yaml:"bundle_path"`

    // name of the Rego package to set on the query's context
    Package     string   `yaml:"package"`

    // unique ID of for this query; used for tagging / tracking;
    // the value of QueryID is the base/default tag for a given eval result;
    // for example, a query that evaluates to true will generate a list of
    // tags with at least one element, with the QueryID appended to the
    // TagsMatch list of tags yielding a final list with at least one element
    QueryID     string   `yaml:"query_id"`

    // query string to use / evaluate for a given input document / event
    QueryString string   `yaml:"query_string"`

    // list of extra tags to return when the query evaluates to error and/or undefined
    TagsIssue   []string `yaml:"tags_issue"`

    // list of extra tags to return when the query evaluates to boolean true
    TagsMatch   []string `yaml:"tags_match"`

    // list of extra tags to return when the query evaluates to boolean false
    TagsMiss    []string `yaml:"tags_miss"`

    // controls whether tracing is enabled during query evaluation
    TraceQuery  bool     `yaml:"trace_query"`
}
