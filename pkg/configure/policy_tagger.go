package configure

type TaggerPolicyConfig struct {
    // Tagger type can be used for different types of events, but a given
    // Tagger object must process a single EventType as queries/policies
    // expect/require a specific schema for a given EventType
    EventType      string

    // event.<field> where tags will be added for policies that
    // return some error and/or evaluate to "undefined"
    TagsErrorField string

    // event.<field> where tags will be added for policies that
    // evaluate to boolan false
    TagsFalseField string

    // event.<field> where tags will be added for policies that
    // evaluate to boolan true
    TagsTrueField  string
}
