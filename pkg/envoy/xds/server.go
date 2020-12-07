package fireside

import (
    "context"
    "encoding/json"
    "fmt"
    "net"

    "sync"
    "time"

    cachev3 "github.com/envoyproxy/go-control-plane/pkg/cache/v3"
    discoverygrpc "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
    serverv3 "github.com/envoyproxy/go-control-plane/pkg/server/v3"

    log "github.com/sirupsen/logrus"
    "google.golang.org/grpc"

    configure "fireside/pkg/configure"
)

type Callbacks struct {
    Signal       chan struct{}
    Debug        bool
    Fetches      int
    Requests     int
    RequestsChan chan *discoverygrpc.DiscoveryRequest
    mu           sync.Mutex
}

func (cb *Callbacks) Report() {
    cb.mu.Lock()
    defer cb.mu.Unlock()
    log.WithFields(log.Fields{"fetches": cb.Fetches, "requests": cb.Requests}).Info("cb.Report()  callbacks")
}
func (cb *Callbacks) OnStreamOpen(_ context.Context, id int64, typ string) error {
    log.Infof("OnStreamOpen %d open for %s", id, typ)
    return nil
}
func (cb *Callbacks) OnStreamClosed(id int64) {
    log.Infof("OnStreamClosed %d closed", id)
}
func (cb *Callbacks) OnStreamRequest(id int64, r *discoverygrpc.DiscoveryRequest) error {
    log.Infof("OnStreamRequest for Envoy node ID %s : TypeUrl %v", r.Node.Id, r.TypeUrl)
    cb.RequestsChan <- r
    cb.mu.Lock()
    defer cb.mu.Unlock()
    cb.Requests++
    if cb.Signal != nil {
        close(cb.Signal)
        cb.Signal = nil
    }
    return nil
}
func (cb *Callbacks) OnStreamResponse(int64, *discoverygrpc.DiscoveryRequest, *discoverygrpc.DiscoveryResponse) {
    log.Infof("OnStreamResponse...")
    cb.Report()
}
func (cb *Callbacks) OnFetchRequest(ctx context.Context, req *discoverygrpc.DiscoveryRequest) error {
    log.Infof("OnFetchRequest...")
    cb.mu.Lock()
    defer cb.mu.Unlock()
    cb.Fetches++
    if cb.Signal != nil {
        close(cb.Signal)
        cb.Signal = nil
    }
    return nil
}
func (cb *Callbacks) OnFetchResponse(*discoverygrpc.DiscoveryRequest, *discoverygrpc.DiscoveryResponse) {
    log.Infof("OnFetchResponse...")
}

type XdsServer struct {
    ActiveNodes  []string
    Cache        cachev3.SnapshotCache
    Cb           *Callbacks
    Ctx          context.Context
    Policies     []configure.Policy
    Port         uint
    RequestsChan chan *discoverygrpc.DiscoveryRequest
    Server       serverv3.Server
    Snapshots    map[string]*EnvoySnapshot
}

func NewXdsServer(config *configure.Config) *XdsServer {
    ctx := context.Background()

    log.Info("Starting control plane for Envoy xDS")

    reqchan := make(chan *discoverygrpc.DiscoveryRequest)
    signal := make(chan struct{})
    cb := &Callbacks{
        Signal:   signal,
        Fetches:  0,
        Requests: 0,
        RequestsChan: reqchan,
    }
    cache := cachev3.NewSnapshotCache(true, cachev3.IDHash{}, nil)

    policies := config.Policies

    port := config.Inputs.Envoy.Xds.Server.Port
    srv := serverv3.NewServer(ctx, cache, cb)
    snaps := make(map[string]*EnvoySnapshot)

    return &XdsServer{
        Cache: cache,
        Cb: cb,
        Ctx: ctx,
        Policies: policies,
        Port: port,
        RequestsChan: reqchan,
        Server: srv,
        Snapshots: snaps,
    }
}

// RunManagementServer starts an xDS server at the given port
func (xc *XdsServer) RunManagementServer() {
    var grpcOptions []grpc.ServerOption
    grpcOptions = append(grpcOptions, grpc.MaxConcurrentStreams(configure.GrpcMaxConcurrentStreams))
    grpcServer := grpc.NewServer(grpcOptions...)

    lis, err := net.Listen("tcp", fmt.Sprintf(":%d", xc.Port))
    if err != nil {
        log.WithError(err).Fatal("failed to listen")
    }

    // register services
    discoverygrpc.RegisterAggregatedDiscoveryServiceServer(grpcServer, xc.Server)

    log.WithFields(log.Fields{"port": xc.Port}).Info("management server listening")
    go func() {
        if err = grpcServer.Serve(lis); err != nil {
            log.Error(err)
        }
    }()
    <-xc.Ctx.Done()

    grpcServer.GracefulStop()
}

func (xc *XdsServer) ListStatusKeys() []string {
    status_keys := xc.Cache.GetStatusKeys()

    for _, s_key := range status_keys {
        log.Debug("Detected Envoy node with status key = " + s_key)
    }
    return status_keys
}

func (xc *XdsServer) GetNodeInfo(nodeId string) []byte {
    log.Debug("Getting info for Envoy node ID " + nodeId)
    info := xc.Cache.GetStatusInfo(nodeId)
    node := info.GetNode()
    nodeJson, err := json.Marshal(node)
    if err != nil {
        log.Error(err)
    }
    log.Debug("Printing info for Envoy node ID " + nodeId + " : " + string(nodeJson))
    return nodeJson
}

