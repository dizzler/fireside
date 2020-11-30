package fireside

import (
	"fmt"
	configure "fireside/pkg/configure"
	log "github.com/sirupsen/logrus"

	cachev3 "github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	types "github.com/envoyproxy/go-control-plane/pkg/cache/types"
)

var (
	debug       bool
	onlyLogging bool
	withALS     bool

	port        uint
	gatewayPort uint
	alsPort     uint

	mode string

	version int32

	cache cachev3.SnapshotCache
)

type EnvoySnapshot struct {
	// NewSnapshotCache to create for storing snapshot updates
	Cache    cachev3.SnapshotCache
	// Unique ID of the Envoy node for which a snapshot will be generated & applied
	NodeId   string
        // aggregated policy config determines the subset of aggregated resources
	// to create for a given node (i.e. the snapshot)
	Policy   *configure.PolicyConfig
	// use the GenerateSnapshot() method to instantiate the Snapshot field
	Snapshot cachev3.Snapshot
	// version of the snapshot to create (e.g. "v1", "v2", "v3", etc.)
	Version  string
}

func NewEnvoySnapshot(cache cachev3.SnapshotCache, policy *configure.PolicyConfig) *EnvoySnapshot {
	return &EnvoySnapshot{Cache: cache, Policy: policy}
}

func (ns *EnvoySnapshot) ApplySnapshot() {
	err := ns.Cache.SetSnapshot(ns.NodeId, ns.Snapshot)
	if err != nil {
		log.Fatalf("failed to set Envoy snapshot for nodeId : %s : %v", err)
	}
}

func (ns *EnvoySnapshot) AssertSnapshotIsConsistent() error {
	err := ns.Snapshot.Consistent()
	if err != nil {
		return err
	}
	return nil
}

func (ns *EnvoySnapshot) GenerateSnapshot() {
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
	var secrets []types.Resource
	for _, secretCfg := range policy.Secrets {
		log.Debugf("creating Envoy secret configs for CA = %s : CRT = %s", secretCfg.CaSecretName, secretCfg.CrtSecretName)
		authSecrets := MakeSecrets(&secretCfg)
		for _, authSecret := range authSecrets {
			secrets = append(secrets, authSecret)
		}
	}
	// create cachev3.NewSnapshot using generated resources
	ns.Snapshot = cachev3.NewSnapshot(
		ns.Version,
		endpoints,
		clusters,
		routes,
		listeners,
		runtimes,
		secrets,
	)
}

func (ns *EnvoySnapshot) SetNodeId(nodeId string) {
	ns.NodeId = nodeId
}

func (ns *EnvoySnapshot) SetVersion(snapVersion int32) {
	ns.Version = fmt.Sprintf("v%d", snapVersion)
}
