package envoy_proxy_provider

import (
	"context"
	"fmt"
	"net"
	"github.com/dailyburn/ratchet/data"
        "google.golang.org/grpc"

        accesslog "github.com/envoyproxy/go-control-plane/envoy/service/accesslog/v2"
        log "github.com/sirupsen/logrus"
)

// EnvoyAccesslogReader receives data from Envoy network proxies through a gRPC server.
// We have to set a placeholder field - if we leave this as an empty struct we get some properties
// for comparison and memory addressing that are not desirable and cause comparison bugs
// (see: http://dave.cheney.net/2014/03/25/the-empty-struct)
type EnvoyAccesslogReader struct {
	AccessLogService  *AccessLogService
        AlsPort           uint
        Ctx               context.Context
}

// NewEnvoyAccesslogReader instantiates a new instance of EnvoyAccesslogReader
func NewEnvoyAccesslogReader(alsPort uint) *EnvoyAccesslogReader {
        ctx := context.Background()
	als := &AccessLogService{}

	als.alc = make(chan data.JSON)

        log.Printf("Starting gRPC server for receiving Envoy access logs")

	return &EnvoyAccesslogReader{AccessLogService: als, AlsPort: alsPort, Ctx: ctx}
}

// ProcessData sends whatever it receives to the outputChan
func (r *EnvoyAccesslogReader) ProcessData(d data.JSON, outputChan chan data.JSON, killChan chan error) {
	go RunAccessLogServer(r.AccessLogService, r.AlsPort, r.Ctx)
}

// Finish - see interface for documentation.
func (r *EnvoyAccesslogReader) Finish(outputChan chan data.JSON, killChan chan error) {
        for srcEvent := range r.AccessLogService.alc {
		outputChan <- srcEvent
        }
}

func (r *EnvoyAccesslogReader) String() string {
	return "EnvoyAccesslogReader"
}

// RunAccessLogServer starts an gRPC server for receiving accesslogs from Envoy
func RunAccessLogServer(als *AccessLogService, port uint, ctx context.Context) {
        grpcServer := grpc.NewServer()
        lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
        if err != nil {
                log.WithError(err).Fatal("failed to listen for gRPC Envoy access log server")
        }

        accesslog.RegisterAccessLogServiceServer(grpcServer, als)
        log.WithFields(log.Fields{"port": port}).Info("gRPC server listening for Envoy access logs")

        go func() {
                err = grpcServer.Serve(lis)
                if err = grpcServer.Serve(lis); err != nil {
                        log.Error(err)
                }
        }()
        <-ctx.Done()

        grpcServer.GracefulStop()
}
