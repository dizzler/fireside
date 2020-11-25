package envoy_proxy_provider

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	//"io/ioutil"
	"net"
	//"os"

	"sync"
	//"sync/atomic"
	//"time"

	//envoy_api_v2_auth "github.com/envoyproxy/go-control-plane/envoy/api/v2/auth"
	//envoy_api_v2_core "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	//core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	//auth "github.com/envoyproxy/go-control-plane/envoy/extensions/transport_sockets/tls/v3"

	//"github.com/golang/protobuf/ptypes"

	//"github.com/envoyproxy/go-control-plane/pkg/cache/types"
	cachev3 "github.com/envoyproxy/go-control-plane/pkg/cache/v3"

	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"

	discoverygrpc "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"

	serverv3 "github.com/envoyproxy/go-control-plane/pkg/server/v3"
)

var (
	debug       bool
	onlyLogging bool
	withALS     bool

	localhost = "127.0.0.1"

	port        uint
	gatewayPort uint
	alsPort     uint

	mode string

	version int32

	cache cachev3.SnapshotCache

	strSlice = []string{"www.bbc.com", "www.yahoo.com", "blog.salrashid.me"}
)

const (
	XdsCluster = "xds_cluster"
	Ads        = "ads"
	Xds        = "xds"
	Rest       = "rest"
)

func init() {
	flag.BoolVar(&debug, "debug", true, "Use debug logging")
	flag.UintVar(&port, "port", 18000, "Management server port")
	flag.UintVar(&gatewayPort, "gateway", 18001, "Management server port for HTTP gateway")
	flag.StringVar(&mode, "ads", Ads, "Management server type (ads only now)")
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

// RunManagementServer starts an xDS server at the given port.
func RunManagementServer(ctx context.Context, server serverv3.Server, port uint) {
	var grpcOptions []grpc.ServerOption
	grpcOptions = append(grpcOptions, grpc.MaxConcurrentStreams(grpcMaxConcurrentStreams))
	grpcServer := grpc.NewServer(grpcOptions...)

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		log.WithError(err).Fatal("failed to listen")
	}

	// register services
	discoverygrpc.RegisterAggregatedDiscoveryServiceServer(grpcServer, server)

	// NOT used since we run ADS
	// endpointservice.RegisterEndpointDiscoveryServiceServer(grpcServer, server)
	// clusterservice.RegisterClusterDiscoveryServiceServer(grpcServer, server)
	// routeservice.RegisterRouteDiscoveryServiceServer(grpcServer, server)
	// listenerservice.RegisterListenerDiscoveryServiceServer(grpcServer, server)

	log.WithFields(log.Fields{"port": port}).Info("management server listening")
	go func() {
		if err = grpcServer.Serve(lis); err != nil {
			log.Error(err)
		}
	}()
	<-ctx.Done()

	grpcServer.GracefulStop()
}

func ServeEnvoyXds(port uint) {
	ctx := context.Background()

	log.Info("Starting control plane for Envoy xDS")

	signal := make(chan struct{})
	cb := &Callbacks{
		Signal:   signal,
		Fetches:  0,
		Requests: 0,
	}
	cache = cachev3.NewSnapshotCache(true, cachev3.IDHash{}, nil)

	srv := serverv3.NewServer(ctx, cache, cb)

	// start the xDS server
	go RunManagementServer(ctx, srv, port)

	<-signal

	status_keys := cache.GetStatusKeys()
	for _, s_key := range status_keys {
		log.Info("Getting info for status key " + s_key)
		info := cache.GetStatusInfo(s_key)
		node := info.GetNode()
		if len(node.Id) > 0 {
			log.Info("Got info for Envoy node ID " + node.Id)
		} else {
			log.Error("Failed to get Envoy node info and/or ID")
		}
		nodeJson, err := json.Marshal(node)
		if err != nil {
			log.Error(err)
		}
		log.Debug(fmt.Println(string(nodeJson)))
	}
}
