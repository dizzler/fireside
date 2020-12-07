package fireside

import (
    "errors"
    "fmt"
    "reflect"
    "strconv"
    "strings"
    configure "fireside/pkg/configure"
    tls "fireside/pkg/tls"

    cachev3 "github.com/envoyproxy/go-control-plane/pkg/cache/v3"
    log "github.com/sirupsen/logrus"
    types "github.com/envoyproxy/go-control-plane/pkg/cache/types"
)

type EnvoySnapshot struct {
    // Unique ID of the Envoy node for which a snapshot will be generated & applied
    NodeId          string
    // aggregated policy config determines the subset of aggregated resources
    // to create for a given node (i.e. the snapshot)
    Policy          *configure.PolicyConfig
    // use the GenerateSnapshot() method to instantiate the Snapshot field
    Snapshot        *cachev3.Snapshot
    // Data structure for storing certs and keys associated with Envoy TLS secrets
    TlsTrustDomains []*tls.TlsTrustDomain
    // version of the snapshot to create (e.g. "v1", "v2", "v3", etc.)
    Version         string
}

func NewEnvoySnapshot(policy *configure.PolicyConfig, trustDomains []*tls.TlsTrustDomain) *EnvoySnapshot {
    return &EnvoySnapshot{Policy: policy, TlsTrustDomains: trustDomains}
}

func (ns *EnvoySnapshot) AssertSnapshotIsConsistent() error {
    snap := *ns.Snapshot
    if err := snap.Consistent(); err != nil { return err }
    return nil
}

func (ns *EnvoySnapshot) GenerateSnapshot() error {
    var policy *configure.PolicyConfig = ns.Policy
    // create Envoy (upstream) endpoints (assigned to clusters) as defined by policy
    var endpoints []types.Resource
    for _, endpointCfg := range policy.Endpoints {
        log.Debug("creating Envoy endpoint config for cluster " + endpointCfg.ClusterName)
        endpoint := MakeEndpoint(&endpointCfg)
        endpoints = append(endpoints, endpoint)
    }
    // create Envoy (upstream) clusters as defined by policy
    var clusters []types.Resource
    for _, clusterCfg := range policy.Clusters {
        log.Debug("creating Envoy cluster config " + clusterCfg.Name)
        cluster := MakeCluster(&clusterCfg)
        clusters = append(clusters, cluster)
    }
    // create Envoy route configs as defined by policy ; explicit creation of route
    // configs implicitly results in the creation of associated resources, such as:
    //   - MakeVirtualHost
    var routes []types.Resource
    for _, routeCfg := range policy.RouteConfigs {
        log.Debug("creating Envoy route config " + routeCfg.Name)
        route := MakeRouteConfig(&routeCfg, policy)
        routes = append(routes, route)
    }
    // create Envoy (downstream) listeners as defined by policy ; explicit creation of
    // listener implicitly results in the creation of associated resources, such as:
    //   - MakeFilterChain
    //   - MakeNetworkFilter
    //   - MakeHttpConnectionManagerConfig
    //     - MakeAccesslogConfig
    //     - MakeHttpFilter
    //   - MakeTcpProxyConfig
    var listeners []types.Resource
    for _, listenerCfg := range policy.Listeners {
        log.Debug("creating Envoy listener config " + listenerCfg.Name)
        listener := MakeListener(&listenerCfg, policy)
        listeners = append(listeners, listener)
    }
    // create Envoy runtime, discoverable within Envoy via RTDS
    var runtimes []types.Resource
    for _, runtimeCfg := range policy.Runtimes {
        log.Debug("creating Envoy runtime config " + runtimeCfg.Name)
        runtime := MakeRuntime(&runtimeCfg)
        runtimes = append(runtimes, runtime)
    }
    // create Envoy secrets for TLS certs used in downstream and/or upstream connections
    secrets, err := MakeTlsSecrets(ns.TlsTrustDomains)
    if err != nil {
        return err
    }
    // create cachev3.NewSnapshot using generated resources
    snap := cachev3.NewSnapshot(
        ns.Version,
        endpoints,
        clusters,
        routes,
        listeners,
        runtimes,
        secrets,
    )
    ns.Snapshot = &snap
    return nil
}

func (ns *EnvoySnapshot) SetNodeId(nodeId string) {
    ns.NodeId = nodeId
}

func (ns *EnvoySnapshot) SetVersion(snapVersion int32) {
    ns.Version = fmt.Sprintf("v%d", snapVersion)
}

func VersionToInt32(version string) (int32, error) {
    var versionInt32 int32
    // get the version of the snapshot in native (string) format
    if len(version) == 0 {
        return versionInt32, errors.New("failed to convert snapshot version to int32 ; version (string) is not set)")
    }
    // remove the 'v' from the beginning of version string
    ersion := strings.Trim(version, "v")

    // convert string version to int32() representation
    versionInt, err := strconv.ParseInt(ersion, 0, 32)
    versionInt32 = int32(versionInt)
    log.Debug(fmt.Println(versionInt, err, reflect.TypeOf(versionInt32)))

    return versionInt32, nil
}
