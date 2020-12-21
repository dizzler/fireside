package events

import (
    "fireside/pkg/configure"
    "fireside/pkg/envoy/accesslog"
    "fireside/pkg/pipeline/conduit"
    "fireside/pkg/pipeline/processors"

    log "github.com/sirupsen/logrus"
)

var (
    awsCheckCerts   bool = false
    awsOutputConfig *configure.AwsOutputConfig
    outputConfig    *configure.OutputConfig
)

func CreateEventsPipelines(config *configure.Config) {
    CreateEventsPipelineEnvoy(config)
    CreateEventsPipelineFalco(config)
}

func CreateEventsPipelineEnvoy(config *configure.Config) {
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

    ///////////////////////////////   pipeline1   ///////////////////////////////
    const p1 string = "pipeline1"
    // Initialize the data extraction/input processors for pipeline
    eventInEnvoy1 := accesslog.NewEnvoyAccesslogReader(config.Inputs.Envoy.Accesslog.Server.Port)

    // Initialize the transformation/enrichment processors for the pipeline
    var transformerSpec string = ""
    eventFmt1 := processors.NewFormatter(transformerSpec)

    // Initialize the loading/exporting processors for the pipeline
    // Send pipeline output to a directory on the local filesystem
    eventOut1 := processors.NewFsCacheWriter(
        config.Outputs.Cache.Events.Directory,
        configure.CachePrefixEnvoy,
        outputConfig,
        &config.Pipelines.Envoy.State)

    // Create and validate the pipeline1 Layout
    layout1, layerr1 := conduit.NewPipelineLayout(
        conduit.NewPipelineStage(
            conduit.Do(eventInEnvoy1).Outputs(eventFmt1),
        ),
        conduit.NewPipelineStage(
            conduit.Do(eventFmt1).Outputs(eventOut1),
        ),
        conduit.NewPipelineStage(
            conduit.Do(eventOut1),
        ),
    )
    if layerr1 != nil {
	log.WithError(layerr1).Fatal("failed to validate pipeline layout prior to creating data processing pipeline : " + p1)
    }

    // Create a new pipeline using the initialized processors
    pipeline1 := conduit.NewBranchingPipeline(layout1)

    // Run the data processing pipeline and wait for either an error or nil to be returned
    go func() {
        err1 := <-pipeline1.Run()
        if err1 != nil {
            log.WithError(err1).Fatal("error in data processing pipeline : " + p1)
        }
    }()
}

func CreateEventsPipelineFalco(config *configure.Config) {
    ///////////////////////////////   pipeline2   ///////////////////////////////
    const p2 string = "pipeline2"
    // Initialize the data processors for pipeline2
    eventInFalco2 := processors.NewFileReader(&config.Inputs.Files[0])
    eventTagger2 := processors.NewEventTagger(&config.Pipelines.Falco.Policies.EventTagger, &config.Pipelines.Falco.State)
    eventOut2 := processors.NewFsCacheWriter(
        config.Outputs.Cache.Events.Directory,
        configure.CachePrefixFalco,
        outputConfig,
        &config.Pipelines.Falco.State)

    // Create and validate the pipeline2 Layout
    layout2, layerr2 := conduit.NewPipelineLayout(
        conduit.NewPipelineStage(
            conduit.Do(eventInFalco2).Outputs(eventTagger2),
        ),
        conduit.NewPipelineStage(
            conduit.Do(eventTagger2).Outputs(eventOut2),
        ),
        conduit.NewPipelineStage(
            conduit.Do(eventOut2),
        ),
    )
    if layerr2 != nil {
        log.WithError(layerr2).Fatal("failed to validate pipeline layout prior to creating data processing pipeline : " + p2)
    }

    // Create a new pipeline using the initialized processors
    pipeline2 := conduit.NewBranchingPipeline(layout2)

    // Run the data processing pipeline and wait for either an error or nil to be returned
    go func() {
        err2 := <-pipeline2.Run()
        if err2 != nil {
            log.WithError(err2).Fatal("error in data processing pipeline : " + p2)
        }
    }()
}
