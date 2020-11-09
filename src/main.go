package main

import (
        "flag"
        "os"

        my_envoy_accesslog "envoy_accesslog"
        log "github.com/sirupsen/logrus"

        "github.com/dailyburn/ratchet"
        "github.com/dailyburn/ratchet/processors"
)

var (
        alsPort    uint
        debug      bool
        localhost  string = "127.0.0.1"
	outputFile string
)

func init() {
        flag.UintVar(&alsPort, "alsPort", 5446, "Listen port for Access Log Server")
        flag.BoolVar(&debug, "debug", true, "Use debug logging")
	//flag.StringVar(&outputFile, "outputFile", "/tmp/fireside_access_logs.out", "Send pipeline output to a file for troubleshooting")
}

func main() {
        flag.Parse()
	if debug {
		log.SetLevel(log.DebugLevel)
	}

	// Initialize the processors that will be used in the data processing pipeline
        envoyAccesslogProc := my_envoy_accesslog.NewEnvoyAccesslogReader(alsPort)
	passthroughProc := processors.NewPassthrough()
	iowriterProc := processors.NewIoWriter(os.Stdout)

	// Create a new Pipeline using the initialized processors
	pipeline := ratchet.NewPipeline(envoyAccesslogProc, passthroughProc, iowriterProc)

	// Run the Pipeline and wait for either an error or nil to be returned
	err := <-pipeline.Run()
	if err != nil {
                log.WithError(err).Fatal("error in data processing pipeline")
	}

	cc := make(chan struct{})
	<-cc
	os.Exit(0)
}
