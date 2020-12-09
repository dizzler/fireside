package fireside

import (
    "errors"
    "fmt"
    "time"

    configure "fireside/pkg/configure"
    tls "fireside/pkg/tls"

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
    tlsauth "github.com/envoyproxy/go-control-plane/envoy/extensions/transport_sockets/tls/v3"
    runtime "github.com/envoyproxy/go-control-plane/envoy/service/runtime/v3"
    types "github.com/envoyproxy/go-control-plane/pkg/cache/types"
    "github.com/envoyproxy/go-control-plane/pkg/resource/v3"
    "github.com/envoyproxy/go-control-plane/pkg/wellknown"
    "github.com/golang/protobuf/proto"
    "github.com/golang/protobuf/ptypes"
    anypb "google.golang.org/protobuf/types/known/anypb"
    pstruct "github.com/golang/protobuf/ptypes/struct"
    wrappers "github.com/golang/protobuf/ptypes/wrappers"
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
    case configure.Ads:
        source.ConfigSourceSpecifier = &core.ConfigSource_Ads{
            Ads: &core.AggregatedConfigSource{},
        }
    case configure.Xds:
        source.ConfigSourceSpecifier = &core.ConfigSource_ApiConfigSource{
            ApiConfigSource: &core.ApiConfigSource{
                TransportApiVersion:       resource.DefaultAPIVersion,
                ApiType:                   core.ApiConfigSource_GRPC,
                SetNodeOnFirstMessageOnly: true,
                GrpcServices: []*core.GrpcService{{
                    TargetSpecifier: &core.GrpcService_EnvoyGrpc_{
                        EnvoyGrpc: &core.GrpcService_EnvoyGrpc{ClusterName: configure.XdsCluster},
                    },
                }},
            },
        }
    case configure.Rest:
        source.ConfigSourceSpecifier = &core.ConfigSource_ApiConfigSource{
            ApiConfigSource: &core.ApiConfigSource{
                ApiType:             core.ApiConfigSource_REST,
                TransportApiVersion: resource.DefaultAPIVersion,
                ClusterNames:        []string{configure.XdsCluster},
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
    var transportSocket *core.TransportSocket
    if len(config.TransportSocket.Type) > 0 {
        transportSocket = MakeTransportSocket(&config.TransportSocket)
    }

    edsSource := configSource(config.Mode)

    connectTimeout := 5 * time.Second
    return &cluster.Cluster{
        Name:                 config.Name,
        ConnectTimeout:       ptypes.DurationProto(connectTimeout),
        ClusterDiscoveryType: &cluster.Cluster_Type{Type: cluster.Cluster_EDS},
        EdsClusterConfig: &cluster.Cluster_EdsClusterConfig{
            EdsConfig: edsSource,
        },
	TransportSocket: transportSocket,
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
        filterList      []configure.EnvoyFilter = policy.Filters
        networkFilters  []*listener.Filter
	transportSocket *core.TransportSocket
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
    if len(config.TransportSocket.Type) > 0 {
	transportSocket = MakeTransportSocket(&config.TransportSocket)
    }
    return &listener.FilterChain{
        Filters: networkFilters,
	TransportSocket: transportSocket,
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
                    Kind: &pstruct.Value_StringValue{StringValue: configure.FiresideStr},
                },
            },
        },
    }
}

