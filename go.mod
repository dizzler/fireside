module main

go 1.14

require (
	envoy_accesslog v0.0.0
	github.com/aws/aws-sdk-go v1.35.24 // indirect
	github.com/dailyburn/bigquery v0.0.0-20171116202005-b6f18972580e // indirect
	github.com/dailyburn/ratchet v0.0.0
	github.com/dailyburn/ratchet/util v0.0.0
	github.com/envoyproxy/go-control-plane v0.9.4
	github.com/golang/protobuf v1.4.2 // indirect
	github.com/jlaffaye/ftp v0.0.0-20201021201046-0de5c29d4555 // indirect
	github.com/kisielk/sqlstruct v0.0.0-20201105191214-5f3e10d3ab46 // indirect
	github.com/pkg/sftp v1.12.0 // indirect
	github.com/sirupsen/logrus v1.6.0
	google.golang.org/api v0.35.0 // indirect
	google.golang.org/grpc v1.31.1
)

replace envoy_accesslog => ./src/envoy_accesslog

replace github.com/dailyburn/ratchet => ./src/ratchet

replace github.com/dailyburn/ratchet/util => ./src/ratchet/util
