package output_processors

import (
	"errors"
	"log"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sts"
)

// NewAwsClient returns a new output.Client for accessing the AWS API.
func NewAwsClient(config *OutputConfig) (*Client, error) {

	if config.AWS.AccessKeyID != "" && config.AWS.SecretAccessKey != "" && config.AWS.Region != "" {
		os.Setenv("AWS_ACCESS_KEY_ID", config.AWS.AccessKeyID)
		os.Setenv("AWS_SECRET_ACCESS_KEY", config.AWS.SecretAccessKey)
		os.Setenv("AWS_DEFAULT_REGION", config.AWS.Region)
	}

	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(config.AWS.Region)},
	)
	if err != nil {
		log.Printf("[ERROR] : AWS - %v\n", "Error while creating AWS Session")
		return nil, errors.New("Error while creating AWS Session")
	}

	_, err = sts.New(session.New()).GetCallerIdentity(&sts.GetCallerIdentityInput{})
	if err != nil {
		log.Printf("[ERROR] : AWS - %v\n", "Error while getting AWS Token")
		return nil, errors.New("Error while getting AWS Token")
	}

	return &Client{
		OutputType:      "AWS",
		Config:          config,
		AwsSession:      sess,
	}, nil
}