// MakeTlsSecrets creates all Envoy secrets specified in policy config
// via a lookup to a tls.TlsTrustDomain
func MakeTlsSecrets(trustDomains []*tls.TlsTrustDomain) ([]types.Resource, error) {
    var resources []types.Resource
    for _, trustDomain := range trustDomains {
        log.Debugf("creating Envoy secret config %s", trustDomain.Name)
        // get certificate data for the TlsTrustDomain's signing CA
        log.Debugf("creating Envoy secret config for TLS CA %s", trustDomain.Name)
        // get the bytes for the PEM encoded CA certificate for the TlsTrustDomain
        caBytes := trustDomain.GetCaBytes()
        // create an Envoy secret formatted for a CA certificate
        caSecret := MakeTlsCaSecret(trustDomain.Name, caBytes)
        resources = append(resources, caSecret)
        // get certificate and key data for Clients associated with the TlsTrustDomain
        for element, trustClient := range trustDomain.Clients {
            if trustClient.Type != configure.SecretTypeTlsClient {
                return nil, errors.New(
                    fmt.Sprintf("cannot create new Client TLS secret for invalid Secret:Type ...", trustClient.Name, trustClient.Type),
                )
            }
            log.Debugf("creating Envoy secret config for TLS Client %s", trustClient.Name)
            clientName, clientBytes, keyBytes, err := trustDomain.GetClientBytes(element)
            if err != nil { return nil, err }
            clientSecret := MakeTlsCrtSecret(clientName, clientBytes, keyBytes)
            resources = append(resources, clientSecret)
        }
        // get certificate and key data for Servers associated with the TlsTrustDomain
        for element, trustServer := range trustDomain.Servers {
            if trustServer.Type != configure.SecretTypeTlsServer {
                return nil, errors.New(
                    fmt.Sprintf("cannot create new Server TLS secret for invalid Secret:Type ...", trustServer.Name, trustServer.Type),
                )
            }
            log.Debugf("creating Envoy secret config for TLS Server %s", trustServer.Name)
            serverName, serverBytes, keyBytes, err := trustDomain.GetServerBytes(element)
            if err != nil { return nil, err }
            serverSecret := MakeTlsCrtSecret(serverName, serverBytes, keyBytes)
            resources = append(resources, serverSecret)
        }
    }
    return resources, nil
}

// MakeTlsTrustDomains creates a tls.TlsTrustDomain (struct) abject for each
// signing CA in the list of Envoy "Secrets". Must be run before MakeTlsSecrets.
func MakeTlsTrustDomains(secrets []configure.EnvoySecret) ([]*tls.TlsTrustDomain, error) {
    var trustDomains []*tls.TlsTrustDomain
    for _, secretCfg := range secrets {
        // create a new TlsTrustDomain for each CA signing cert/secret
        if secretCfg.Type == configure.SecretTypeTlsCa {
            log.Infof("creating TLS CA domain for Envoy secret config %s", secretCfg.Name)
            trustDomain, err := tls.NewTlsTrustDomain(&secretCfg)
            if err != nil {
                log.WithError(err).Fatal("failed to created NewTlsTrustDomain for Envoy secret : " + secretCfg.Name)
            }
            // create the CA certificate and key and add as attributes of the TlsTrustDomain
            trustDomain.NewTlsCa(&secretCfg)
            trustDomains = append(trustDomains, trustDomain)
            if trust_issues := MakeTlsTrusts(secrets, trustDomain); trust_issues != nil {
                log.WithError(err).Fatal("failed to MakeTlsTrusts for TlsTrustDomain : " + trustDomain.Name)
            }
        }
    }
    return trustDomains, nil
}

// MakeTlsTrusts creates certificates and keys for clients and servers in the
// same TlsTrustDomain / signed by the same CA.
func MakeTlsTrusts(secrets []configure.EnvoySecret, td *tls.TlsTrustDomain) error {
    // use a nested for loop to sign client and server certs
    // with the key from the created CA for the given TlsTrustDomain
    for _, secretCfg := range secrets {
	log.Debug("MakeTlsTrusts function processing secret config name = " + secretCfg.Name)
        // generate and/or get certificate and key data for clients
        // and servers associated with the created CA
        if secretCfg.Ca.Name == td.Name {
            switch secretCfg.Type {
            case configure.SecretTypeTlsCa:
                log.Debugf("skipping task for TLS CA %s ; crt/key already configured", secretCfg.Name)
            case configure.SecretTypeTlsClient:
                log.Debug("creating NewTlsClient for secret config name = " + secretCfg.Name)
                // create the Client certificate and key by signing with the
                // CA key for the TlsTrustDomain ; add Client data as an element
                // within the "Clients" attribute of the TlsTrustDomain
                td.NewTlsClient(&secretCfg)
            case configure.SecretTypeTlsServer:
                log.Debug("creating NewTlsServer for secret config name = " + secretCfg.Name)
                // create the Server certificate and key by signing with the
                // CA key for the TlsTrustDomain ; add Server data as an element
                // within the "Servers" attribute of the TlsTrustDomain
                td.NewTlsServer(&secretCfg)
            default:
                log.Debugf("skipping TLS cert creation for non-TLS secret type %s", secretCfg.Type)
            }
        } else {
            log.Debugf("skipping task for TTD = %s : CA = %s : SecretCfg = %s", secretCfg.Name, secretCfg.Ca.Name, td.Name)
        }
    }
    return nil
}

