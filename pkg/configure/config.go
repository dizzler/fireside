package fireside

import (
    "flag"
    "fmt"
    log "github.com/sirupsen/logrus"
    "os"
    "gopkg.in/yaml.v2"
)


// Config struct for webapp config
type Config struct {

    Logging struct {
	Debug bool `yaml:"debug"`
    } `yaml:"logging"`

    // configs for input processors and providers
    Inputs struct {
	Envoy struct {
	    Accesslog struct {
		// configs for the Envoy Accesslog (collector) server
		Server struct {
		    Host string `yaml:"host"`
		    Port uint   `yaml:"port"`
		} `yaml:"server"`
	    } `yaml:"accesslog"`
	    Xds struct {
		// configs for the Envoy xDS (control plane) server
		Server struct {
		    Host string `yaml:"host"`
		    Port uint   `yaml:"port"`
		} `yaml:"server"`
	    } `yaml:"xds"`
	} `yaml:"envoy"`
    } `yaml:"inputs"`

    // configs for transformer processors and providers
    Transformers struct {
	Enable bool `yaml:"enable"`
    } `yaml:"transformers"`

    // configs for output processors and providers
    Outputs struct {

        AWS struct {
            // optional ; equivalent of (defaults to) AWS_ACCESS_KEY_ID
            AccessKeyId     string `yaml:"access_key_id"`
            // REQUIRED if S3Bucket is set
            // equivalent of (defaults to) AWS_DEFAULT_REGION
            Region          string `yaml:"region"`
            // optional ; equivalent of (defaults to) AWS_SECRET_ACCESS_KEY
            SecretAccessKey string `yaml:"secret_access_key"`

            S3 struct {
                // base directory path within the target/upload S3 bucket
                BasePath      string `yaml:"base_path"`
                // name of the target S3 bucket
                Bucket        string `yaml:"bucket"`
            } `yaml:"s3"`
        } `yaml:"aws"`

	Cache struct {
	    Events struct {
		Directory string `yaml:"directory"`
		Prefix    string `yaml:"prefix"`
	    } `yaml:"events"`
	} `yaml:"cache"`

    } `yaml:"outputs"`

    // policies to be applied to managed nodes and/or data streams
    Policies []Policy `yaml:"policies"`
}

// NewConfig returns a new decoded Config struct
func NewConfig(configPath string) (*Config, error) {
    // Create config structure
    config := &Config{}

    // Open config file
    file, err := os.Open(configPath)
    if err != nil {
        return nil, err
    }
    defer file.Close()

    // Init new YAML decode
    d := yaml.NewDecoder(file)

    // Start YAML decoding from file
    if err := d.Decode(&config); err != nil {
        return nil, err
    }

    return config, nil
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

// ParseFlags will create and parse the CLI flags
// and return the path to be used elsewhere
func ParseFlags() (string, string, error) {
    // String that contains the configured configuration path
    var (
	    configPath string
	    runMode    string
    )

    // Setup a CLI flag called "-config" to allow users
    // to supply the configuration file
    flag.StringVar(&configPath, "config", "/etc/fireside/config.yml", "Filesystem path of bootstrap config file")

    // Setup a CLI flag called "-mode" in order to allow users to
    // change the operating / running "mode" for the fireside executable
    flag.StringVar(&runMode, "mode", "server", "Operational mode in which to run ; set to one of: 'server', 'ca'")

    // Actually parse the flags
    flag.Parse()

    // Validate the path first
    if err := ValidateConfigPath(configPath); err != nil {
        return "", "", err
    }

    switch runMode {
    case "ca":
	    log.Info("running mode = ca")
    case "server":
	    log.Info("running mode = server")
    default:
	    log.Fatal("unsupported value for running 'mode' : " + runMode)
    }

    // Return the configuration path
    return configPath, runMode, nil
}
