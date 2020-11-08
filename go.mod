module main

go 1.14

require (
	envoy_accesslog_service v0.0.0
	github.com/envoyproxy/go-control-plane v0.9.4
	github.com/golang/protobuf v1.4.2 // indirect
	github.com/sirupsen/logrus v1.6.0
	google.golang.org/grpc v1.31.1
)

replace data => ./src/data

replace envoy_accesslog_service => ./src/envoy_accesslog_service
