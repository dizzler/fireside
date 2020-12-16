package configure

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
    AccessLoggers []EnvoyAccesslogConfig `yaml:"access_loggers"`
    Clusters      []EnvoyCluster         `yaml:"clusters"`
    Endpoints     []EnvoyEndpoint        `yaml:"endpoints"`
    Filters       []EnvoyFilter          `yaml:"filters"`
    FilterChains  []EnvoyFilterChain     `yaml:"filter_chains"`
    Listeners     []EnvoyListener        `yaml:"listeners"`
    RouteConfigs  []EnvoyRouteConfig     `yaml:"route_configs"`
    Runtimes      []EnvoyRuntime         `yaml:"runtimes"`
    Secrets       []EnvoySecret          `yaml:"secrets"`
    VirtualHosts  []EnvoyVirtualHost     `yaml:"virtual_hosts"`
}

type PolicyFilter struct {
    Key   string `yaml:"key"`
    Value string `yaml:"value"`
}

type EnvoyAccesslogConfig struct {
    Name        string `yaml:"name"`
    ClusterName string `yaml:"cluster_name"`
    ConfigType  string `yaml:"config_type"`
    LogName     string `yaml:"log_name"`
}

type EnvoyCluster struct {
    Name            string               `yaml:"name"`
    Mode            string               `yaml:"mode"`
    TransportSocket EnvoyTransportSocket `yaml:"transport_socket"`
}

type EnvoyEndpoint struct {
    ClusterName string `yaml:"cluster_name"`
    Host        string `yaml:"host"`
    Port        uint32 `yaml:"port"`
}

type EnvoyFilter struct {
    Name                string            `yaml:"name"`
    Type                string            `yaml:"type"`
    AccessLoggers       []string          `yaml:"access_loggers"`
    ComponentFilterType string            `yaml:"component_filter_type"`
    ComponentFilters    []string          `yaml:"component_filters"`
    ConfigType          string            `yaml:"config_type"`
    Config              EnvoyFilterConfig `yaml:"config"`
    Mode                string            `yaml:"mode"`
    StatPrefix          string            `yaml:"stat_prefix"`
    UpstreamTarget      string            `yaml:"upstream_target"`
}

// Create an empty struct to contain filter-specific configs
type EnvoyFilterConfig struct {
    DynamicStats bool `yaml:"dynamic_stats"`
}

type EnvoyFilterChain struct {
    Name            string               `yaml:"name"`
    Filters         []string             `yaml:"filters"`
    TransportSocket EnvoyTransportSocket `yaml:"transport_socket"`
}

type EnvoyListener struct {
    Name         string   `yaml:"name"`
    Mode         string   `yaml:"mode"`
    Type         string   `yaml:"type"`
    Host         string   `yaml:"host"`
    Port         uint32   `yaml:"port"`
    FilterChains []string `yaml:"filter_chains"`
    Routes       []string `yaml:"routes"`
}

type EnvoyRouteConfig struct {
    Name         string   `yaml:"name"`
    VirtualHosts []string `yaml:"virtual_hosts"`
}

type EnvoyRuntime struct {
    Name string `yaml:"name"`
}

type EnvoySecret struct {
    Name      string                     `yaml:"name"`
    Type      string                     `yaml:"type"`
    BaseDir   string                     `yaml:"base_dir"`
    Ca        EnvoySecretTlsCa           `yaml:"ca"`
    Crt       EnvoySecretTlsCrt          `yaml:"crt"`
    Key       EnvoySecretTlsKey          `yaml:"key"`
    Provision EnvoySecretProvisionConfig `yaml:"provision"`
}

type EnvoySecretProvisionConfig struct {
    CreateIfAbsent bool `yaml:"create_if_absent"`
    ForceRecreate  bool `yaml:"force_recreate"`
}

type EnvoySecretTlsCa struct {
    Name string `yaml:"name"`
}

type EnvoySecretTlsCrt struct {
    CommonName    string   `yaml:"common_name"`
    Country       string   `yaml:"country"`
    DnsNames      []string `yaml:"dns_names"`
    FileName      string   `yaml:"file_name"`
    IpAddresses   []string `yaml:"ip_addresses"`
    Locality      string   `yaml:"locality"`
    Organization  string   `yaml:"organization"`
    PostalCode    string   `yaml:"postal_code"`
    Province      string   `yaml:"province"`
    StreetAddress string   `yaml:"street_address"`
}

type EnvoySecretTlsKey struct {
    FileName string `yaml:"file_name"`
}

type EnvoyTransportSocket struct {
    Name       string   `yaml:"name"`
    Type       string   `yaml:"type"`
    Secrets    []string `yaml:"secrets"`
}

type EnvoyVirtualHost struct {
    Name        string   `yaml:"name"`
    ClusterName string   `yaml:"cluster_name"`
    Domains     []string `yaml:"domains"`
    PrefixMatch string   `yaml:"prefix_match"`
}
