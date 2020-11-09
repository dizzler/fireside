package envoy_accesslog

import (
        "encoding/json"
	"fmt"
	"sync"
	"time"

	alf "github.com/envoyproxy/go-control-plane/envoy/data/accesslog/v2"
	als "github.com/envoyproxy/go-control-plane/envoy/service/accesslog/v2"
	"github.com/dailyburn/ratchet/data"
	log "github.com/sirupsen/logrus"
)

const (
        eventCategoryHttp = "proxy_http"
        eventCategoryTcp = "proxy_tcp"
        eventSrcGrpc = "envoy_grpc"
        eventType    = "envoy_accesslog"
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
	entries  []data.JSON
	mu       sync.Mutex
	alc      chan data.JSON
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

func (svc *AccessLogService) log(entry data.JSON) {
	svc.mu.Lock()
	defer svc.mu.Unlock()
	svc.entries = append(svc.entries, entry)

	// write each accesslog entry to a channel
	svc.alc <- entry

	// Log each JSON message to stdout when debug logging is enabled
        log.Debug(fmt.Println(string(entry)))
}

// Dump releases the collected log entries and clears the log entry list.
func (svc *AccessLogService) Dump(f func(data.JSON)) {
	svc.mu.Lock()
	defer svc.mu.Unlock()
	for _, entry := range svc.entries {
		f(entry)
	}
	svc.entries = nil
}

// StreamAccessLogs implements the access log service.
func (svc *AccessLogService) StreamAccessLogs(stream als.AccessLogService_StreamAccessLogsServer) error {
	var logName string

	for {
		msg, err := stream.Recv()
		if err != nil {
			continue
		}
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
                                                EventCategory: eventCategoryHttp,
                                                EventSource: eventSrcGrpc,
                                                EventType: eventType,
                                                LogName: logName,
                                                LogTimestamp: time.Now().Format(time.RFC3339),
                                                Request: req,
                                                Response: resp}
                                        logJson, err := json.Marshal(logMsg)
					if err != nil {
                                                log.Error(err)
					}
                                        svc.log(logJson)
				}
			}
		case *als.StreamAccessLogsMessage_TcpLogs:
			for _, entry := range entries.TcpLogs.LogEntry {
				if entry != nil {
					common := entry.CommonProperties
					if common == nil {
						common = &alf.AccessLogCommon{}
					}
					var req = &alf.HTTPRequestProperties{}
					var resp = &alf.HTTPResponseProperties{}
                                        logMsg := &EnvoyAccessLogJson{
                                                CommonProperties: common,
                                                EventCategory: eventCategoryTcp,
                                                EventSource: eventSrcGrpc,
                                                EventType: eventType,
                                                LogName: logName,
                                                LogTimestamp: time.Now().Format(time.RFC3339),
                                                Request: req,
                                                Response: resp}
                                        logJson, err := json.Marshal(logMsg)
					if err != nil {
                                                log.Error(err)
					}
                                        svc.log(logJson)
				}
			}
		}
	}
}
