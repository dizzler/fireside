package fireside

import (
	"context"
	"encoding/json"
	"fmt"
	"net"

	"sync"
	"time"

	//envoy_api_v2_auth "github.com/envoyproxy/go-control-plane/envoy/api/v2/auth"
	//envoy_api_v2_core "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	//core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	//auth "github.com/envoyproxy/go-control-plane/envoy/extensions/transport_sockets/tls/v3"
	//"github.com/envoyproxy/go-control-plane/pkg/cache/types"

	//"github.com/golang/protobuf/ptypes"
	cachev3 "github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	discoverygrpc "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	serverv3 "github.com/envoyproxy/go-control-plane/pkg/server/v3"

	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"

	configure "fireside/pkg/configure"
)

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
	log.Infof("OnStreamRequest %v", r.TypeUrl)
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

type Callbacks struct {
	Signal   chan struct{}
	Debug    bool
	Fetches  int
	Requests int
	mu       sync.Mutex
}

const grpcMaxConcurrentStreams = 1000000

type XdsServer struct {
        Cache  cachev3.SnapshotCache
	Cb     *Callbacks
	Ctx    context.Context
	Port   uint
	Server serverv3.Server
}

func NewXdsServer(config *configure.Config) *XdsServer {
	ctx := context.Background()

	log.Info("Starting control plane for Envoy xDS")

	signal := make(chan struct{})
	cb := &Callbacks{
		Signal:   signal,
		Fetches:  0,
		Requests: 0,
	}
	cache := cachev3.NewSnapshotCache(true, cachev3.IDHash{}, nil)

	port := config.Inputs.Envoy.Xds.Server.Port
	srv := serverv3.NewServer(ctx, cache, cb)

	return &XdsServer{Cache: cache, Cb: cb, Ctx: ctx, Port: port, Server: srv}
}

// RunManagementServer starts an xDS server at the given port.
func (xc *XdsServer) RunManagementServer() {
	var grpcOptions []grpc.ServerOption
	grpcOptions = append(grpcOptions, grpc.MaxConcurrentStreams(grpcMaxConcurrentStreams))
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

func ServeEnvoyXds(config *configure.Config) {
	xdss := NewXdsServer(config)
	// start the xDS server
	go xdss.RunManagementServer()
	// Create a ticket for periodic refresh of state
	var refreshSeconds int = 5
	refreshInterval := time.Duration(refreshSeconds)
	refreshTicker := time.NewTicker(refreshInterval * time.Second)
	// Create a common 'quit' channel for stopping ticker(s) as needed
	quit := make(chan struct{})
	go func() {
		for {
			select {
			case <- refreshTicker.C:
				nodes := xdss.ListStatusKeys()
				for _, node := range nodes {
					xdss.GetNodeInfo(node)
				}
			case <- quit:
				refreshTicker.Stop()
				return
			}
		}
	}()

	<-xdss.Cb.Signal
}
