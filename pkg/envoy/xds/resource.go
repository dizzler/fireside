package fireside

import (
	"io/ioutil"
	"time"

	configure "fireside/pkg/configure"

	log "github.com/sirupsen/logrus"

	alf "github.com/envoyproxy/go-control-plane/envoy/config/accesslog/v3"
	cluster "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	endpoint "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	listener "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	route "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	als "github.com/envoyproxy/go-control-plane/envoy/extensions/access_loggers/grpc/v3"
	hcm "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	router "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/router/v3"
	tcp "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/tcp_proxy/v3"
	auth "github.com/envoyproxy/go-control-plane/envoy/extensions/transport_sockets/tls/v3"
	runtime "github.com/envoyproxy/go-control-plane/envoy/service/runtime/v3"
	//"github.com/envoyproxy/go-control-plane/pkg/cache/types"
	//"github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	"github.com/envoyproxy/go-control-plane/pkg/resource/v3"
	"github.com/envoyproxy/go-control-plane/pkg/wellknown"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	//any "github.com/golang/protobuf/ptypes/any"
	anypb "google.golang.org/protobuf/types/known/anypb"
	pstruct "github.com/golang/protobuf/ptypes/struct"
	wrappers "github.com/golang/protobuf/ptypes/wrappers"
)

const (
	// commonly used string
	fireside_str = "fireside"

	// XdsCluster is the cluster name for the control server (used by non-ADS set-up)
	XdsCluster = "xds_cluster"

	// Ads mode for resources: one aggregated xDS service
	Ads = "ads"

	// Xds mode for resources: individual xDS services
	Xds = "xds"

	// Rest mode for resources: polling using Fetch
	Rest = "rest"
)

var (
	// RefreshDelay for the polling config source
	RefreshDelay = 500 * time.Millisecond
)

// data source configuration
func configSource(mode string) *core.ConfigSource {
	source := &core.ConfigSource{}
	source.ResourceApiVersion = resource.DefaultAPIVersion
	switch mode {
	case Ads:
		source.ConfigSourceSpecifier = &core.ConfigSource_Ads{
			Ads: &core.AggregatedConfigSource{},
		}
	case Xds:
		source.ConfigSourceSpecifier = &core.ConfigSource_ApiConfigSource{
			ApiConfigSource: &core.ApiConfigSource{
				TransportApiVersion:       resource.DefaultAPIVersion,
				ApiType:                   core.ApiConfigSource_GRPC,
				SetNodeOnFirstMessageOnly: true,
				GrpcServices: []*core.GrpcService{{
					TargetSpecifier: &core.GrpcService_EnvoyGrpc_{
						EnvoyGrpc: &core.GrpcService_EnvoyGrpc{ClusterName: XdsCluster},
					},
				}},
			},
		}
	case Rest:
		source.ConfigSourceSpecifier = &core.ConfigSource_ApiConfigSource{
			ApiConfigSource: &core.ApiConfigSource{
				ApiType:             core.ApiConfigSource_REST,
				TransportApiVersion: resource.DefaultAPIVersion,
				ClusterNames:        []string{XdsCluster},
				RefreshDelay:        ptypes.DurationProto(RefreshDelay),
			},
		}
	}
	return source
}

// MakeAccesslogGrpc creates a gRPC access logger configuration for use
// in other Envoy resource configs (e.g. HTTP Connection Manager)
func MakeAccesslogConfig(config *configure.EnvoyAccesslogConfig) *alf.AccessLog {
	var accesslogCfg *als.HttpGrpcAccessLogConfig
	switch {
	case config.ConfigType == wellknown.HTTPGRPCAccessLog:
		accesslogCfg = &als.HttpGrpcAccessLogConfig{
			CommonConfig: &als.CommonGrpcAccessLogConfig{
				LogName: config.LogName,
				GrpcService: &core.GrpcService{
					TargetSpecifier: &core.GrpcService_EnvoyGrpc_{
						EnvoyGrpc: &core.GrpcService_EnvoyGrpc{
							ClusterName: config.ClusterName,
						},
					},
				},
			},
		}
	default:
		log.Fatal("unable to create acceesslog config for type : " + config.ConfigType)
	}

	ptac := MarshalAnyPtype(accesslogCfg)
	return &alf.AccessLog{
		Name: config.ConfigType,
		ConfigType: &alf.AccessLog_TypedConfig{
			TypedConfig: ptac,
		},
	}
}

