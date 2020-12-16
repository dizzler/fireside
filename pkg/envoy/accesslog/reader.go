package accesslog

import (
    "context"
    "encoding/json"
    "fmt"
    "net"
    "google.golang.org/grpc"

    "fireside/pkg/configure"
    "fireside/pkg/pipeline/data"

    alv2 "github.com/envoyproxy/go-control-plane/envoy/service/accesslog/v2"
    log "github.com/sirupsen/logrus"
)

// EnvoyAccesslogReader receives data from Envoy network proxies through a gRPC server.
// We have to set a placeholder field - if we leave this as an empty struct we get some properties
// for comparison and memory addressing that are not desirable and cause comparison bugs
// (see: http://dave.cheney.net/2014/03/25/the-empty-struct)
type EnvoyAccesslogReader struct {
    AccessLogService *AccessLogService
    AlsPort          uint
    Ctx              context.Context
}

// NewEnvoyAccesslogReader instantiates a new instance of EnvoyAccesslogReader
func NewEnvoyAccesslogReader(alsPort uint) *EnvoyAccesslogReader {
    ctx := context.Background()
    als := &AccessLogService{}

    als.alc = make(chan data.JSON)

    log.Printf("starting gRPC server for receiving Envoy access logs")

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

    alv2.RegisterAccessLogServiceServer(grpcServer, als)
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

// AccessLogService buffers access logs from the remote Envoy nodes.
type AccessLogService struct {
    alc chan data.JSON
}

// StreamAccessLogs implements the access log service.
func (svc *AccessLogService) StreamAccessLogs(stream alv2.AccessLogService_StreamAccessLogsServer) error {
    var logName string

    log.Debug("Running StreamAccessLogs function for AccessLogService type")
    for {
        msg, err := stream.Recv()
        if err != nil {
	    continue
        }
        log.Debug("Received envoy accesslog message from gRPC stream")
        if msg.Identifier != nil {
	    logName = msg.Identifier.LogName
        }
        switch entries := msg.LogEntries.(type) {
        case *alv2.StreamAccessLogsMessage_HttpLogs:
            for _, entry := range entries.HttpLogs.LogEntry {
                if entry != nil {
                    // convert the source entry to JSON to avoid dealing with struct fields
                    entryJson, jerr := json.Marshal(entry)
                    if jerr != nil {
                        log.WithError(err).Error("failed to insert Envoy HTTP accesslog entry data into event wrapper")
                    } else {
                        eventWrapper := configure.NewFiresideEvent(
                            configure.EventCategoryHttp,
                            configure.EventTypeEnvoyAccesslog,
                            logName,
                            "",
                            configure.EventSrcGrpc)
                        eventJson, err := configure.InsertFiresideEventData(entryJson, eventWrapper)
                        if err != nil {
                            log.WithError(err).Error("failed to marshal Fireside Event JSON for Envoy HTTP accesslog entry")
                        }
                        // send the full JSON event to a channel for further processing
                        svc.alc <- eventJson
                    }
                }
            }
        case *alv2.StreamAccessLogsMessage_TcpLogs:
            for _, entry := range entries.TcpLogs.LogEntry {
                if entry != nil {
                    // convert the source entry to JSON to avoid dealing with struct fields
                    entryJson, jerr := json.Marshal(entry)
                    if jerr != nil {
                        log.WithError(err).Error("failed to insert Envoy TCP accesslog entry data into event wrapper")
                    } else {
                        eventWrapper := configure.NewFiresideEvent(
                            configure.EventCategoryTcp,
                            configure.EventTypeEnvoyAccesslog,
                            logName,
                            "",
                            configure.EventSrcGrpc)
                        eventJson, err := configure.InsertFiresideEventData(entryJson, eventWrapper)
                        if err != nil {
                            log.WithError(err).Error("failed to marshal Fireside Event JSON for Envoy TCP accesslog entry")
                        }
                        // send the full JSON event to a channel for further processing
                        svc.alc <- eventJson
                    }
                }
            }
        }
    }
}

type logger struct{}

func (logger logger) Infof(format string, args ...interface{}) {
    log.Infof(format, args...)
}

func (logger logger) Errorf(format string, args ...interface{}) {
    log.Errorf(format, args...)
}
