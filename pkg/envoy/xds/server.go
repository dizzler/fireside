package fireside

import (
    "context"
    "encoding/json"
    "fmt"
    "net"

    "sync"
    "time"

    cachev3 "github.com/envoyproxy/go-control-plane/pkg/cache/v3"
    cluster "github.com/envoyproxy/go-control-plane/envoy/service/cluster/v3"
    endpoint "github.com/envoyproxy/go-control-plane/envoy/service/endpoint/v3"
    discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
    listener "github.com/envoyproxy/go-control-plane/envoy/service/listener/v3"
    route "github.com/envoyproxy/go-control-plane/envoy/service/route/v3"
    runtime "github.com/envoyproxy/go-control-plane/envoy/service/runtime/v3"
    rsrc "github.com/envoyproxy/go-control-plane/pkg/resource/v3"
    secret "github.com/envoyproxy/go-control-plane/envoy/service/secret/v3"
    serverv3 "github.com/envoyproxy/go-control-plane/pkg/server/v3"

    log "github.com/sirupsen/logrus"
    "google.golang.org/grpc"

    configure "fireside/pkg/configure"
)

type Callbacks struct {
    Debug            bool
    Fetches          int
    Requests         int
    RequestsChan     chan *discovery.DiscoveryRequest
    ResponsesChan    chan *discovery.DiscoveryResponse
    Signal           chan struct{}
    StreamOpenChan   chan int64
    StreamClosedChan chan int64
    mu               sync.Mutex
}