// check snapshot consistency before applying in order to ensure resource
// dependencies are met within the snapshot
func (xc *XdsServer) PolicySnapshotCheck(nodeId string) bool {
    log.Debug("checking snapshot consistency for Envoy node ID = " + nodeId)
    snap := xc.Snapshots[nodeId]
    if err := snap.AssertSnapshotIsConsistent(); err != nil {
        log.WithError(err).Errorf("snapshot inconsistency: %+v\n", snap.Snapshot)
        return false
    }
    return true
}

// generates a snapshot of aggregated, dynamic resources for the specified Envoy node ID
func (xc *XdsServer) PolicySnapshotGenerate(nodeId string) error {
    for _, envoyPolicy := range xc.Policies {
        for _, policyFilter := range envoyPolicy.Filters {
            // apply the first policy match for the specified nodeId
            switch {
            case (policyFilter.Key == configure.Filterkey_Node) && (policyFilter.Value == nodeId):
		// create / get TLS CA, certs and keys based on policy configs
		log.Info("creating TLS Trust Domains for Envoy TLS 'secrets'")
		trustDomains, err := MakeTlsTrustDomains(envoyPolicy.Config.Secrets)
		if err != nil {
                    return err
		}

                // create a new snapshot of aggregated resources for the Envoy node
                snap := NewEnvoySnapshot(&envoyPolicy.Config, trustDomains)
                // associated the generated snapshot with the specified Envoy node ID
                snap.SetNodeId(nodeId)

		// generate the resource definitions used in the Envoy node snapshot
                log.Debug("generating snapshot for Envoy nodeId " + nodeId)
                if err := snap.GenerateSnapshot(); err != nil {
                    return err
                }

                // add the *EnvoySnapshot (config wrapper) to the map of snapshots
                xc.Snapshots[nodeId] = snap
            default:
                log.Warn("no policy filter match; unable to PolicySnapshotGenerate() for Envoy node ID = " + nodeId)
            }
        }
    }
    return nil
}

// applies the policy to the Envoy node by applying any pending update
// from the snapshot cache for the specified node ID
func (xc *XdsServer) PolicySnapshotSet(nodeId string, version int32) error {
    snap := *xc.Snapshots[nodeId]
    // set the version for the snapshot update of the corresponding Envoy node ID
    snap.SetVersion(version)
    strVersion := xc.Snapshots[nodeId].Version
    log.Debugf("set snapshot version to %s for Envoy node ID %s", strVersion, nodeId)

    // apply the snapshot for the node by updating the snapshot cache
    log.Infof("applying snapshot version %s for Envoy node ID %s", strVersion, nodeId)
    if err := xc.Cache.SetSnapshot(nodeId, *snap.Snapshot); err != nil {
        log.WithError(err).Errorf("failed to set snapshot for Envoy node ID %s : Version %s", nodeId, strVersion)
        return err
    }
    return nil
}

// adds a given Envoy node ID to the list of 'ActiveNodes' for the xDS server
func (xc *XdsServer) RegisterNodeId(nodeId string) {
    log.Debug("updating XDS 'ActiveNodes' ; adding Envoy node ID = " + nodeId)
    xc.ActiveNodes = append(xc.ActiveNodes, nodeId)
}

// handle discovery requests from xDS gRPC stream
func (xc *XdsServer) StreamHandleDiscoveryRequest() {
    for req := range xc.RequestsChan {
        // validate the request
        if len(req.Node.Id) == 0 {
            log.Error("invalid DiscoveryRequest does not contain Envoy node ID in 'Node' field")
        } else {
            // update the list of xDS "ActiveNodes"
            nodeId := req.Node.Id
            xc.RegisterNodeId(nodeId)

            // check the type of resource being requested
            if len(req.TypeUrl) > 0 {
                log.Debug("received DiscoveryRequest for Envoy resource type : " + req.TypeUrl)
            }

            // set the version for the snapshot update
            var (
                verr    error
                Version int32 = 1
            )
            if len(req.VersionInfo) > 0 {
                Version, verr = VersionToInt32(req.VersionInfo)
                if verr != nil {
                    log.Error("failed to convert snapshot version from string to int32 for Envoy node id = " + nodeId)
                }
            }
            // generates aggregate Envoy resources, combined into a single snapshot
            log.Infof("generating snapshot version %d for Envoy node ID %s", Version, nodeId)
            if err := xc.PolicySnapshotGenerate(nodeId); err != nil {
                log.WithError(err).Error("failed to generate snapshot from policy configs for Envoy node ID = " + nodeId)
            } else {
                // verifies the consistency of the snapshot by validating that
                // resource dependencies are met
                if !xc.PolicySnapshotCheck(nodeId) {
                    log.Error("snapshot failed consistency checks for Envoy node ID = " + nodeId)
                } else {
                    // apply policy by updating snapshot cache for matching Envoy node IDs
                    log.Info("setting snapshot for Envoy node ID = " + nodeId)
                    err := xc.PolicySnapshotSet(nodeId, Version)
                    if err != nil {
                        log.Error("failed to apply policy / set snapshot for Envoy node ID = " + nodeId)
                    }
                }
            }
        }
    }
}

func ServeEnvoyXds(config *configure.Config) {
    xdss := NewXdsServer(config)
    // start the xDS server
    go xdss.RunManagementServer()

    // use separate goroutine(s) to run the stream handler(s) for
    // serving xDS responses to Envoy clients
    go xdss.StreamHandleDiscoveryRequest()

    // Create a ticker for periodic refresh of state
    var refreshSeconds int = 15
    refreshInterval := time.Duration(refreshSeconds)
    refreshTicker := time.NewTicker(refreshInterval * time.Second)
    // Create a common 'quit' channel for stopping ticker(s) as needed
    quit := make(chan struct{})
    go func() {
        for {
            select {
            case <- refreshTicker.C:
                xdss.ListStatusKeys()
            case <- quit:
                refreshTicker.Stop()
                return
            }
        }
    }()

    <-xdss.Cb.Signal
}
