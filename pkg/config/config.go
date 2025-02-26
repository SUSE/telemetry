package config

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"path/filepath"
	"slices"

	"gopkg.in/yaml.v3"

	"github.com/SUSE/telemetry/pkg/types"
	"github.com/SUSE/telemetry/pkg/utils"
)

type Config struct {
	TelemetryBaseURL string             `yaml:"telemetry_base_url"`
	Enabled          bool               `yaml:"enabled"`
	ClientId         string             `yaml:"client_id"`
	CustomerId       string             `yaml:"customer_id"`
	Tags             []string           `yaml:"tags"`
	DataStores       DBConfig           `yaml:"datastores"`
	ClassOptions     ClassOptionsConfig `yaml:"classOptions"`
	Logging          LogConfig          `yaml:"logging"`
	Extras           any                `yaml:"extras,omitempty"`
	cfgPath          string
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
	ClientId:         "",
	CustomerId:       "0",
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

func NewConfig(cfgPath string) (*Config, error) {

	//Initialise with default configuration
	cfg := &Config{}
	*cfg = DefaultCfg

	// get absolute path of config file
	absPath, err := filepath.Abs(cfgPath)
	if err != nil {
		return cfg, fmt.Errorf("unable to resolve absolute path of config file %q: %w", cfgPath, err)
	}

	cfgFile := utils.NewManagedFile()
	cfgFile.Init(
		absPath,
		// TODO (fmccarthy): revisit the file ownership
		"", // current user
		"", // current user's primary group
		0644,
	)

	exists, err := cfgFile.Exists()
	if err != nil || !exists {
		if err != nil {
			slog.Debug(
				"existence check for config file failed",
				slog.String("cfgPath", cfgPath),
				slog.String("err", err.Error()),
			)
		}
		slog.Warn("config file doesn't exist. Using default configuration", slog.String("cfgPath", cfgPath))
		return cfg, nil
	}

	err = cfgFile.Create()
	if err != nil {
		return cfg, fmt.Errorf("failed to open config file %q: %w", cfgPath, err)
	}

	// config file exists and is accessible so record it
	cfg.cfgPath = cfgPath

	contents, err := cfgFile.Read()
	if err != nil {
		return cfg, fmt.Errorf("failed to read contents of config file %q: %w", cfgPath, err)
	}

	slog.Debug("Config", slog.String("contents", string(contents)))
	err = yaml.Unmarshal(contents, &cfg)
	if err != nil {
		return cfg, fmt.Errorf("failed to parse contents of config file %q: %w", cfgPath, err)
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
