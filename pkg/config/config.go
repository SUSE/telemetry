package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"path/filepath"
	"slices"

	"gopkg.in/yaml.v3"

	"github.com/SUSE/telemetry/pkg/types"
	"github.com/SUSE/telemetry/pkg/utils"
)

const (
	// config file defaults
	DEF_CFG_DIR   = `/etc/susetelemetry`
	DEF_CFG_FILE  = `telemetry.yaml`
	DEF_CFG_PATH  = DEF_CFG_DIR + `/` + DEF_CFG_FILE
	DEF_CFG_USER  = `susetelm`
	DEF_CFG_GROUP = `susetelm`
	DEF_CFG_PERM  = 0640

	// config defaults
	DEF_CFG_ENABLED     = false
	DEF_CFG_BASE_URL    = `https://scc.suse.com/telemetry/`
	DEF_CFG_CLIENT_ID   = ``
	DEF_CFG_CUSTOMER_ID = ``

	// data store defaults
	DEF_CFG_DB_DRIVER = `sqlite3`
	DEF_CFG_DB_DIR    = `/var/lib/` + DEF_CFG_USER + `/client`
	DEF_CFG_DB_FILE   = `telemetry.db`
	DEF_CFG_DB_PATH   = DEF_CFG_DB_DIR + `/` + DEF_CFG_DB_FILE

	// logging defaults
	DEF_CFG_LOG_LEVEL    = `info`
	DEF_CFG_LOG_LOCATION = `stderr`
	DEF_CFG_LOG_STYLE    = `text`

	// class defaults
	DEF_CFG_OPT_OUT = true
	DEF_CFG_OPT_IN  = false
)

// datastore config for staging provided telemetry data
type DBConfig struct {
	Driver string `yaml:"driver"`
	Params string `yaml:"params"`
}

func (dc *DBConfig) String() string {
	str, _ := json.Marshal(dc)
	return string(str)
}

// logging config
type LogConfig struct {
	Level    string `yaml:"level" json:"level"`
	Location string `yaml:"location" json:"location"`
	Style    string `yaml:"style" json:"style"`
}

func (lc *LogConfig) String() string {
	str, _ := json.Marshal(lc)
	return string(str)
}

// telemetry class options configuration
type ClassOptionsConfig struct {
	OptOut bool                  `yaml:"opt_out" json:"opt_out"`
	OptIn  bool                  `yaml:"opt_in" json:"opt_in"`
	Allow  []types.TelemetryType `yaml:"allow" json:"allow"`
	Deny   []types.TelemetryType `yaml:"deny" json:"deny"`
}

func (cc *ClassOptionsConfig) String() string {
	str, _ := json.Marshal(cc)
	return string(str)
}

type Config struct {
	TelemetryBaseURL string             `yaml:"telemetry_base_url"`
	Enabled          bool               `yaml:"enabled"`
	ClientId         string             `yaml:"client_id"`
	CustomerId       string             `yaml:"customer_id"`
	Tags             []string           `yaml:"tags"`
	DataStores       DBConfig           `yaml:"datastores"`
	ClassOptions     ClassOptionsConfig `yaml:"class_options"`
	Logging          LogConfig          `yaml:"logging"`
	Extras           any                `yaml:"extras,omitempty"`

	cfgPath string
	cfgDir  string
	cfgFile utils.FileManager
}

func (c *Config) ConfigDir() string {
	return c.cfgDir
}

func (c *Config) ConfigPath() string {
	return c.cfgPath
}

func (c *Config) ConfigName() string {
	return filepath.Base(c.cfgPath)
}

func (c *Config) ConfigUser() string {
	return c.cfgFile.User()
}

func (c *Config) ConfigGroup() string {
	return c.cfgFile.Group()
}

func NewDefaultConfig() *Config {

	return &Config{
		TelemetryBaseURL: DEF_CFG_BASE_URL,
		Enabled:          DEF_CFG_ENABLED,
		ClientId:         DEF_CFG_CLIENT_ID,
		CustomerId:       DEF_CFG_CUSTOMER_ID,
		Tags:             []string{},

		DataStores: DBConfig{
			Driver: DEF_CFG_DB_DRIVER,
			Params: DEF_CFG_DB_PATH,
		},

		Logging: LogConfig{
			Level:    DEF_CFG_LOG_LEVEL,
			Location: DEF_CFG_LOG_LOCATION,
			Style:    DEF_CFG_LOG_STYLE,
		},

		ClassOptions: ClassOptionsConfig{
			OptOut: DEF_CFG_OPT_OUT,
			OptIn:  DEF_CFG_OPT_IN,
			Allow:  []types.TelemetryType{},
			Deny:   []types.TelemetryType{},
		},

		cfgPath: DEF_CFG_PATH,
	}
}

