package configure

import (
    "flag"
    "fmt"
    "os"

    log "github.com/sirupsen/logrus"
)

func LoadConfig() (*Config, string) {
    // Get the path of the config file from command-line flag
    configPath, runMode, err := ParseFlags()
    if err != nil { log.Fatal(err) }

    // Create a new Config struct from config file inputs
    log.Info("loading FireSide config from file = " + configPath)
    config, err := NewConfig(configPath)
    if err != nil { log.Fatal(err) }

    // prefer a non-empty environment var, if available, to set logging format
    var logFmt string = os.Getenv(EnvKeyLoggingFormat)
    // default to using config setting for logging format
    if len(logFmt) == 0 {
        logFmt = config.Logging.Format
    }
    // set the logging format
    switch logFmt {
    case LoggingFormatJson:
        // format log output as JSON instead of the default ASCII formatter.
        log.SetFormatter(&log.JSONFormatter{})
    case LoggingFormatText:
        // format log output as (default) text
        log.SetFormatter(&log.TextFormatter{})
    default:
        // format log output as (default) text
        log.SetFormatter(&log.TextFormatter{})
    }

    // prefer a non-empty environment var, if available, to set logging level
    var levelStr string = os.Getenv(EnvKeyLoggingLevel)
    if len(levelStr) == 0 && len(config.Logging.Level) > 0 {
        // use config value to set logging level
        levelStr = config.Logging.Level
    } else {
        // default to logging at level="info"
        levelStr = "info"
    }
    // convert/parse the logging level string into an actual log.Level
    level, loaderr := log.ParseLevel(levelStr)
    if loaderr != nil { log.Fatal(loaderr) }
    // set the logging level
    log.SetLevel(level)
    log.Info("logging FireSide runtime events at (or above) level = " + levelStr)

    // return the parse config, plus a string for the run mode
    return config, runMode
}

// ParseFlags will create and parse the CLI flags
// and return the path to be used elsewhere
func ParseFlags() (string, string, error) {
    // string that contains the configured configuration path
    var (
        configPath string
        runMode    string
    )

    // setup a CLI flag called "-config" to allow users
    // to supply the configuration file
    flag.StringVar(&configPath, "config", "/etc/fireside/config.yaml", "Filesystem path of bootstrap config file")

    // Setup a CLI flag called "-mode" in order to allow users to
    // change the operating / running "mode" for the fireside executable
    flag.StringVar(&runMode, "mode", "server", "Operational mode in which to run ; set to one of: 'server', 'ca'")

    // parse the CLI flags
    flag.Parse()

    // validate the path first
    if err := ValidateConfigPath(configPath); err != nil {
        return "", "", err
    }

    switch runMode {
    case "ca","server":
        log.Info("running FireSide with mode = " + runMode)
    default:
        log.Fatal("unsupported value for running 'mode' : " + runMode)
    }

    // Return the configuration path
    return configPath, runMode, nil
}

// ValidateConfigPath just makes sure, that the path provided is a file,
// that can be read
func ValidateConfigPath(path string) error {
    s, err := os.Stat(path)
    if err != nil {
        return err
    }
    if s.IsDir() {
        return fmt.Errorf("'%s' is a directory, not a normal file", path)
    }
    return nil
}
