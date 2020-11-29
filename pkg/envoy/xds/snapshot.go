package fireside

import (
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
	//cluster "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	//core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	//discoverygrpc "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	//endpoint "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	//envoy_api_v2_auth "github.com/envoyproxy/go-control-plane/envoy/api/v2/auth"
	//envoy_api_v2_core "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	//hcm "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	//listener "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	//route "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
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

/*
func MakeXdsCluster(clusterName string) *cluster.Cluster {
        hst := &core.Address{Address: &core.Address_SocketAddress{
                SocketAddress: &core.SocketAddress{
                        Address:  remoteHost,
                        Protocol: core.SocketAddress_TCP,
                        PortSpecifier: &core.SocketAddress_PortValue{
                                PortValue: uint32(443),
                        },
                },
        }}
        uctx := &envoy_api_v2_auth.UpstreamTlsContext{}
        tctx, err := ptypes.MarshalAny(uctx)
        if err != nil {
                log.Fatal(err)
        }

        return &cluster.Cluster{
                Name:                 clusterName,
                ConnectTimeout:       ptypes.DurationProto(2 * time.Second),
                ClusterDiscoveryType: &cluster.Cluster_Type{Type: cluster.Cluster_LOGICAL_DNS},
                DnsLookupFamily:      cluster.Cluster_V4_ONLY,
                LbPolicy:             cluster.Cluster_ROUND_ROBIN,
                LoadAssignment: &endpoint.ClusterLoadAssignment{
                        ClusterName: clusterName,
                        Endpoints: []*endpoint.LocalityLbEndpoints{{
                                LbEndpoints: []*endpoint.LbEndpoint{
                                        {
                                                HostIdentifier: &endpoint.LbEndpoint_Endpoint{
                                                        Endpoint: &endpoint.Endpoint{
                                                                Address: hst,
                                                        }},
                                        },
                                },
                        }},
                },
                TransportSocket: &core.TransportSocket{
                        Name: "envoy.transport_sockets.tls",
                        ConfigType: &core.TransportSocket_TypedConfig{
                                TypedConfig: tctx,
                        },
                },
        }
}


func MakeXdsListener(listenerName string) *listener.Listener {
        var targetHost = v
        var targetPrefix = "/"
        var virtualHostName = "local_service"
        var routeConfigName = "local_route"

        log.Infof(">>>>>>>>>>>>>>>>>>> creating listener " + listenerName)

        rte := &route.RouteConfiguration{
                Name: routeConfigName,
                VirtualHosts: []*route.VirtualHost{{
                        Name:    virtualHostName,
                        Domains: []string{"*"},
                        Routes: []*route.Route{{
                                Match: &route.RouteMatch{
                                        PathSpecifier: &route.RouteMatch_Prefix{
                                                Prefix: targetPrefix,
                                        },
                                },
                                Action: &route.Route_Route{
                                        Route: &route.RouteAction{
                                                ClusterSpecifier: &route.RouteAction_Cluster{
                                                        Cluster: clusterName,
                                                },
                                                PrefixRewrite: "/robots.txt",
                                                HostRewriteSpecifier: &route.RouteAction_HostRewriteLiteral{
                                                        HostRewriteLiteral: targetHost,
                                                },
                                        },
                                },
                        }},
                }},
        }

        manager := &hcm.HttpConnectionManager{
                CodecType:  hcm.HttpConnectionManager_AUTO,
                StatPrefix: "ingress_http",
                RouteSpecifier: &hcm.HttpConnectionManager_RouteConfig{
                        RouteConfig: rte,
                },
                HttpFilters: []*hcm.HttpFilter{{
                        Name: wellknown.Router,
                }},
        }

        pbst, err := ptypes.MarshalAny(manager)
        if err != nil {
                log.Fatal(err)
        }

        // use the following imports
        // envoy_api_v2_core "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
        // envoy_api_v2_auth "github.com/envoyproxy/go-control-plane/envoy/api/v2/auth"
        // core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
        // auth "github.com/envoyproxy/go-control-plane/envoy/extensions/transport_sockets/tls/v3"

        // 1. send TLS certs filename back directly

        //sdsTls := &envoy_api_v2_auth.DownstreamTlsContext{
        //        CommonTlsContext: &envoy_api_v2_auth.CommonTlsContext{
        //                TlsCertificates: []*envoy_api_v2_auth.TlsCertificate{{
        //                        CertificateChain: &envoy_api_v2_core.DataSource{
        //                                Specifier: &envoy_api_v2_core.DataSource_InlineBytes{InlineBytes: []byte(pub)},
        //                        },
        //                        PrivateKey: &envoy_api_v2_core.DataSource{
        //                                Specifier: &envoy_api_v2_core.DataSource_InlineBytes{InlineBytes: []byte(priv)},
        //                        },
        //                }},
        //        },
        //}

        // or
        // 2. send TLS SDS Reference value
        // sdsTls := &envoy_api_v2_auth.DownstreamTlsContext{
        // 	CommonTlsContext: &envoy_api_v2_auth.CommonTlsContext{
        // 		TlsCertificateSdsSecretConfigs: []*envoy_api_v2_auth.SdsSecretConfig{{
        // 			Name: "server_cert",
        // 		}},
        // 	},
        // }

        // 3. SDS via ADS

        sdsTls := &auth.DownstreamTlsContext{
                CommonTlsContext: &auth.CommonTlsContext{
                        TlsCertificateSdsSecretConfigs: []*auth.SdsSecretConfig{{
                                Name: "server_cert",
                                SdsConfig: &core.ConfigSource{
                                        ConfigSourceSpecifier: &core.ConfigSource_Ads{
                                                Ads: &core.AggregatedConfigSource{},
                                        },
                                        ResourceApiVersion: core.ApiVersion_V3,
                                },
                        }},
                },
        }

        scfg, err := ptypes.MarshalAny(sdsTls)
        if err != nil {
                log.Fatal(err)
        }

        return &listener.Listener{
                Name: listenerName,
                Address: &core.Address{
                        Address: &core.Address_SocketAddress{
                                SocketAddress: &core.SocketAddress{
                                        Protocol: core.SocketAddress_TCP,
                                        Address:  localhost,
                                        PortSpecifier: &core.SocketAddress_PortValue{
                                                PortValue: 10000,
                                        },
                                },
                        },
                },
                FilterChains: []*listener.FilterChain{{
                        Filters: []*listener.Filter{{
                                Name: wellknown.HTTPConnectionManager,
                                ConfigType: &listener.Filter_TypedConfig{
                                        TypedConfig: pbst,
                                },
                        }},
                        TransportSocket: &core.TransportSocket{
                                Name: "envoy.transport_sockets.tls",
                                ConfigType: &core.TransportSocket_TypedConfig{
                                        TypedConfig: scfg,
                                },
                        },
                }},
        }
}


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
