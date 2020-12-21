package configure

import (
    "os"
    "gopkg.in/yaml.v2"
)

// data structure for FireSide application configuration
type Config struct {

    // configs related to application audit logging
    Logging struct {
        // format the log output as JSON or (formatted) string;
        // valid values include:
        //     'json'
        //     'string' (default)
        //     '' (default)
        Format string   `yaml:"format"`
	// string value is converted to log.Level at runtime
	Level  string `yaml:"level"`
	// destination where logs are sent; defaults to os.Stdout
	// when this field is an empty string
	Output string `yaml:"output"`
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
        // list of files from which to read event inputs
	Files []InputConfigFileReader `yaml:"files"`
    } `yaml:"inputs"`

    // configs for data processing pipelines
    Pipelines struct {
        Envoy PipelineConfig `yaml:"envoy"`
        Falco PipelineConfig `yaml:"falco"`
    } `yaml:"pipelines"`

    // configs for output processors and providers
    Outputs struct {

        AWS struct {
            // optional ; equivalent of (defaults to) AWS_ACCESS_KEY_ID
            AccessKeyId     string `yaml:"access_key_id"`
	    // optional ; sets the name of the AWS profile to use for credentials
	    // preferred over (used before) explicit access/secret keys
	    Profile         string `yaml:"profile"`
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
