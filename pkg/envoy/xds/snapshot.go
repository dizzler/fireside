package fireside

import (
	configure "fireside/pkg/configure"
	log "github.com/sirupsen/logrus"
	//"fmt"
	//"os"

	//"sync"
	//"sync/atomic"
	//"time"


	//"github.com/golang/protobuf/ptypes"

	//log "github.com/sirupsen/logrus"
	//"google.golang.org/grpc"

	// sub-packages of go-control-plane ============================
	//auth "github.com/envoyproxy/go-control-plane/envoy/extensions/transport_sockets/tls/v3"
	cachev3 "github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	cluster "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	//core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	//discoverygrpc "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	endpoint "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	//envoy_api_v2_auth "github.com/envoyproxy/go-control-plane/envoy/api/v2/auth"
	//envoy_api_v2_core "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	//hcm "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	listener "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	route "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	//serverv3 "github.com/envoyproxy/go-control-plane/pkg/server/v3"
	//types "github.com/envoyproxy/go-control-plane/pkg/cache/types"
	//wellknown "github.com/envoyproxy/go-control-plane/pkg/wellknown"
	// =============================================================
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

func MakeNodeSnapshot(config *configure.Policy) {
	// create vars used in this function
	var (
		policy *configure.PolicyConfig = &config.Config
		routes []*route.RouteConfiguration
	)
	// create Envoy route configs as defined by policy ; explicit creation of route
	// configs implicitly results in the creation of associated resources, such as:
	//   - MakeVirtualHost
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
	var listeners []*listener.Listener
	for _, listenerCfg := range policy.Listeners {
		log.Debug("creating Envoy listener config " + listenerCfg.Name)
		listener := MakeListener(&listenerCfg, policy)
		listeners = append(listeners, listener)
	}
	// create Envoy (upstream) clusters as defined by policy
	var clusters []*cluster.Cluster
	for _, clusterCfg := range policy.Clusters {
		log.Debug("creating Envoy cluster config " + clusterCfg.Name)
		cluster := MakeCluster(&clusterCfg)
		clusters = append(clusters, cluster)
	}
	// create Envoy (upstream) endpoints (assigned to clusters) as defined by policy
	var endpoints []*endpoint.ClusterLoadAssignment
	for _, endpointCfg := range policy.Endpoints {
		log.Debug("creating Envoy endpoint config for cluster " + endpointCfg.ClusterName)
		endpoint := MakeEndpoint(&endpointCfg)
		endpoints = append(endpoints, endpoint)
	}
	// MakeRuntime
	// MakeSecrets
}

/*
func MakeXdsSecret(secretName string, pub []byte, priv []byte) *auth.Secret {
        var secretName = "server_cert"

        log.Infof(">>>>>>>>>>>>>>>>>>> creating Secret " + secretName)
        return &auth.Secret{
                Name: secretName,
                Type: &auth.Secret_TlsCertificate{
                        TlsCertificate: &auth.TlsCertificate{
                                CertificateChain: &core.DataSource{
                                        Specifier: &core.DataSource_InlineBytes{InlineBytes: []byte(pub)},
                                },
                                PrivateKey: &core.DataSource{
                                        Specifier: &core.DataSource_InlineBytes{InlineBytes: []byte(priv)},
                                },
                        },
                },
        }
}
*/


/*
func MakeXdsAggResource(clusterName string, listenerName string, secretName string) {
        atomic.AddInt32(&version, 1)
        log.Infof(">>>>>>>>>>>>>>>>>>> creating snapshot Version " + fmt.Sprint(version))

        snap := cachev3.NewSnapshot(fmt.Sprint(version), nil, cluster, nil, listener, nil, secret)
        if err := snap.Consistent(); err != nil {
                log.Errorf("snapshot inconsistency: %+v\n%+v", snap, err)
                os.Exit(1)
        }
        err = cache.SetSnapshot(nodeId, snap)
        if err != nil {
                log.Fatalf("Could not set snapshot %v", err)
        }

        time.Sleep(60 * time.Second)
}
*/


/*
// EnvoyNodeSnapshot holds parameters for a synthetic snapshot.
type EnvoyNodeSnapshot struct {
	// Xds indicates snapshot mode: ads, xds, or rest
	Xds string
	// Version for the snapshot.
	Version string
	// UpstreamPort for the single endpoint on the localhost.
	UpstreamPort uint32
	// BasePort is the initial port for the listeners.
	BasePort uint32
	// NumClusters is the total number of clusters to generate.
	NumClusters int
	// NumHTTPListeners is the total number of HTTP listeners to generate.
	NumHTTPListeners int
	// NumTCPListeners is the total number of TCP listeners to generate.
	// Listeners are assigned clusters in a round-robin fashion.
	NumTCPListeners int
	// NumRuntimes is the total number of RTDS layers to generate.
	NumRuntimes int
	// TLS enables SDS-enabled TLS mode on all listeners
	TLS bool
}

// Generate produces a snapshot from the parameters.
func (ts EnvoyNodeSnapshot) Generate() cache.Snapshot {
	clusters := make([]types.Resource, ts.NumClusters)
	endpoints := make([]types.Resource, ts.NumClusters)
	for i := 0; i < ts.NumClusters; i++ {
		name := fmt.Sprintf("cluster-%s-%d", ts.Version, i)
		clusters[i] = MakeCluster(ts.Xds, name)
		endpoints[i] = MakeEndpoint(name, ts.UpstreamPort)
	}

	routes := make([]types.Resource, ts.NumHTTPListeners)
	for i := 0; i < ts.NumHTTPListeners; i++ {
		name := fmt.Sprintf("route-%s-%d", ts.Version, i)
		routes[i] = MakeRoute(name, cache.GetResourceName(clusters[i%ts.NumClusters]))
	}

	total := ts.NumHTTPListeners + ts.NumTCPListeners
	listeners := make([]types.Resource, total)
	for i := 0; i < total; i++ {
		port := ts.BasePort + uint32(i)
		// listener name must be same since ports are shared and previous listener is drained
		name := fmt.Sprintf("listener-%d", port)
		var listener *listener.Listener
		if i < ts.NumHTTPListeners {
			listener = MakeHTTPListener(ts.Xds, name, port, cache.GetResourceName(routes[i]))
		} else {
			listener = MakeTCPListener(name, port, cache.GetResourceName(clusters[i%ts.NumClusters]))
		}

		if ts.TLS {
			for i, chain := range listener.FilterChains {
				tlsc := &auth.DownstreamTlsContext{
					CommonTlsContext: &auth.CommonTlsContext{
						TlsCertificateSdsSecretConfigs: []*auth.SdsSecretConfig{{
							Name:      tlsName,
							SdsConfig: configSource(ts.Xds),
						}},
						ValidationContextType: &auth.CommonTlsContext_ValidationContextSdsSecretConfig{
							ValidationContextSdsSecretConfig: &auth.SdsSecretConfig{
								Name:      rootName,
								SdsConfig: configSource(ts.Xds),
							},
						},
					},
				}
				mt, _ := ptypes.MarshalAny(tlsc)
				chain.TransportSocket = &core.TransportSocket{
					Name: "envoy.transport_sockets.tls",
					ConfigType: &core.TransportSocket_TypedConfig{
						TypedConfig: mt,
					},
				}
				listener.FilterChains[i] = chain
			}
		}

		listeners[i] = listener
	}

	runtimes := make([]types.Resource, ts.NumRuntimes)
	for i := 0; i < ts.NumRuntimes; i++ {
		name := fmt.Sprintf("runtime-%d", i)
		runtimes[i] = MakeRuntime(name)
	}

	var secrets []types.Resource
	if ts.TLS {
		for _, s := range MakeSecrets(tlsName, rootName) {
			secrets = append(secrets, s)
		}
	}

	out := cache.NewSnapshot(
		ts.Version,
		endpoints,
		clusters,
		routes,
		listeners,
		runtimes,
		secrets,
	)

	return out
}
*/
