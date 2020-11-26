package fireside

import (
	configure "fireside/pkg/configure"
        envoy "fireside/pkg/envoy"
	outproc "fireside/pkg/output_processors"
	transform "fireside/pkg/transformers"

        log "github.com/sirupsen/logrus"

        "github.com/dailyburn/ratchet"
)

var (
        awsCheckCerts   bool = false
        awsOutputConfig *outproc.AwsOutputConfig
        outputConfig    *outproc.OutputConfig
)

func CreateEventPipeline(config *configure.Config) {
        // Set the various *Config values used throughout the data processing pipeline
        awsOutputConfig = &outproc.AwsOutputConfig{
                Region: config.Outputs.AWS.Region,
                AccessKeyID: config.Outputs.AWS.AccessKeyId,
                SecretAccessKey: config.Outputs.AWS.SecretAccessKey,
                S3BasePath: config.Outputs.AWS.S3.BasePath,
		S3Bucket: config.Outputs.AWS.S3.Bucket}
        outputConfig = &outproc.OutputConfig{
                AWS: awsOutputConfig,
                CheckCert: awsCheckCerts}

	// Initialize the data extraction/input processors for pipeline
        envoyAccesslogProc := envoy.NewEnvoyAccesslogReader(config.Inputs.Envoy.Accesslog.Server.Port)

        // Initialize the transformation/enrichment processors for the pipeline
	var transformerSpec string = ""
	transformerProc := transform.NewFormatter(transformerSpec)

	// Initialize the loading/exporting processors for the pipeline
	// Send pipeline output to a directory on the local filesystem
	cacheWriterProc := outproc.NewFsCacheWriter(
		config.Outputs.Cache.Events.Directory,
		config.Outputs.Cache.Events.Prefix,
		outputConfig)

        // Create a new pipeline using the initialized processors
	pipeline := ratchet.NewPipeline(envoyAccesslogProc, transformerProc, cacheWriterProc)

	// Run the data processing pipeline and wait for either an error or nil to be returned
	err := <-pipeline.Run()
	if err != nil {
                log.WithError(err).Fatal("error in data processing pipeline")
	}
}