func (cb *Callbacks) Report() {
    cb.mu.Lock()
    defer cb.mu.Unlock()
    log.WithFields(log.Fields{"fetches": cb.Fetches, "requests": cb.Requests}).Info("cb.Report()  callbacks")
}
func (cb *Callbacks) OnStreamOpen(_ context.Context, id int64, typ string) error {
    log.Infof("OnStreamOpen %d open for %s", id, typ)
    cb.StreamOpenChan <- id
    return nil
}
func (cb *Callbacks) OnStreamClosed(id int64) {
    log.Infof("OnStreamClosed %d closed", id)
    cb.StreamClosedChan <- id
}
func (cb *Callbacks) OnStreamRequest(id int64, req *discovery.DiscoveryRequest) error {
    log.Infof("OnStreamRequest for Envoy node ID %s : TypeUrl %v", req.Node.Id, req.TypeUrl)
    cb.RequestsChan <- req
    cb.mu.Lock()
    defer cb.mu.Unlock()
    cb.Requests++
    if cb.Signal != nil {
        close(cb.Signal)
        cb.Signal = nil
    }
    return nil
}
func (cb *Callbacks) OnStreamResponse(int64, *discovery.DiscoveryRequest, *discovery.DiscoveryResponse) {
    log.Infof("OnStreamResponse...")
    cb.Report()
}
func (cb *Callbacks) OnFetchRequest(ctx context.Context, req *discovery.DiscoveryRequest) error {
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
func (cb *Callbacks) OnFetchResponse(*discovery.DiscoveryRequest, *discovery.DiscoveryResponse) {
    log.Infof("OnFetchResponse...")
}

type XdsServer struct {
    Cache            cachev3.SnapshotCache
    Cb               *Callbacks
    Ctx              context.Context
    Policies         []configure.Policy
    Port             uint
    Server           serverv3.Server
    Snapshots        map[string]*EnvoySnapshot
    Stream           *XdsStream
    StreamClosedChan chan int64
    StreamOpenChan   chan int64
}

func NewXdsServer(config *configure.Config) *XdsServer {
    ctx := context.Background()

    log.Info("Starting control plane for Envoy xDS")

    reqchan := make(chan *discovery.DiscoveryRequest, 10)
    respchan := make(chan *discovery.DiscoveryResponse, 10)
    s_open_chan := make(chan int64)
    s_closed_chan := make(chan int64)
    s_reqchan := make(chan *discovery.DiscoveryRequest, 10)
    s_respchan := make(chan *discovery.DiscoveryResponse, 10)
    signal := make(chan struct{})

    // implement the Callbacks interface for the xDS server
    cb := &Callbacks{
        Fetches:  0,
        Requests: 0,
        RequestsChan: reqchan,
        ResponsesChan: respchan,
        Signal:   signal,
        StreamOpenChan: s_open_chan,
        StreamClosedChan: s_closed_chan,
    }

    // create Envoy snapshot cache
    cache := cachev3.NewSnapshotCache(true, cachev3.IDHash{}, nil)

    policies := config.Policies
    port := config.Inputs.Envoy.Xds.Server.Port
    srv := serverv3.NewServer(ctx, cache, cb)
    snaps := make(map[string]*EnvoySnapshot)

    stream := &XdsStream{
        ctx: ctx,
        Requests: s_reqchan,
        Responses: s_respchan,
    }

    return &XdsServer{
        Cache: cache,
        Cb: cb,
        Ctx: ctx,
        Policies: policies,
        Port: port,
        Server: srv,
        Snapshots: snaps,
	Stream: stream,
        StreamOpenChan: s_open_chan,
        StreamClosedChan: s_closed_chan,
    }
}

// RunManagementServer starts an xDS server at the given port
func (xc *XdsServer) ListStatusKeys() []string {
    status_keys := xc.Cache.GetStatusKeys()

    for _, s_key := range status_keys {
        log.Debug("Detected Envoy node with status key = " + s_key)
    }
    return status_keys
}

// generates snapshot resources for the specified Envoy node ID by referencing policy configs
func (xc *XdsServer) GenerateNodeSnapshotResources(nodeId string) {
    // set the version for the snapshot update
    var Version int32 = 1
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

// handle Discovery requests from xDS gRPC stream
func (xc *XdsServer) HandleDiscoveryRequest() {
    for req := range xc.Cb.RequestsChan {
        // validate the request
        if len(req.Node.Id) == 0 {
            log.Error("invalid DiscoveryRequest does not contain Envoy node ID in 'Node' field")
        } else if len(req.TypeUrl) > 0 {
            // check the type of resource being requested
            log.Debug("received DiscoveryRequest for Envoy resource type : " + req.TypeUrl)
            log.Debug("received DiscoveryRequest for Envoy resource version : " + req.VersionInfo)
	    xc.Stream.Requests <- req
            go func() {
                var err error
                switch req.TypeUrl {
                case rsrc.EndpointType:
                    err = xc.Server.StreamEndpoints(xc.Stream)
                case rsrc.ClusterType:
                    err = xc.Server.StreamClusters(xc.Stream)
                case rsrc.RouteType:
                    err = xc.Server.StreamRoutes(xc.Stream)
                case rsrc.ListenerType:
                    err = xc.Server.StreamListeners(xc.Stream)
                case rsrc.SecretType:
                    err = xc.Server.StreamSecrets(xc.Stream)
                case rsrc.RuntimeType:
                    err = xc.Server.StreamRuntime(xc.Stream)
                case configure.UnknownType:
                    err = xc.Server.StreamAggregatedResources(xc.Stream)
                }
                if err != nil {
                    log.Errorf("gRPC stream with Envoy xDS node %s reported error : %v", req.Node.Id, err)
                }
		var sig struct{}
                xc.Cb.Signal <- sig
            }()
        } else {
            log.Errorf("received DiscoveryRequest without TypeUrl specified for Envoy node = %s", req.Node.Id)
        }
    }
}

// handle newly opened streams
func (xc *XdsServer) HandleOpenStream() {
    for _ = range xc.StreamOpenChan {
        // look for Envoy nodes for which we have not generated a snapshot
        for _, node := range xc.ListStatusKeys() {
            if _, ok := xc.Snapshots[node]; ok {
                log.Debug("found existing snapshot for Envoy node ID = " + node)
            } else {
                log.Debug("creating new snapshot from policy configs for Envoy node ID = " + node)
                go xc.GenerateNodeSnapshotResources(node)
            }
        }
    }
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
    // get the string format of the snapshot version
    strVersion := snap.GetVersion(nodeId)
    log.Debugf("set snapshot version to %s for Envoy node ID %s", strVersion, nodeId)

    // apply the snapshot for the node by updating the snapshot cache
    log.Infof("applying snapshot version %s for Envoy node ID %s", strVersion, nodeId)
    if err := xc.Cache.SetSnapshot(nodeId, *snap.Snapshot); err != nil {
        log.WithError(err).Errorf("failed to set snapshot for Envoy node ID %s : Version %s", nodeId, strVersion)
        return err
    }
    return nil
}

// runs an xDS management server for dynamic configuration of Envoy proxy resources
func (xc *XdsServer) RunManagementServer() {
    var grpcOptions []grpc.ServerOption
    grpcOptions = append(grpcOptions, grpc.MaxConcurrentStreams(configure.GrpcMaxConcurrentStreams))
    grpcServer := grpc.NewServer(grpcOptions...)

    lis, err := net.Listen("tcp", fmt.Sprintf(":%d", xc.Port))
    if err != nil {
        log.WithError(err).Fatal("failed to listen")
    }

    // register services
    discovery.RegisterAggregatedDiscoveryServiceServer(grpcServer, xc.Server)
    cluster.RegisterClusterDiscoveryServiceServer(grpcServer, xc.Server)
    endpoint.RegisterEndpointDiscoveryServiceServer(grpcServer, xc.Server)
    listener.RegisterListenerDiscoveryServiceServer(grpcServer, xc.Server)
    route.RegisterRouteDiscoveryServiceServer(grpcServer, xc.Server)
    runtime.RegisterRuntimeDiscoveryServiceServer(grpcServer, xc.Server)
    secret.RegisterSecretDiscoveryServiceServer(grpcServer, xc.Server)

    log.WithFields(log.Fields{"port": xc.Port}).Info("management server listening")
    go func() {
        if err = grpcServer.Serve(lis); err != nil {
            log.Error(err)
        }
    }()
    <-xc.Ctx.Done()

    grpcServer.GracefulStop()
}

func ServeEnvoyXds(config *configure.Config) {
    xdss := NewXdsServer(config)
    // start the xDS server
    go xdss.RunManagementServer()

    // use separate goroutine(s) to run the stream handler(s) for
    // serving xDS responses to Envoy clients
    go xdss.HandleOpenStream()
    go xdss.HandleDiscoveryRequest()

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
