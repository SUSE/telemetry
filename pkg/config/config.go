package config

import (
	"fmt"
	"log"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	TelemetryBaseURL string   `yaml:"telemetry_base_url"`
	Enabled          bool     `yaml:"enabled"`
	CustomerID       string   `yaml:"customer_id"`
	Tags             []string `yaml:"tags"`
	DataStores       DBConfig `yaml:"datastores"`
	Extras           any      `yaml:"extras,omitempty"`
}

// Defaults
var DefaultDBCfg = DBConfig{
	Driver: "sqlite3",
	Params: "/tmp/telemetry/client/telemetry.db",
}

var DefaultCfg = Config{
	//TelemetryBaseURL: "https://scc.suse.com/telemetry/",
	TelemetryBaseURL: "http://localhost:9999/telemetry",
	Enabled:          false,
	CustomerID:       "0",
	Tags:             []string{},
	DataStores:       DefaultDBCfg,
}

// Datastore config for staging the data
type DBConfig struct {
	Driver string `yaml:"driver"`
	Params string `yaml:"params"`
}

func NewConfig(cfgFile string) (*Config, error) {

	//Default configuration
	cfg := &DefaultCfg

	_, err := os.Stat(cfgFile)
	if os.IsNotExist(err) {
		log.Printf("config file '%s' doesn't exist. Using default configuration.", cfgFile)
		return cfg, nil
	}

	contents, err := os.ReadFile(cfgFile)
	if err != nil {
		return cfg, fmt.Errorf("failed to read contents of config file '%s': %s", cfgFile, err)
	}

	log.Printf("Contents: %q", contents)
	err = yaml.Unmarshal(contents, &cfg)
	if err != nil {
		return cfg, fmt.Errorf("failed to parse contents of config file '%s': %s", cfgFile, err)
	}

	return cfg, nil
}
