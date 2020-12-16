package main

import (
    "os"

    configure "fireside/pkg/configure"
    envoy_xds "fireside/pkg/envoy/xds"
    log "github.com/sirupsen/logrus"
    eventpipe "fireside/pkg/pipeline/events"
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
        log.Info("debug logging enabled")
    }
    return config, runMode
}

func RunModeCa(config *configure.Config) {
    log.Debug("running RunModeCa function")
    log.Info("creating TLS Trust Domains for Envoy TLS 'secrets'")
    for _, policy := range config.Policies {
        _, err := envoy_xds.MakeTlsTrustDomains(policy.Config.Secrets)
        if err != nil {
            log.WithError(err).Fatal("failure encountered while creating TLS Trust Domain")
        }
    }
    return
}

func RunModeServer(config *configure.Config) {
    log.Debug("running RunModeServer function")
    // create a data processing pipeline for events
    go eventpipe.CreateEventPipelines(config)

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
    default:
        log.Fatal("unsupported value for run 'mode' = " + RunMode)
    }

    // exit with a clean return code
    os.Exit(0)
}
