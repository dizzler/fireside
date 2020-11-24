module main

go 1.14

require (
	envoy_accesslog v0.0.0
	github.com/creasty/defaults v1.5.1 // indirect
	github.com/dailyburn/bigquery v0.0.0-20171116202005-b6f18972580e // indirect
	github.com/dailyburn/ratchet v0.0.0
	github.com/dailyburn/ratchet/util v0.0.0
	github.com/envoyproxy/go-control-plane v0.9.4
	github.com/gofrs/uuid v3.3.0+incompatible // indirect
	github.com/golang/protobuf v1.4.2 // indirect
	github.com/jlaffaye/ftp v0.0.0-20201021201046-0de5c29d4555 // indirect
	github.com/kisielk/sqlstruct v0.0.0-20201105191214-5f3e10d3ab46 // indirect
	github.com/pkg/sftp v1.12.0 // indirect
	github.com/qntfy/jsonparser v1.0.2 // indirect
	github.com/qntfy/kazaam v3.4.8+incompatible // indirect
	github.com/sirupsen/logrus v1.6.0
	google.golang.org/api v0.35.0 // indirect
	google.golang.org/grpc v1.31.1
	gopkg.in/qntfy/kazaam.v3 v3.4.8 // indirect
	output_processors v0.0.0
	transformers v0.0.0
)

replace envoy_accesslog => ./src/envoy_accesslog

replace github.com/dailyburn/ratchet => ./src/ratchet

replace github.com/dailyburn/ratchet/util => ./src/ratchet/util

replace output_processors => ./src/output_processors

replace transformers => ./src/transformers
