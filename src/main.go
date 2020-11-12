package main

import (
        "flag"
        "os"

        eal "envoy_accesslog"
	outproc "output_processors"
	transform "transformers"

        log "github.com/sirupsen/logrus"

        "github.com/dailyburn/ratchet"
        //"github.com/dailyburn/ratchet/processors"
)

var (
        alsPort         uint
        debug           bool
        localhost       string = "127.0.0.1"
        awsCheckCerts   bool
	awsID           string
	awsRegion       string
	awsSecret       string
        awsS3BasePath   string
	awsS3Bucket     string
        awsOutputConfig *outproc.AwsOutputConfig
	fsCacheDir      string
	fsCachePrefix   string
        outputConfig    *outproc.OutputConfig
)

func check(e error) {
       if e != nil {
	   panic(e)
       }
}

func init() {
        flag.UintVar(&alsPort, "alsPort", 5446, "Listen port for Access Log Server")
        flag.BoolVar(&debug, "debug", false, "Use debug logging")
	flag.StringVar(&awsID, "awsID", "", "Equivalent of (defaults to) AWS_ACCESS_KEY_ID")
	flag.StringVar(&awsRegion, "awsRegion", "", "Equivalent of (defaults to) AWS_DEFAULT_REGION")
	flag.StringVar(&awsS3BasePath, "awsS3BasePath", "firesideS3test", "")
	flag.StringVar(&awsS3Bucket, "awsS3Bucket", "", "Send pipeline output to an AWS S3 bucket")
	flag.StringVar(&awsSecret, "awsSecret", "", "Equivalent of (defaults to) AWS_SECRET_ACCESS_KEY")
	flag.StringVar(&fsCacheDir, "fsCacheDir", "/tmp/fireside/cache", "Directory for temporarily storing cached data")
	flag.StringVar(&fsCachePrefix, "fsCachePrefix", "fireside-event-cache", "Prefix used in filenames for cached data")
}

func main() {
        flag.Parse()
	if debug {
		log.SetLevel(log.DebugLevel)
	}

        // Set the various *Config values used throughout the data processing pipeline
        awsOutputConfig = &outproc.AwsOutputConfig{
                Region: awsRegion,
                AccessKeyID: awsID,
                SecretAccessKey: awsSecret,
                S3BasePath: awsS3BasePath,
		S3Bucket: awsS3Bucket}
        outputConfig = &outproc.OutputConfig{
                AWS: awsOutputConfig,
                CheckCert: awsCheckCerts}

	// Initialize the data extraction/input processors for pipeline
        envoyAccesslogProc := eal.NewEnvoyAccesslogReader(alsPort)

        // Initialize the transformation/enrichment processors for the pipeline
	var transformerSpec string = ""
	transformerProc := transform.NewFormatter(transformerSpec)
	//passthroughProc := processors.NewPassthrough()

	// Initialize the loading/exporting processors for the pipeline, then...
        // Create a new pipeline using the initialized processors
        // Send pipeline output to an S3 bucket
	//s3WriterProc := outproc.NewS3Writer(outputConfig)
	//pipeline := ratchet.NewPipeline(envoyAccesslogProc, passthroughProc, s3WriterProc)
	// Send pipeline output to a directory on the local filesystem
	cacheWriterProc := outproc.NewFsCacheWriter(fsCacheDir, fsCachePrefix)
	pipeline := ratchet.NewPipeline(envoyAccesslogProc, transformerProc, cacheWriterProc)
	//pipeline := ratchet.NewPipeline(envoyAccesslogProc, passthroughProc, cacheWriterProc)

	// Run the data processing pipeline and wait for either an error or nil to be returned
	err := <-pipeline.Run()
	if err != nil {
                log.WithError(err).Fatal("error in data processing pipeline")
	}

	cc := make(chan struct{})
	<-cc
	os.Exit(0)
}
