package main

import (
	configure "fireside/pkg/configure"
        envoy_xds "fireside/pkg/envoy/xds"
        log "github.com/sirupsen/logrus"
	"os"
	pipeline "fireside/pkg/pipeline"
)

func main() {
	// Get the path of the config file from command-line flag
        configPath, err := configure.ParseFlags()
	if err != nil { log.Fatal(err) }

	// Create a new Config struct from config file inputs
	config, err := configure.NewConfig(configPath)
	if err != nil { log.Fatal(err) }

	if config.Logging.Debug {
		log.SetLevel(log.DebugLevel)
		log.Info("Debug logging enabled")
	}

	go pipeline.CreateEventPipeline(config)

	go envoy_xds.ServeEnvoyXds(config)

	cc := make(chan struct{})
	<-cc
	os.Exit(0)
}
