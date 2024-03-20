package config

import (
	"fmt"
	"log"
	"os"

	"gopkg.in/yaml.v3"
)

type DataStoreConfig struct {
	Type string `yaml:"type"`
	Path string `yaml:"path"`
}

type Config struct {
	Enabled    bool            `yaml:"enabled"`
	CustomerID string          `yaml:"customer_id"`
	Tags       []string        `yaml:"tags"`
	ItemDS     DataStoreConfig `yaml:"item_datastore"`
	BundleDS   DataStoreConfig `yaml:"bundle_datastore"`
	Extras     any             `yaml:"extras,omitempty"`
}

func NewConfig() *Config {
	cfg := &Config{}

	return cfg
}
func Load(configPath string, cfg *Config) error {
	_, err := os.Stat(configPath)
	if os.IsNotExist(err) {
		return fmt.Errorf("config file '%s' doesn't exist: %s", configPath, err)
	}

	contents, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read contents of config file '%s': %s", configPath, err)
	}

	log.Printf("Contents: %q", contents)
	err = yaml.Unmarshal(contents, cfg)
	if err != nil {
		return fmt.Errorf("failed to parse contents of config file '%s': %s", configPath, err)
	}

	return nil
}
