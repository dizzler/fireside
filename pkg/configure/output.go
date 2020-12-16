package configure

// AwsOutputConfig is a struct to store configuration for AWS-type output processors
type AwsOutputConfig struct {
    Profile         string
    Region          string
    AccessKeyID     string
    SecretAccessKey string
    S3BasePath      string
    S3Bucket        string
}

// OutputConfig is a struct to store configuration for pipeline output processors
type OutputConfig struct {
    AWS       *AwsOutputConfig
    CheckCert bool
}