// MakeCluster creates a cluster using either ADS or EDS.
func MakeCluster(config *configure.EnvoyCluster) *cluster.Cluster {
	edsSource := configSource(config.Mode)

	connectTimeout := 5 * time.Second
	return &cluster.Cluster{
		Name:                 config.Name,
		ConnectTimeout:       ptypes.DurationProto(connectTimeout),
		ClusterDiscoveryType: &cluster.Cluster_Type{Type: cluster.Cluster_EDS},
		EdsClusterConfig: &cluster.Cluster_EdsClusterConfig{
			EdsConfig: edsSource,
		},
	}
}

// MakeEndpoint creates a localhost endpoint on a given port.
func MakeEndpoint(config *configure.EnvoyEndpoint) *endpoint.ClusterLoadAssignment {
	return &endpoint.ClusterLoadAssignment{
		ClusterName: config.ClusterName,
		Endpoints: []*endpoint.LocalityLbEndpoints{{
			LbEndpoints: []*endpoint.LbEndpoint{{
				HostIdentifier: &endpoint.LbEndpoint_Endpoint{
					Endpoint: &endpoint.Endpoint{
						Address: &core.Address{
							Address: &core.Address_SocketAddress{
								SocketAddress: &core.SocketAddress{
									Protocol: core.SocketAddress_TCP,
									Address:  config.Host,
									PortSpecifier: &core.SocketAddress_PortValue{
										PortValue: config.Port,
									},
								},
							},
						},
					},
				},
			}},
		}},
	}
}

// MakeFilterChain creates a filter chain (i.e. named list of network filters)
// for use by a given Envoy listener
func MakeFilterChain(config *configure.EnvoyFilterChain, policy *configure.PolicyConfig) *listener.FilterChain {
	var (
		filterList     []configure.EnvoyFilter = policy.Filters
		networkFilters []*listener.Filter
	)
	for _, filterName := range config.Filters {
		for _, filterCfg := range filterList {
			if filterName == filterCfg.Name {
				log.Debug("creating network filter " + filterCfg.Name)
				filter := MakeNetworkFilter(&filterCfg, policy)
				networkFilters = append(networkFilters, filter)
			}
		}
	}
	return &listener.FilterChain{
		Filters: networkFilters,
	}
}

// MakeFilter creates a filter for use in an Envoy FilterChan
func MakeHttpFilter(config *configure.EnvoyFilter) *hcm.HttpFilter {
	if config.Type != "http" {
		log.Fatal("cannot MakeHttpFilter for filter type " + config.Type)
	}
	nestedFunc := func(anyp *anypb.Any) *hcm.HttpFilter {
		return &hcm.HttpFilter{
			Name: config.ConfigType,
			ConfigType: &hcm.HttpFilter_TypedConfig{
				TypedConfig: anyp,
			},
		}
	}
	switch {
	case config.ConfigType == wellknown.Router:
		rtr := &router.Router{
			DynamicStats: &wrappers.BoolValue{Value: config.Config.DynamicStats},
		}
		ptypeCfg := MarshalAnyPtype(rtr)
		return nestedFunc(ptypeCfg)
	default:
		log.Fatal("MakeHttpFilter function does not support ConfigType " + config.ConfigType)
		return nil
	}
}

