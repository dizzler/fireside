package main

import (
	configure "fireside/pkg/configure"
        envoy_xds "fireside/pkg/envoy/xds"
        log "github.com/sirupsen/logrus"
	"os"
	pipeline "fireside/pkg/pipeline"
)

const (
	runModeCa     = "ca"
	runModeServer = "server"
)

func RunConfig() (*configure.Config, string) {
	// Get the path of the config file from command-line flag
        configPath, runMode, err := configure.ParseFlags()
	if err != nil { log.Fatal(err) }

	// Create a new Config struct from config file inputs
	config, err := configure.NewConfig(configPath)
	if err != nil { log.Fatal(err) }

	if config.Logging.Debug {
		log.SetLevel(log.DebugLevel)
		log.Info("Debug logging enabled")
	}
	return config, runMode
}

func RunModeCa(config *configure.Config) {
	log.Debug("running RunModeCa function")
	return
}

func RunModeServer(config *configure.Config) {
	log.Debug("running RunModeServer function")
	// create a data processing pipeline for events
	go pipeline.CreateEventPipeline(config)

	// serve dynamic resource configurations to Envoy nodes
	go envoy_xds.ServeEnvoyXds(config)

	cc := make(chan struct{})
	<-cc

	return
}

func main() {
	// gather runtime configuration
	Config, RunMode := RunConfig()

	// run in the selected mode
	switch RunMode {
	case runModeCa:
		RunModeCa(Config)
	case runModeServer:
		RunModeServer(Config)
	}

	// exit with a clean return code
	os.Exit(0)
}
