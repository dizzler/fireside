package main

import (
    "os"

    log "github.com/sirupsen/logrus"

    "fireside/pkg/configure"
    "fireside/pkg/envoy/xds"
    "fireside/pkg/pipeline/events"
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
    log.Info("loading FireSide config from file = " + configPath)
    config, err := configure.NewConfig(configPath)
    if err != nil { log.Fatal(err) }

    if len(config.Logging.Level) > 0 {
        level, err := log.ParseLevel(config.Logging.Level)
        if err != nil { log.Fatal(err) }
        // set the logging level for the entire application runtime
        log.SetLevel(level)
        log.Info("logging FireSide runtime events at (or above) level = " + config.Logging.Level)
    }
    return config, runMode
}

func RunModeCa(config *configure.Config) {
    log.Debug("running RunModeCa function")
    log.Info("creating TLS Trust Domains for Envoy TLS 'secrets'")
    for _, policy := range config.Policies {
        _, err := xds.MakeTlsTrustDomains(policy.Config.Secrets)
        if err != nil {
            log.WithError(err).Fatal("failure encountered while creating TLS Trust Domain")
        }
    }
    return
}

func RunModeServer(config *configure.Config) {
    log.Debug("running RunModeServer function")
    // create a data processing pipeline for events
    go events.CreateEventsPipelines(config)

    // serve dynamic resource configurations to Envoy nodes
    go xds.ServeEnvoyXds(config)

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
