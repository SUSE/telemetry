package config

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"slices"

	"gopkg.in/yaml.v3"

	"github.com/SUSE/telemetry/pkg/types"
)

type Config struct {
	TelemetryBaseURL string             `yaml:"telemetry_base_url"`
	Enabled          bool               `yaml:"enabled"`
	CustomerID       string             `yaml:"customer_id"`
	Tags             []string           `yaml:"tags"`
	DataStores       DBConfig           `yaml:"datastores"`
	ClassOptions     ClassOptionsConfig `yaml:"classOptions"`
	Logging          LogConfig          `yaml:"logging"`
	Extras           any                `yaml:"extras,omitempty"`
}

// Defaults
var DefaultDBCfg = DBConfig{
	Driver: "sqlite3",
	Params: "/tmp/telemetry/client/telemetry.db",
}

var DefaultLogging = LogConfig{
	Level:    "info",
	Location: "stderr",
	Style:    "text",
}

var DefaultClassOptions = ClassOptionsConfig{
	OptOut: true,
	OptIn:  false,
	Allow:  []types.TelemetryType{},
	Deny:   []types.TelemetryType{},
}

var DefaultCfg = Config{
	//TelemetryBaseURL: "https://scc.suse.com/telemetry/",
	TelemetryBaseURL: "http://localhost:9999/telemetry",
	Enabled:          false,
	CustomerID:       "0",
	Tags:             []string{},
	DataStores:       DefaultDBCfg,
	Logging:          DefaultLogging,
	ClassOptions:     DefaultClassOptions,
}

// Datastore config for staging the data
type DBConfig struct {
	Driver string `yaml:"driver"`
	Params string `yaml:"params"`
}

type LogConfig struct {
	Level    string `yaml:"level" json:"level"`
	Location string `yaml:"location" json:"location"`
	Style    string `yaml:"style" json:"style"`
}

func (lc *LogConfig) String() string {
	str, _ := json.Marshal(lc)
	return string(str)
}

type ClassOptionsConfig struct {
	OptOut bool                  `yaml:"opt_out" json:"opt_out"`
	OptIn  bool                  `yaml:"opt_in" json:"opt_in"`
	Allow  []types.TelemetryType `yaml:"allow" json:"allow"`
	Deny   []types.TelemetryType `yaml:"deny" json:"deny"`
}

func NewConfig(cfgFile string) (*Config, error) {

	//Default configuration
	cfg := &DefaultCfg

	_, err := os.Stat(cfgFile)
	if os.IsNotExist(err) {
		slog.Warn("config file doesn't exist. Using default configuration", slog.String("cfgfile", cfgFile))
		return cfg, nil
	}

	contents, err := os.ReadFile(cfgFile)
	if err != nil {
		return cfg, fmt.Errorf("failed to read contents of config file '%s': %s", cfgFile, err)
	}

	slog.Debug("Contents", slog.String("contents", string(contents)))
	err = yaml.Unmarshal(contents, &cfg)
	if err != nil {
		return cfg, fmt.Errorf("failed to parse contents of config file '%s': %s", cfgFile, err)
	}

	return cfg, nil
}

func (c *Config) TelemetryClassEnabled(class types.TelemetryClass) bool {

	switch class {
	case types.MANDATORY_TELEMETRY:
		return true
	case types.OPT_OUT_TELEMETRY:
		return c.ClassOptions.OptOut
	case types.OPT_IN_TELEMETRY:
		return c.ClassOptions.OptIn
	}

	return false
}

func (c *Config) TelemetryTypeEnabled(telemetry types.TelemetryType) bool {
	// if telemetry type is in the allow list then allow it to be sent
	if slices.Contains(c.ClassOptions.Allow, telemetry) {
		return true
	}

	// otherwise if telemetry type is in the deny list then deny it
	if slices.Contains(c.ClassOptions.Deny, telemetry) {
		return false
	}

	// otherwise allow the telemetry type
	return true
}