// MakeSecret generates an Envoy secret config for a Certificate Authority (CA) certificate
func MakeTlsCaSecret(caName string, caChain []byte) *tlsauth.Secret {
    return &tlsauth.Secret{
        Name: caName,
        Type: &tlsauth.Secret_ValidationContext{
            ValidationContext: &tlsauth.CertificateValidationContext{
                TrustedCa: &core.DataSource{
                    Specifier: &core.DataSource_InlineBytes{InlineBytes: []byte(caChain)},
                },
            },
        },
    }
}

// MakeTlsCrtSecret creates an Envoy secret config for a client||server TLS certificate
func MakeTlsCrtSecret(pemName string, pemChain []byte, pemKey []byte) *tlsauth.Secret {
    return &tlsauth.Secret{
        Name: pemName,
        Type: &tlsauth.Secret_TlsCertificate{
            TlsCertificate: &tlsauth.TlsCertificate{
                CertificateChain: &core.DataSource{
                    Specifier: &core.DataSource_InlineBytes{InlineBytes: []byte(pemChain)},
                },
                PrivateKey: &core.DataSource{
                    Specifier: &core.DataSource_InlineBytes{InlineBytes: []byte(pemKey)},
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

// MakeTransportSocket creates a core.TransportSocket which can be used for
// configuring downstream and/or upstream TLS "context" (i.e. encryption)
// as well as other transports such as network tap.
func MakeTransportSocket(config *configure.EnvoyTransportSocket) *core.TransportSocket {
    genSdsSecCfg := func(secretName string) *tlsauth.SdsSecretConfig {
        return &tlsauth.SdsSecretConfig{
            Name: secretName,
            SdsConfig: &core.ConfigSource{
                ConfigSourceSpecifier: &core.ConfigSource_Ads{
                    Ads: &core.AggregatedConfigSource{},
                },
                ResourceApiVersion: core.ApiVersion_V3,
            },
	}
    }
    genTransSock := func(anyp *anypb.Any) *core.TransportSocket {
        return &core.TransportSocket{
            Name: wellknown.TransportSocketTls,
            ConfigType: &core.TransportSocket_TypedConfig{
                TypedConfig: anyp,
            },
        }
    }
    // create the ptype config for the corresponding Type of Transport Socket config
    switch config.Type {
    case configure.TransportSocketTlsDownstream:
        var sdsSecrets []*tlsauth.SdsSecretConfig
        for _, secretName := range config.Secrets {
             sdsSecrets = append(sdsSecrets, genSdsSecCfg(secretName))
        }
        tstd := &tlsauth.DownstreamTlsContext{
            CommonTlsContext: &tlsauth.CommonTlsContext{
                TlsCertificateSdsSecretConfigs: sdsSecrets,
            },
        }
        ptypeCfg := MarshalAnyPtype(tstd)
        return genTransSock(ptypeCfg)
    case configure.TransportSocketTlsUpstream:
        var sdsSecrets []*tlsauth.SdsSecretConfig
        for _, secretName := range config.Secrets {
             sdsSecrets = append(sdsSecrets, genSdsSecCfg(secretName))
        }
        tstu := &tlsauth.UpstreamTlsContext{
            CommonTlsContext: &tlsauth.CommonTlsContext{
                TlsCertificateSdsSecretConfigs: sdsSecrets,
            },
        }
        ptypeCfg := MarshalAnyPtype(tstu)
        return genTransSock(ptypeCfg)
    default:
        log.Fatal("cannot MakeTransportSocket for unsupported Type = " + config.Type)
	return nil
    }
}

// MakeVirtualHost creates a virtual host config to register with
// Envoy's Route Discovery Service (RDS). Used by HTTP Connection Manager
func MakeVirtualHost(config *configure.EnvoyVirtualHost) *route.VirtualHost {
    return &route.VirtualHost{
        Name: config.Name,
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