// MakeNetworkFilter creates an Envoy network filter (e.g. TCP, HTTP Connection Manager, etc.)
// from supplied configs
func MakeNetworkFilter(config *configure.EnvoyFilter, policy *configure.PolicyConfig) *listener.Filter {
	if config.Type != "network" {
		log.Fatal("cannot MakeNetworkFilter for filter type " + config.Type)
	}
	nestedFunc := func(anyp *anypb.Any) *listener.Filter {
		return &listener.Filter{
			Name: config.ConfigType,
			ConfigType: &listener.Filter_TypedConfig{
				TypedConfig: anyp,
			},
		}
	}
	switch {
	case config.ConfigType == wellknown.HTTPConnectionManager:
		connMgr := MakeHttpConnectionManagerConfig(config, policy)
		ptypeCfg := MarshalAnyPtype(connMgr)
		return nestedFunc(ptypeCfg)
	case config.ConfigType == wellknown.TCPProxy:
		tcpProxy := MakeTcpProxyConfig(config)
		ptypeCfg := MarshalAnyPtype(tcpProxy)
		return nestedFunc(ptypeCfg)
	default:
		log.Fatal("MakeNetworkFilter function does not support ConfigType " + config.ConfigType)
		return nil
	}
}

// MakeHttpConnectionManagerConfig creates an hcm.HttpConnectionManager typed
// filter configuration, using either ADS or RDS for route discovery
func MakeHttpConnectionManagerConfig(config *configure.EnvoyFilter, policy *configure.PolicyConfig) *hcm.HttpConnectionManager {
	// set vars used in this function
	var (
		accesslogConfigs []*alf.AccessLog
		accesslogCfgList []configure.EnvoyAccesslogConfig = policy.AccessLoggers
		filterList       []configure.EnvoyFilter = policy.Filters
		httpFilters      []*hcm.HttpFilter
	)
	// generate the list of Envoy accesslog configurations
	for _, loggerName := range config.AccessLoggers {
		for _, loggerCfg := range accesslogCfgList {
			if loggerName == loggerCfg.Name {
				log.Debug("creating access log config " + loggerCfg.Name)
				accesslogConfig := MakeAccesslogConfig(&loggerCfg)
				accesslogConfigs = append(accesslogConfigs, accesslogConfig)
			}
		}
	}

	// generate the list of HTTP filters
	for _, filterName := range config.ComponentFilters {
		for _, filterCfg := range filterList {
			if config.ComponentFilterType == filterCfg.Type {
				if filterName == filterCfg.Name {
					log.Debug("creating http filter " + filterCfg.Name)
					httpFilter := MakeHttpFilter(&filterCfg)
					httpFilters = append(httpFilters, httpFilter)
				}
			}
		}
	}

	// discover the route configuration for this HTTP Connection Manager
	rdsSource := configSource(config.Mode)

	// HTTP filter configuration
	return &hcm.HttpConnectionManager{
		CodecType:  hcm.HttpConnectionManager_AUTO,
		StatPrefix: config.StatPrefix,
		RouteSpecifier: &hcm.HttpConnectionManager_Rds{
			Rds: &hcm.Rds{
				ConfigSource:    rdsSource,
				RouteConfigName: config.UpstreamTarget,
			},
		},
		HttpFilters: httpFilters,
		AccessLog: accesslogConfigs,
	}
}

// MakeListener creates an Envoy listener from supplied configs
func MakeListener(config *configure.EnvoyListener, policy *configure.PolicyConfig) *listener.Listener {
	var (
		filterChains []*listener.FilterChain
		filterChainList []configure.EnvoyFilterChain = policy.FilterChains
	)
	for _, filterChainName := range config.FilterChains {
		for _, filterChainCfg := range filterChainList {
			if filterChainName == filterChainCfg.Name {
				log.Debug("creating filter chain " + filterChainCfg.Name)
				filterChain := MakeFilterChain(&filterChainCfg, policy)
				filterChains = append(filterChains, filterChain)
			}
		}
	}
	return &listener.Listener{
		Name: config.Name,
		Address: &core.Address{
			Address: &core.Address_SocketAddress{
				SocketAddress: &core.SocketAddress{
					Protocol: core.SocketAddress_TCP,
					Address:  config.Host,
					PortSpecifier: &core.SocketAddress_PortValue{
						PortValue: config.Port,
					},
				},
			},
		},
		FilterChains: filterChains,
	}
}

