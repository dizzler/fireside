package main

import (
    "os"

    log "github.com/sirupsen/logrus"

    "fireside/pkg/configure"
    "fireside/pkg/run"
)

func main() {
    // gather runtime configuration
    Config, RunMode := configure.LoadConfig()

    // run in the selected mode
    switch RunMode {
    case configure.RunModeCa:
        run.RunModeCa(Config)
    case configure.RunModeServer:
        run.RunModeServer(Config)
    default:
        log.Fatal("unsupported value for run 'mode' = " + RunMode)
    }

    // exit with a clean return code
    os.Exit(0)
}
