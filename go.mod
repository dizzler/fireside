module fireside

go 1.15

require (
	github.com/aws/aws-sdk-go v1.35.35
	github.com/dailyburn/ratchet v0.0.0
	github.com/dailyburn/ratchet/util v0.0.0
	github.com/envoyproxy/go-control-plane v0.9.7
	github.com/gofrs/uuid v3.3.0+incompatible // indirect
	github.com/golang/protobuf v1.4.2
	github.com/kisielk/sqlstruct v0.0.0-20201105191214-5f3e10d3ab46 // indirect
	github.com/pkg/sftp v1.12.0 // indirect
	github.com/qntfy/jsonparser v1.0.2 // indirect
	github.com/qntfy/kazaam v3.4.8+incompatible // indirect
	github.com/sirupsen/logrus v1.7.0
	google.golang.org/grpc v1.33.2
	gopkg.in/qntfy/kazaam.v3 v3.4.8
	gopkg.in/yaml.v2 v2.4.0
)

replace (
	github.com/dailyburn/ratchet => ./ratchet
	github.com/dailyburn/ratchet/util => ./ratchet/util
)