// MakeRouteConfig creates an HTTP route config that routes to a given cluster,
// where the created route is discoverable (via ADS || RDS) for HTTP Connection Manager.
func MakeRouteConfig(routeCfg *configure.EnvoyRouteConfig, policy *configure.PolicyConfig) *route.RouteConfiguration {
	var (
		// create an empty slice of virtual hosts to create for the route
		routeVhosts []*route.VirtualHost
		vhostList []configure.EnvoyVirtualHost = policy.VirtualHosts
	)
	for _, vhostString := range routeCfg.VirtualHosts {
		for _, vhostCfg := range vhostList {
			if vhostString == vhostCfg.Name {
				log.Debug("creating virtual host " + vhostCfg.Name)
				vhost := MakeVirtualHost(&vhostCfg)
				routeVhosts = append(routeVhosts, vhost)
			}
		}
	}
	return &route.RouteConfiguration{
		Name: routeCfg.Name,
		VirtualHosts: routeVhosts,
	}
}

// MakeRuntime creates an RTDS layer with some fields, where the RTDS layer allows
// the runtime itself to be discoverable via Envoy xDS.
func MakeRuntime(config *configure.EnvoyRuntime) *runtime.Runtime {
	return &runtime.Runtime{
		Name: config.Name,
		Layer: &pstruct.Struct{
			Fields: map[string]*pstruct.Value{
				"xds-server-type": {
					Kind: &pstruct.Value_StringValue{StringValue: fireside_str},
				},
			},
		},
	}
}

// MakeSecrets generates an SDS secret
func MakeSecrets(config *configure.EnvoySecret) []*auth.Secret {
	privateChain, err := ioutil.ReadFile(config.CrtFilePath)
	if err != nil {
		log.Fatal(err)
	}
	privateKey, err := ioutil.ReadFile(config.KeyFilePath)
	if err != nil {
		log.Fatal(err)
	}
	rootCert, err := ioutil.ReadFile(config.CaFilePath)
	if err != nil {
		log.Fatal(err)
	}
	return []*auth.Secret{
		{
			Name: config.CrtSecretName,
			Type: &auth.Secret_TlsCertificate{
				TlsCertificate: &auth.TlsCertificate{
					PrivateKey: &core.DataSource{
						Specifier: &core.DataSource_InlineBytes{InlineBytes: []byte(privateKey)},
					},
					CertificateChain: &core.DataSource{
						Specifier: &core.DataSource_InlineBytes{InlineBytes: []byte(privateChain)},
					},
				},
			},
		},
		{
			Name: config.CaSecretName,
			Type: &auth.Secret_ValidationContext{
				ValidationContext: &auth.CertificateValidationContext{
					TrustedCa: &core.DataSource{
						Specifier: &core.DataSource_InlineBytes{InlineBytes: []byte(rootCert)},
					},
				},
			},
		},
	}
}

// MakeTcpProxyConfig creates a tcp.TcpProxy typed filter config
func MakeTcpProxyConfig(config *configure.EnvoyFilter) *tcp.TcpProxy {
	return &tcp.TcpProxy{
		ClusterSpecifier: &tcp.TcpProxy_Cluster{
			Cluster: config.UpstreamTarget,
		},
		StatPrefix: config.StatPrefix,
	}
}

// MakeVirtualHost creates a virtual host config to register
// with Envoy's Route Discovery Service (RDS). Used by HTTP Connection Manager.
func MakeVirtualHost(config *configure.EnvoyVirtualHost) *route.VirtualHost {
	return &route.VirtualHost{
		Name:    config.Name,
		Domains: config.Domains,
		Routes: []*route.Route{{
			Match: &route.RouteMatch{
				PathSpecifier: &route.RouteMatch_Prefix{
					Prefix: config.PrefixMatch,
				},
			},
			Action: &route.Route_Route{
				Route: &route.RouteAction{
					ClusterSpecifier: &route.RouteAction_Cluster{
						Cluster: config.ClusterName,
					},
				},
			},
		}},
	}
}

// marshal any proto.Message to a protobuf type
func MarshalAnyPtype(m proto.Message) *anypb.Any {
	p, err := ptypes.MarshalAny(m)
	if err != nil {
		log.Fatal("failed to marshal message to protobuf type")
	}
	return p
}
