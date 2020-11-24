package main

import (
        log "github.com/sirupsen/logrus"
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

	CreateEventPipeline(config)
}
