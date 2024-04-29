package config

import (
	"fmt"
	"log"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	cfgPath          string
	TelemetryBaseURL string   `yaml:"telemetry_base_url"`
	Enabled          bool     `yaml:"enabled"`
	CustomerID       string   `yaml:"customer_id"`
	Tags             []string `yaml:"tags"`
	ItemDS           string   `yaml:"item_datastore"`
	BundleDS         string   `yaml:"bundle_datastore"`
	ReportDS         string   `yaml:"report_datastore"`
	Extras           any      `yaml:"extras,omitempty"`
}

func NewConfig(cfgFile string) *Config {
	cfg := &Config{cfgPath: cfgFile}

	return cfg
}

func (cfg *Config) Path() string {
	return cfg.cfgPath
}

func (cfg *Config) Load() error {
	log.Printf("cfgPath: %q", cfg.cfgPath)
	_, err := os.Stat(cfg.cfgPath)
	if os.IsNotExist(err) {
		return fmt.Errorf("config file '%s' doesn't exist: %s", cfg.cfgPath, err)
	}

	contents, err := os.ReadFile(cfg.cfgPath)
	if err != nil {
		return fmt.Errorf("failed to read contents of config file '%s': %s", cfg.cfgPath, err)
	}

	log.Printf("Contents: %q", contents)
	err = yaml.Unmarshal(contents, cfg)
	if err != nil {
		return fmt.Errorf("failed to parse contents of config file '%s': %s", cfg.cfgPath, err)
	}

	return nil
}