func (cfg *Config) create_with_defaults() (err error) {
	// cfg should have a valid cfgPath, cfgDir and cfgFile
	if cfg.cfgPath == "" || cfg.cfgDir == "" || cfg.cfgFile == nil {
		return fmt.Errorf("config not initialised correctly")
	}

	// attempt to setup the user, group and perms for the config file
	slog.Warn(
		"config file doesn't exist, using builtin defaults",
		slog.String("path", cfg.cfgPath),
		slog.String("config", cfg.String()),
	)

	// setup default user
	if err = cfg.cfgFile.SetUser(DEF_CFG_USER); err != nil {
		slog.Debug(
			"failed to set user for config file",
			slog.String("path", cfg.cfgPath),
			slog.String("user", DEF_CFG_USER),
			slog.String("err", err.Error()),
		)
		return err
	}

	// setup default group
	if err = cfg.cfgFile.SetUser(DEF_CFG_USER); err != nil {
		slog.Debug(
			"failed to set user for config file",
			slog.String("path", cfg.cfgPath),
			slog.String("user", DEF_CFG_USER),
			slog.String("err", err.Error()),
		)
		return err
	}

	// setup default perms
	if err = cfg.cfgFile.SetPerm(DEF_CFG_PERM); err != nil {
		slog.Debug(
			"failed to set permissions for config file",
			slog.String("path", cfg.cfgPath),
			slog.String("perm", fs.FileMode(DEF_CFG_PERM).String()),
			slog.String("err", err.Error()),
		)
		return err
	}

	// attempt to create the file with the default config settings
	if err = cfg.cfgFile.Create(); err != nil {
		slog.Debug(
			"failed to create config file",
			slog.String("path", cfg.cfgPath),
			slog.String("err", err.Error()),
		)
		return err
	}

	// attempt to init config file with default config settings
	if err = cfg.Save(); err != nil {
		slog.Debug(
			"failed to init config file contents",
			slog.String("path", cfg.cfgPath),
			slog.String("err", err.Error()),
		)
		return err
	}

	return nil
}

func (cfg *Config) Save() (err error) {
	content, err := yaml.Marshal(cfg)
	if err != nil {
		slog.Debug(
			"failed to yaml.Marshal() config",
			slog.String("config", cfg.String()),
			slog.String("err", err.Error()),
		)

		return fmt.Errorf("failed to yaml.Marshal() config content: %w", err)
	}

	if err = cfg.cfgFile.Update(content); err != nil {
		slog.Debug(
			"failed to update config",
			slog.String("path", cfg.cfgPath),
			slog.String("err", err.Error()),
		)

		return fmt.Errorf("failed to write config %q: %w", cfg.cfgPath, err)
	}

	return
}

func (cfg *Config) Load() (err error) {
	contents, err := cfg.cfgFile.Read()
	if err != nil {
		return fmt.Errorf("failed to read contents of config file %q: %w", cfg.cfgPath, err)
	}

	slog.Debug(
		"Config read",
		slog.String("contents", string(contents)),
	)

	err = yaml.Unmarshal(contents, &cfg)
	if err != nil {
		slog.Debug(
			"failed to yaml.Marshal() config",
			slog.String("config", cfg.String()),
			slog.String("err", err.Error()),
		)

		return fmt.Errorf("failed to yaml.Unmarshal() contents of config %q: %w", cfg.cfgPath, err)
	}

	slog.Debug(
		"Config parsed",
		slog.String("config", cfg.String()),
	)

	return
}

func NewConfig(cfgPath string) (*Config, error) {

	//Initialise with default configuration
	cfg := NewDefaultConfig()

	// attempt to use an existing config file
	cfgFile := utils.NewManagedFile()
	err := cfgFile.UseExistingFile(cfgPath)
	if err != nil {
		// fail if the error is something other than the config not existing
		if !errors.Is(err, fs.ErrNotExist) {
			return nil, fmt.Errorf("unable to access config file %q: %w", cfgPath, err)
		}

		// if the file didn't exist ensure the managed file path is setup
		err = cfgFile.SetPath(cfg.cfgPath)
		if err != nil {
			return nil, fmt.Errorf("failed to set config file path %q: %w", cfgPath, err)
		}
	}

	// set the config path, dir and the cfgFile
	cfg.cfgPath = cfgFile.Path()
	cfg.cfgDir = filepath.Dir(cfg.cfgPath)
	cfg.cfgFile = cfgFile

	// if the config file doesn't exist, attempt to setup with defaults
	if exists, _ := cfgFile.Exists(); !exists {
		// attempt to create and initialise the configuration
		if err = cfg.create_with_defaults(); err != nil {
			return nil, err
		}
	}

	// attempt to open config file, creating it necessary
	err = cfgFile.Create()
	if err != nil {
		return cfg, fmt.Errorf("failed to open config file %q: %w", cfgPath, err)
	}

	err = cfg.Load()
	if err != nil {
		return nil, err
	}

	return cfg, nil
}

func (c *Config) String() string {
	bytes, _ := json.Marshal(c)
	return string(bytes)
}

func (c *Config) TelemetryClassEnabled(class types.TelemetryClass) bool {
	if !c.Enabled {
		return false
	}

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
	if !c.Enabled {
		return false
	}

	// if telemetry type is in the deny list then deny it from being sent
	if slices.Contains(c.ClassOptions.Deny, telemetry) {
		return false
	}

	// otherwise if telemetry type is in the allow list then allow it to be sent
	if slices.Contains(c.ClassOptions.Allow, telemetry) {
		return true
	}

	// default to allowing if allow list empty
	return len(c.ClassOptions.Allow) == 0
}
