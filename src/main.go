package main

import (
        epp "envoy_proxy_provider"
        log "github.com/sirupsen/logrus"
	"os"
)

func main() {
	// Get the path of the config file from command-line flag
        configPath, err := ParseFlags()
	if err != nil { log.Fatal(err) }

	// Create a new Config struct from config file inputs
	config, err := NewConfig(configPath)
	if err != nil { log.Fatal(err) }

	if config.Logging.Debug {
		log.SetLevel(log.DebugLevel)
		log.Info("Debug logging enabled")
	}

	go CreateEventPipeline(config)

	go epp.ServeEnvoyXds(config.Inputs.Envoy.Xds.Server.Port)

	cc := make(chan struct{})
	<-cc
	os.Exit(0)
}
