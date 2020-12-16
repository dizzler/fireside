package fireside

import "os"

// Common strings used throughout fireside
const (
    ArchiveSuffix string = "tar.gz"
    FiresideStr = "fireside"
    // Ads mode for resources: one aggregated xDS service
    Ads = "ads"
    // *NIX file mode to set on TLS certificates written to filesystem
    FileModeCrt os.FileMode = 0640
    // *NIX file mode to set on TLS keys written to filesystem
    FileModeKey os.FileMode = 0600
    // Filter policies by matching NodeId
    Filterkey_Node = "node-id"
    // Upper limit on th maximum number of concurrent gRPC streams that can be handled
    GrpcMaxConcurrentStreams = 1000000
    // Suffix to add to name of files created for caching event data
    OutputSuffix  string = "out"
    // Rest mode for resources: polling using Fetch
    Rest = "rest"
    SecretTypeTlsCa = "tls-ca"
    SecretTypeTlsClient = "tls-client"
    SecretTypeTlsServer = "tls-server"
    // internal names for supported types of Transport Sockets, used for TLS & Tap
    TransportSocketTlsDownstream = "downstream-tls-context"
    TransportSocketTlsUpstream = "upstream-tls-context"
    // Xds mode for resources: individual xDS services
    Xds = "xds"
    // XdsCluster is the cluster name for the control server (used by non-ADS set-up)
    XdsCluster = "xds_cluster"
)

// Common strings for different types of events
const (
    CachePrefixEnvoy        = "fireside-envoy-event-cache"
    CachePrefixFalco        = "fireside-falco-event-cache"
    EventCategorySysAudit   = "sys_audit"
    EventCategoryHttp       = "proxy_http"
    EventCategoryTcp        = "proxy_tcp"
    EventSrcGrpc            = "envoy_grpc"
    EventTypeEnvoyAccesslog = "envoy_accesslog"
    EventTypeFalcoAlert     = "falco_alert"
    UnknownType             = "unknown-type"
)
