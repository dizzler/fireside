package configure

// config struct for Policy
type Policy struct {
    // unique ID of the policy
    Id           string             `yaml:"id"`

    // type of policy ; used for filtering on subsets of policies
    Type         string             `yaml:"type"`

    // lower priority wins when multiple policies conflict for some node
    Priority     uint               `yaml:"priority"`

    // filter policy application to only apply on selected node(s)
    Filters      []PolicyFilter     `yaml:"filters"`

    // configs for policies to apply to Envoy proxies in order to
    // provision dynamic resources via Envoy xDS APIs
    EnvoyPolicy  EnvoyPolicyConfig  `yaml:"envoy_policy"`
}

type PolicyFilter struct {
    Key   string `yaml:"key"`
    Value string `yaml:"value"`
}
