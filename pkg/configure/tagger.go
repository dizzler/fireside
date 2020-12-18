package configure

type TaggerPolicyConfig struct {
    // Tagger type can be used for different types of events, but a given
    // Tagger object must process a single EventType as queries/policies
    // expect/require a specific schema for a given EventType
    EventType      string `yaml:"event_type"`

    // list of configurations for the tagger queries that implement TaggerPolicy logic
    QueryConfigs   []TaggerQueryConfig `yaml:"query_configs"`

    // event.<field> where tags will be added for policies that
    // return some error and/or evaluate to "undefined"
    TagsErrorField string `yaml:"tags_error_field"`

    // event.<field> where tags will be added for policies that
    // evaluate to boolan false
    TagsFalseField string `yaml:"tags_false_field"`

    // event.<field> where tags will be added for policies that
    // evaluate to boolan true
    TagsTrueField  string `yaml:"tags_true_field"`
}

type TaggerQueryConfig struct {
    // name of the boolan "binding" (var) to process after query eval;
    // the value of the binding / variable determines which set of
    // tags return (and later insert) for a given source document/event
    BindingName   string `yaml:"binding_name"`

    // filesystem path of OPA bundle to load into memory; expects to find
    // either a compressed bundle file or a directory to be loaded as a bundle;
    // a "bundle" is a set of directories, .rego modules and .json|.yaml data
    // files under BundlePath that conform to OPA's "Bundle File Format":
    //     https://www.openpolicyagent.org/docs/latest/management/#bundle-file-format
    BundlePath    string `yaml:"bundle_path"`

    // name of the Rego package to set on the query's context
    Package       string `yaml:"package"`

    // unique ID of for this query; used for tagging / tracking;
    // the value of QueryID is the base/default tag for a given eval result;
    // for example, a query that evaluates to true will generate a list of
    // tags with at least one element, with the QueryID appended to the
    // TagsTrue list of tags yielding a final list with at least one element
    QueryID       string `yaml:"query_id"`

    // query string to use / evaluate for a given input document / event
    QueryString   string `yaml:"query_string"`

    // list of extra tags to return when the query evaluates to error and/or undefined
    TagsError     []string `yaml:"tags_error"`

    // list of extra tags to return when the query evaluates to boolean false
    TagsFalse     []string `yaml:"tags_false"`

    // list of extra tags to return when the query evaluates to boolean true
    TagsTrue      []string `yaml:"tags_true"`
}
