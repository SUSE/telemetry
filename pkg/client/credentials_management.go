package client

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"path/filepath"

	"github.com/SUSE/telemetry/pkg/config"
	"github.com/SUSE/telemetry/pkg/restapi"
	"github.com/SUSE/telemetry/pkg/utils"
)

const (
	CREDENTIALS_NAME = `credentials`
	CREDENTIALS_PERM = 0600
)

type TelemetryCreds struct {
	restapi.ClientRegistrationResponse
}

func (ta *TelemetryCreds) String() string {
	bytes, _ := json.Marshal(ta)
	return string(bytes)
}

type TelemetryClientCredentials struct {
	TelemetryCreds
	config    *config.Config
	credsFile utils.FileManager
	valid     bool
	no_retry  bool
}

func NewTelemetryClientCredentials(cfg *config.Config) (*TelemetryClientCredentials, error) {
	credsPath := filepath.Join(cfg.ConfigDir(), CREDENTIALS_NAME)
	c := &TelemetryClientCredentials{
		config:   cfg,
		valid:    false,
		no_retry: false,
	}

	// create a managed file to manage the credentials file based upon
	// the location, ownership and permissions of the config file with
	// backups disabled.
	fm := utils.NewManagedFile()
	err := fm.Init(
		credsPath,
		c.config.ConfigUser(),
		c.config.ConfigGroup(),
		CREDENTIALS_PERM,
	)
	fm.DisableBackups()

	if err != nil {
		slog.Debug(
			"failed to setup credentials file manager",
			slog.String("path", credsPath),
			slog.String("err", err.Error()),
		)
		return nil, fmt.Errorf("failed to setup credentials file manager: %w", err)
	}

	c.credsFile = fm

	return c, nil
}

func (c *TelemetryClientCredentials) UpdateCreds(creds *restapi.ClientAuthenticationResponse) (err error) {
	// stored the provided credentials
	c.ClientRegistrationResponse = *creds

	// mark credentials as valid
	c.valid = true

	// attempt to save the updated credentials
	err = c.Save()
	if err != nil {
		// we failed to save the credentials so mark them as invalid
		c.valid = false

		return
	}

	return
}

func (c *TelemetryClientCredentials) Exists() bool {
	exists, _ := c.credsFile.Exists()
	return exists
}

func (c *TelemetryClientCredentials) RetriesEnabled() bool {
	return !c.no_retry
}

func (c *TelemetryClientCredentials) DisableRetries() {
	c.no_retry = true
}

func (c *TelemetryClientCredentials) Valid() bool {
	return c.valid
}

func (c *TelemetryClientCredentials) Path() string {
	return c.credsFile.Path()
}

func (c *TelemetryClientCredentials) Accessible() bool {
	exists, _ := c.credsFile.Accessible()
	return exists
}

func (c *TelemetryClientCredentials) Credentials() TelemetryCreds {
	return c.TelemetryCreds
}

func (c *TelemetryClientCredentials) String() string {
	return fmt.Sprintf(
		"<p:%q, v:%v, c:%q>",
		c.Path(),
		c.valid,
		c.TelemetryCreds.String(),
	)
}

func (c *TelemetryClientCredentials) Save() (err error) {
	// saving an invalid credentials is not supported
	if !c.valid {
		return fmt.Errorf("client credentials not valid; cannot save")
	}

	err = c.credsFile.Create()
	if err != nil {
		slog.Debug(
			"failed to create/open credentials",
			slog.String("path", c.Path()),
			slog.String("err", err.Error()),
		)
	}
	defer c.credsFile.Close()

	// marshal the credentials public fields as JSON
	bytes, err := json.Marshal(c)
	if err != nil {
		slog.Error(
			"failed to json.Marshal() client credentials",
			slog.String("creds", c.String()),
			slog.String("err", err.Error()),
		)
		return
	}

	// save the JSON encoded credentials fields
	err = c.credsFile.Update(bytes)
	if err != nil {
		slog.Error(
			"failed to save client credentials file",
			slog.String("creds", c.String()),
			slog.String("err", err.Error()),
		)
		return
	}

	slog.Debug(
		"client credentials saved",
		slog.String("reg", c.String()),
	)
	return
}

func (c *TelemetryClientCredentials) Load() (err error) {
	// check that the specified path exists, failing appropriately
	exists, err := c.credsFile.Exists()
	if err != nil {
		slog.Debug(
			"unable to check for client credentials file",
			slog.String("path", c.Path()),
			slog.String("err", err.Error()),
		)
	}
	if !exists {
		slog.Error(
			"client credentials file not found",
			slog.String("path", c.Path()),
		)
		return
	}

	// open the credentials file
	err = c.credsFile.Open(
		false, // no need to create, should already exist
	)
	if err != nil {
		slog.Error(
			"failed to open client credentials",
			slog.String("path", c.Path()),
		)
		return
	}
	defer c.credsFile.Close()

	// retrieve the contents of the specified client credentials file
	bytes, err := c.credsFile.Read()
	if err != nil {
		slog.Error(
			"failed to read client credentials file",
			slog.String("path", c.Path()),
			slog.String("err", err.Error()),
		)
		return
	}

	// unmarshal the contents of the client credentials file into the
	// client credentials structure
	err = json.Unmarshal(bytes, c)
	if err != nil {
		slog.Error(
			"failed to json.Unmarshal() client credentials file contents",
			slog.String("path", c.Path()),
			slog.String("contents", string(bytes)),
			slog.String("err", err.Error()),
		)
		return
	}

	// validate the loaded contents
	err = c.Validate()
	if err != nil {
		slog.Error(
			"failed to validate client credentials file contents",
			slog.String("path", c.Path()),
			slog.String("contents", string(bytes)),
			slog.String("err", err.Error()),
		)
		return
	}

	// mark credentials as valid
	c.valid = true

	slog.Debug(
		"client credentials loaded",
		slog.String("path", c.Path()),
		slog.String("reg", c.String()),
	)

	return
}

func (c *TelemetryClientCredentials) Remove() (err error) {
	// mark in-memory version as invalid
	c.valid = false

	// nothing to do if file doesn't exist
	exists, _ := c.credsFile.Exists()
	if !exists {
		return
	}

	// delete the credentials file
	err = c.credsFile.Delete()
	if err != nil {
		slog.Error(
			"failed to delete client credentials",
			slog.String("path", c.Path()),
			slog.String("err", err.Error()),
		)
		return fmt.Errorf("failed to os.Remove(%q): %w", c.Path(), err)
	}

	return
}

func (c *TelemetryClientCredentials) Validate() (err error) {

	err = c.TelemetryCreds.Validate()
	if err != nil {
		slog.Debug(
			"client credentials validation failed",
			slog.String("err", err.Error()),
		)
	}

	return
}
