package client

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"path/filepath"

	"github.com/SUSE/telemetry/pkg/config"
	"github.com/SUSE/telemetry/pkg/types"
	"github.com/SUSE/telemetry/pkg/utils"
	"github.com/google/uuid"
)

const (
	REGISTRATION_NAME = `registration`
	REGISTRATION_PERM = 0600
)

type TelemetryClientRegistration struct {
	types.ClientRegistration
	config   *config.Config
	regFile  utils.FileManager
	valid    bool
	no_retry bool
}

func NewTelemetryClientRegistration(cfg *config.Config) (*TelemetryClientRegistration, error) {
	regPath := filepath.Join(cfg.ConfigDir(), REGISTRATION_NAME)
	r := &TelemetryClientRegistration{
		config:   cfg,
		valid:    false,
		no_retry: false,
	}

	// create a managed file to manage the registration file based upon
	// the location, ownership and permissions of the config file with
	// backups disabled.
	fm := utils.NewManagedFile()
	err := fm.Init(
		regPath,
		r.config.ConfigUser(),
		r.config.ConfigGroup(),
		REGISTRATION_PERM,
	)
	fm.DisableBackups()

	if err != nil {
		slog.Debug(
			"failed to setup registration file manager",
			slog.String("path", regPath),
			slog.String("err", err.Error()),
		)
		return nil, fmt.Errorf("failed to setup registration file manager: %w", err)
	}

	r.regFile = fm

	return r, nil
}

func (r *TelemetryClientRegistration) Exists() bool {
	exists, _ := r.regFile.Exists()
	return exists
}

func (r *TelemetryClientRegistration) RetriesEnabled() bool {
	return !r.no_retry
}

func (r *TelemetryClientRegistration) DisableRetries() {
	r.no_retry = true
}

func (r *TelemetryClientRegistration) Valid() bool {
	return r.valid
}

func (r *TelemetryClientRegistration) Path() string {
	return r.regFile.Path()
}

func (r *TelemetryClientRegistration) Accessible() bool {
	accessible, _ := r.regFile.Accessible()
	return accessible
}

func (r *TelemetryClientRegistration) Generate() (err error) {
	// if no client id is configured, generate a new client id and
	// attempt to save it
	if r.config.ClientId == "" {
		r.config.ClientId = uuid.New().String()
		err = r.config.Save()
		if err != nil {
			slog.Debug(
				"failed to save updated config after generating a new client id",
				slog.String("config", r.config.ConfigPath()),
				slog.String("err", err.Error()),
			)
			return fmt.Errorf("failed to save config after generating client id: %w", err)
		}
	}

	r.ClientId = r.config.ClientId
	r.SystemUUID = getSystemUUID()
	r.Timestamp = types.Now().String()

	// mark registration as valid
	r.valid = true

	// attempt to save the newly generated registration
	err = r.Save()
	if err != nil {
		// we failed to save the registration so mark it as invalid
		r.valid = false

		return
	}

	return
}

func (r *TelemetryClientRegistration) Registration() types.ClientRegistration {
	return r.ClientRegistration
}

func (r *TelemetryClientRegistration) String() string {
	return fmt.Sprintf(
		"<p:%q, v:%v, r:%q>",
		r.Path(),
		r.valid,
		r.ClientRegistration.String(),
	)
}

func (r *TelemetryClientRegistration) Save() (err error) {
	// saving an invalid registration is not supported
	if !r.valid {
		return fmt.Errorf("client registration not valid; cannot save")
	}

	err = r.regFile.Create()
	if err != nil {
		slog.Debug(
			"failed to create/open registration",
			slog.String("path", r.Path()),
			slog.String("err", err.Error()),
		)
	}
	defer r.regFile.Close()

	// marshal the registration public fields as JSON
	bytes, err := json.Marshal(r)
	if err != nil {
		slog.Error(
			"failed to json.Marshal() client registration",
			slog.String("reg", r.String()),
			slog.String("err", err.Error()),
		)
		return
	}

	// save the JSON encoded registration fields
	err = r.regFile.Update(bytes)
	if err != nil {
		slog.Error(
			"failed to save client registration file",
			slog.String("reg", r.String()),
			slog.String("err", err.Error()),
		)
		return
	}

	slog.Debug(
		"client registration saved",
		slog.String("reg", r.String()),
	)
	return
}

func (r *TelemetryClientRegistration) Load() (err error) {
	// check that the specified path exists, failing appropriately
	exists, err := r.regFile.Exists()
	if err != nil {
		slog.Debug(
			"unable to check for client registration file",
			slog.String("path", r.Path()),
			slog.String("err", err.Error()),
		)
	}
	if !exists {
		slog.Error(
			"client registration file not found",
			slog.String("path", r.Path()),
		)
		return
	}

	// open the registration file
	err = r.regFile.Open(
		false, // no need to create, should already exist
	)
	if err != nil {
		slog.Error(
			"failed to open client registration",
			slog.String("path", r.Path()),
		)
		return
	}
	defer r.regFile.Close()

	// retrieve the contents of the specified client registration file
	bytes, err := r.regFile.Read()
	if err != nil {
		slog.Error(
			"failed to read client registration file",
			slog.String("path", r.Path()),
			slog.String("err", err.Error()),
		)
		return
	}

	// unmarshal the contents of the client registration file into the
	// client registration structure
	err = json.Unmarshal(bytes, r)
	if err != nil {
		slog.Error(
			"failed to json.Unmarshal() client registration file contents",
			slog.String("path", r.Path()),
			slog.String("contents", string(bytes)),
			slog.String("err", err.Error()),
		)
		return
	}

	// validate the loaded contents
	err = r.Validate()
	if err != nil {
		slog.Error(
			"failed to validate client registration file contents",
			slog.String("path", r.Path()),
			slog.String("contents", string(bytes)),
			slog.String("err", err.Error()),
		)
		return
	}

	// mark registration as valid
	r.valid = true

	slog.Debug(
		"client registration loaded",
		slog.String("path", r.Path()),
		slog.String("reg", r.String()),
	)

	return
}

func (r *TelemetryClientRegistration) Remove() (err error) {
	// mark in-memory version as invalid
	r.valid = false

	// nothing to do if file doesn't exist
	exists, _ := r.regFile.Exists()
	if !exists {
		return
	}

	// delete the registration file
	err = r.regFile.Delete()
	if err != nil {
		slog.Error(
			"failed to delete client registration",
			slog.String("path", r.Path()),
			slog.String("err", err.Error()),
		)
		return fmt.Errorf("failed to os.Remove(%q): %w", r.Path(), err)
	}

	return
}

func (c *TelemetryClientRegistration) Validate() (err error) {

	err = c.ClientRegistration.Validate()
	if err != nil {
		slog.Debug(
			"client registration validation failed",
			slog.String("err", err.Error()),
		)
	}

	return
}
