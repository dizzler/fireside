package fireside

// Common strings used throughout fireside
const (
    ArchiveSuffix string = "tar.gz"
    FiresideStr = "fireside"
    // Ads mode for resources: one aggregated xDS service
    Ads = "ads"
    Filterkey_Node = "node-id"
    GrpcMaxConcurrentStreams = 1000000
    OutputSuffix  string = "out"
    // Rest mode for resources: polling using Fetch
    Rest = "rest"
    // Xds mode for resources: individual xDS services
    Xds = "xds"
    // XdsCluster is the cluster name for the control server (used by non-ADS set-up)
    XdsCluster = "xds_cluster"
    SecretTypeTlsCa = "tls-ca"
    SecretTypeTlsClient = "tls-client"
    SecretTypeTlsServer = "tls-server"
)
