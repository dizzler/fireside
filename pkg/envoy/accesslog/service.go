package fireside

import (
    "encoding/json"
    "time"

    configure "fireside/pkg/configure"

    alf "github.com/envoyproxy/go-control-plane/envoy/data/accesslog/v2"
    als "github.com/envoyproxy/go-control-plane/envoy/service/accesslog/v2"
    "github.com/dailyburn/ratchet/data"
    log "github.com/sirupsen/logrus"
)

type logger struct{}

func (logger logger) Infof(format string, args ...interface{}) {
    log.Infof(format, args...)
}
func (logger logger) Errorf(format string, args ...interface{}) {
    log.Errorf(format, args...)
}

// AccessLogService buffers access logs from the remote Envoy nodes.
type AccessLogService struct {
    alc chan data.JSON
}

// Defines the structure of Envoy access logs as JSON data
type EnvoyAccessLogJson struct {
    CommonProperties *alf.AccessLogCommon        `json:"common_properties"`
    EventCategory    string                      `json:"event_category"`
    EventSource      string                      `json:"event_source"`
    EventType        string                      `json:"event_type"`
    LogName          string                      `json:"log_name"`
    LogTimestamp     string                      `json:"log_timestamp"`
    Request          *alf.HTTPRequestProperties  `json:"request"`
    Response         *alf.HTTPResponseProperties `json:"response"`
}

// StreamAccessLogs implements the access log service.
func (svc *AccessLogService) StreamAccessLogs(stream als.AccessLogService_StreamAccessLogsServer) error {
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
        case *als.StreamAccessLogsMessage_HttpLogs:
            for _, entry := range entries.HttpLogs.LogEntry {
                if entry != nil {
                    common := entry.CommonProperties
                    req := entry.Request
                    resp := entry.Response
                    if common == nil {
                        common = &alf.AccessLogCommon{}
                    }
                    if req == nil {
                        req = &alf.HTTPRequestProperties{}
                    }
                    if resp == nil {
                        resp = &alf.HTTPResponseProperties{}
                    }
                    logMsg := &EnvoyAccessLogJson{
                        CommonProperties: common,
                        EventCategory: configure.EventCategoryHttp,
                        EventSource: configure.EventSrcGrpc,
                        EventType: configure.EventType,
                        LogName: logName,
                        LogTimestamp: time.Now().Format(time.RFC3339),
                        Request: req,
                        Response: resp,
                    }
                    logJson, err := json.Marshal(logMsg)
                    if err != nil {
                            log.Error(err)
                    }
                    svc.alc <- logJson
                }
            }
        case *als.StreamAccessLogsMessage_TcpLogs:
            for _, entry := range entries.TcpLogs.LogEntry {
                if entry != nil {
                    common := entry.CommonProperties
                    if common == nil {
                        common = &alf.AccessLogCommon{}
                    }
                    logMsg := &EnvoyAccessLogJson{
                        CommonProperties: common,
                        EventCategory: configure.EventCategoryTcp,
                        EventSource: configure.EventSrcGrpc,
                        EventType: configure.EventType,
                        LogName: logName,
                        LogTimestamp: time.Now().Format(time.RFC3339),
                    }
                    logJson, err := json.Marshal(logMsg)
                    if err != nil {
                        log.Error(err)
                    }
                    svc.alc <- logJson
                }
            }
        }
    }
}
