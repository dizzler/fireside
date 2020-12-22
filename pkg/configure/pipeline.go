package configure

// data structure for configuring an event processing "Pipeline"
type PipelineConfig struct {
    // pipeline state used to initialize component processors
    State    PipelineConfigState    `yaml:"state"`

    // pipeline policies
    Policies PipelineConfigPolicies `yaml:"policies"`
}

// wraps the set of policies associated with a given pipeline
type PipelineConfigPolicies struct {
    // configs for data processing policies that use OPA queries
    // to drive tagging decisions
    EventTagger QueryStreamPolicyConfig  `yaml:"event_tagger"`
}

type PipelineConfigState struct {
    // control whether the pipeline components should be created/run
    Enable             bool `yaml:"enable"`

    // default number of "ProcessData" method calls that may run
    // in parallel threads for data processors in the pipeline
    DefaultConcurrency int  `yaml:"default_concurrency"`
}
