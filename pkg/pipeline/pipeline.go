package fireside

import (
    configure "fireside/pkg/configure"
    envoy_logs "fireside/pkg/envoy/accesslog"
    outproc "fireside/pkg/output_processors"
    transform "fireside/pkg/transformers"

    log "github.com/sirupsen/logrus"

    "github.com/dailyburn/ratchet"
)

var (
    awsCheckCerts   bool = false
    awsOutputConfig *configure.AwsOutputConfig
    outputConfig    *configure.OutputConfig
)

func CreateEventPipeline(config *configure.Config) {
    // Set the various *Config values used throughout the data processing pipeline
    awsOutputConfig = &configure.AwsOutputConfig{
        Profile: config.Outputs.AWS.Profile,
        Region: config.Outputs.AWS.Region,
        AccessKeyID: config.Outputs.AWS.AccessKeyId,
        SecretAccessKey: config.Outputs.AWS.SecretAccessKey,
        S3BasePath: config.Outputs.AWS.S3.BasePath,
        S3Bucket: config.Outputs.AWS.S3.Bucket}
    outputConfig = &configure.OutputConfig{
        AWS: awsOutputConfig,
        CheckCert: awsCheckCerts}

    // Initialize the data extraction/input processors for pipeline
    eventInEnvoy1 := envoy_logs.NewEnvoyAccesslogReader(config.Inputs.Envoy.Accesslog.Server.Port)

    // Initialize the transformation/enrichment processors for the pipeline
    var transformerSpec string = ""
    eventFmt1 := transform.NewFormatter(transformerSpec)

    // Initialize the loading/exporting processors for the pipeline
    // Send pipeline output to a directory on the local filesystem
    eventOut1 := outproc.NewFsCacheWriter(
        config.Outputs.Cache.Events.Directory,
        config.Outputs.Cache.Events.Prefix,
        outputConfig)

    // Create and validate the PipelineLayout for the complex, multi-phase "Events Pipeline"
    layout, layerr := ratchet.NewPipelineLayout(
        ratchet.NewPipelineStage(
            ratchet.Do(eventInEnvoy1).Outputs(eventFmt1),
        ),
        ratchet.NewPipelineStage(
            ratchet.Do(eventFmt1).Outputs(eventOut1),
        ),
        ratchet.NewPipelineStage(
            ratchet.Do(eventOut1),
        ),
    )
    if layerr != nil {
        log.WithError(layerr).Fatal("failed to validate pipeline layout prior to creating data processing pipeline")
    }

    // Create a new pipeline using the initialized processors
    pipeline := ratchet.NewBranchingPipeline(layout)

    // Run the data processing pipeline and wait for either an error or nil to be returned
    err := <-pipeline.Run()
    if err != nil {
        log.WithError(err).Fatal("error in data processing pipeline")
    }
}
