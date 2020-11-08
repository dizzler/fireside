package main

import (
        "context"
        "flag"
        "fmt"
        "google.golang.org/grpc"
        "net"
	"os"

        accesslog "github.com/envoyproxy/go-control-plane/envoy/service/accesslog/v2"
        envoy_als "envoy_accesslog_service"
        log "github.com/sirupsen/logrus"
)

const (
	grpcMaxConcurrentStreams = 1000000
)

var (
        alsPort uint
        debug   bool
        localhost = "127.0.0.1"
)

func init() {
        flag.UintVar(&alsPort, "alsPort", 5446, "Listen port for Access Log Server")
        flag.BoolVar(&debug, "debug", true, "Use debug logging")
}

type logger struct {}

// RunAccessLogServer starts an gRPC server for receiving accesslogs from Envoy
func RunAccessLogServer(ctx context.Context, als *envoy_als.AccessLogService, port uint) {
        grpcServer := grpc.NewServer()
        lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
        if err != nil {
                log.WithError(err).Fatal("failed to listen")
        }

        accesslog.RegisterAccessLogServiceServer(grpcServer, als)
        log.WithFields(log.Fields{"port": port}).Info("access log server listening")

        go func() {
                if err = grpcServer.Serve(lis); err != nil {
                        log.Error(err)
                }
        }()
        <-ctx.Done()

        grpcServer.GracefulStop()
}

func main() {
        flag.Parse()
	if debug {
		log.SetLevel(log.DebugLevel)
	}
	ctx := context.Background()

	log.Printf("Starting FireSide control plane for Envoy")

	als := &envoy_als.AccessLogService{}
	go RunAccessLogServer(ctx, als, alsPort)

	alschan := make(chan struct{})
	<-alschan
	os.Exit(0)
}

