package fireside

import (
    "context"
    "errors"
    "fmt"

    "google.golang.org/grpc"
    log "github.com/sirupsen/logrus"

    discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
)

type XdsStream struct {
        ctx       context.Context
        Requests  chan *discovery.DiscoveryRequest
        Responses chan *discovery.DiscoveryResponse
        nonce     int
        sendError bool
        grpc.ServerStream
}

func (stream *XdsStream) Context() context.Context {
        return stream.ctx
}

func (stream *XdsStream) Send(resp *discovery.DiscoveryResponse) error {
        // check that nonce is monotonically incrementing
        stream.nonce = stream.nonce + 1
        if resp.Nonce != fmt.Sprintf("%d", stream.nonce) {
                log.Errorf("Nonce => got %q, want %d", resp.Nonce, stream.nonce)
        }
        // check that version is set
        if resp.VersionInfo == "" {
                log.Error("VersionInfo => got none, want non-empty")
        }
        // check resources are non-empty
        if len(resp.Resources) == 0 {
                log.Error("Resources => got none, want non-empty")
        }
        // check that type URL matches in resources
        if resp.TypeUrl == "" {
                log.Error("TypeUrl => got none, want non-empty")
        }
        for _, res := range resp.Resources {
                if res.TypeUrl != resp.TypeUrl {
                        log.Errorf("TypeUrl => got %q, want %q", res.TypeUrl, resp.TypeUrl)
                }
        }
        stream.Responses <- resp
        if stream.sendError {
                return errors.New("send error")
        }
        return nil
}

func (stream *XdsStream) Recv() (*discovery.DiscoveryRequest, error) {
        req, more := <-stream.Requests
        if !more {
                return nil, errors.New("empty")
        }
        return req, nil
}
