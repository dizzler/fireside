package fireside

// config struct for Policy
type Policy struct {
    // unique ID of the policy
    Id       string         `yaml:"id"`

    // type of policy ; used for filtering on subsets of policies
    Type     string         `yaml:"type"`

    // lower priority wins when multiple policies conflict for some node
    Priority uint           `yaml:"priority"`

    // filter policy application to only apply on selected node(s)
    Filters  []PolicyFilter `yaml:"filters"`

    // configs for policies to apply
    Config   PolicyConfig   `yaml:"config"`
}

type PolicyConfig struct {
    Clusters     []EnvoyCluster     `yaml:"clusters"`
    Endpoints    []EnvoyEndpoint    `yaml:"endpoints"`
    Filters      []EnvoyFilter      `yaml:"filters"`
    Listeners    []EnvoyListener    `yaml:"listeners"`
    RouteConfigs []EnvoyRouteConfig `yaml:"route_configs"`
    Secrets      []EnvoySecret      `yaml:"secrets"`
    VirtualHosts []EnvoyVirtualHost `yaml:"virtual_hosts"`
}

type PolicyFilter struct {
    Key   string `yaml:"key"`
    Value string `yaml:"value"`
}

type EnvoyCluster struct {
    Name string `yaml:"name"`
    Mode string `yaml:"mode"`
}

type EnvoyEndpoint struct {
    ClusterName string `yaml:"cluster_name"`
    Host        string `yaml:"host"`
    Port        uint32 `yaml:"port"`
}

type EnvoyFilter struct {
    Name   string            `yaml:"name"`
    Type   string            `yaml:"type"`
    Config map[string]string `yaml:"config"`
}

type EnvoyFilterChain struct {
    Name    string   `yaml:"name"`
    Filters []string `yaml:"filters"`
}

type EnvoyListener struct {
    Name         string             `yaml:"name"`
    Type         string             `yaml:"type"`
    Host         string             `yaml:"host"`
    Port         uint32             `yaml:"port"`
    FilterChains []EnvoyFilterChain `yaml:"filter_chains"`
    Routes       []string           `yaml:"routes"`
}

type EnvoyRouteConfig struct {
    Name         string   `yaml:"name"`
    VirtualHosts []string `yaml:"virtual_hosts"`
}

type EnvoySecret struct {
    CaFilePath    string `yaml:"ca_file_path"`
    CaSecretName  string `yaml:"ca_secret_name"`
    CrtFilePath   string `yaml:"crt_file_path"`
    CrtSecretName string `yaml:"crt_secret_name"`
    KeyFilePath   string `yaml:"key_file_path"`
}

type EnvoyVirtualHost struct {
    Name        string   `yaml:"name"`
    ClusterName string   `yaml:"cluster_name"`
    Domains     []string `yaml:"domains"`
    PrefixMatch string   `yaml:"prefix_match"`
}