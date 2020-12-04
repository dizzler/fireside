package main

import (
    "encoding/json"
    "os"

    configure "fireside/pkg/configure"
    envoy_xds "fireside/pkg/envoy/xds"
    log "github.com/sirupsen/logrus"
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
    log.Info("Creating TLS Trust Domains for Envoy TLS 'secrets'")
    for _, policy := range config.Policies {
        tlsTrustDomain, err := envoy_xds.MakeTlsTrustDomains(policy.Config.Secrets)
        if err != nil {
            log.WithError(err).Fatal("failure encountered while creating TLS Trust Domain")
        }
        if config.Logging.Debug {
            tddJson, merr := json.Marshal(tlsTrustDomain)
	    if merr != nil {
                log.WithError(merr).Fatal("failed to marshal json for TLS Trust Domain")
            }
            log.Debug(string(tddJson))
        }
    }
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
