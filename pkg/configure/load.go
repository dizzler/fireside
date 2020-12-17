package configure

import log "github.com/sirupsen/logrus"

func LoadConfig() (*Config, string) {
    // Get the path of the config file from command-line flag
    configPath, runMode, err := ParseFlags()
    if err != nil { log.Fatal(err) }

    // Create a new Config struct from config file inputs
    log.Info("loading FireSide config from file = " + configPath)
    config, err := NewConfig(configPath)
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
