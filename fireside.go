package main

import (
    "os"

    log "github.com/sirupsen/logrus"

    "fireside/pkg/configure"
    "fireside/pkg/run"
)

var (
    config  *configure.Config
    runMode string
)

func init() {
    // gather runtime configuration
    config, runMode = configure.LoadConfig()
}

func main() {
    // run in the selected mode
    switch runMode {
    case configure.RunModeCa:
        run.RunModeCa(config)
    case configure.RunModeServer:
        run.RunModeServer(config)
    default:
        log.Fatal("unsupported value for run 'mode' = " + runMode)
    }

    // exit with a clean return code
    os.Exit(0)
}
