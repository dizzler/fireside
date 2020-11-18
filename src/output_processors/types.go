package output_processors

// OutputConfig is a struct to store configuration for pipeline output processors
type OutputConfig struct {
	AWS       *AwsOutputConfig
	CheckCert bool
}

type OutputNotify struct {
	Cleanup         bool      // delete local archive file after upload success
	OutputTags      []string  // list of tags for output processor routing and/or filtering
	UploadFilePaths []string  // list of archive file paths to upload to object storage
	UploadSubpath   string    // subpath to prepend to file name||path; appended after output "BasePath"
	UseBasename     bool      // upload the file without copying local directory structure
}

type OutputReceiver struct {
	NotifyChan chan OutputNotify
}

// AwsOutputConfig is a struct to store configuration for AWS-type output processors
type AwsOutputConfig struct {
	Region          string
	AccessKeyID     string
	SecretAccessKey string
	S3BasePath      string
	S3Bucket        string
}
