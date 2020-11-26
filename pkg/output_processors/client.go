package fireside

import (
	"github.com/aws/aws-sdk-go/aws/session"
)

// Client communicates with the different API.
type Client struct {
	OutputType   string
	Config       *OutputConfig
	AwsSession   *session.Session
}
