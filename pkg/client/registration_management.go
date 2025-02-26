package client

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"

	"github.com/SUSE/telemetry/pkg/types"
	"github.com/google/uuid"
)

const (
	REGISTRATION_PATH = CONFIG_DIR + "/registration"
)

type TelemetryClientRegistration struct {
	types.ClientRegistration
	path     string
	valid    bool
	no_retry bool
}

func NewTelemetryClientRegistration() *TelemetryClientRegistration {
	return &TelemetryClientRegistration{
		path:     REGISTRATION_PATH,
		valid:    false,
		no_retry: false,
	}
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
	return r.path
}

// for testing purposes
func (r *TelemetryClientRegistration) SetPath(path string) {
	r.path = path
}

func (r *TelemetryClientRegistration) Accessible() bool {
	if _, err := os.Open(r.path); err != nil {
		return false
	}
	return true
}

func (r *TelemetryClientRegistration) Generate() {
	r.ClientId = uuid.New().String()
	r.SystemUUID = getSystemUUID()
	r.Timestamp = types.Now().String()
	r.valid = true
}

func (r *TelemetryClientRegistration) Registration() types.ClientRegistration {
	return r.ClientRegistration
}

func (r *TelemetryClientRegistration) String() string {
	return fmt.Sprintf(
		"<p:%q, v:%v, r:%q>",
		r.path,
		r.valid,
		r.ClientRegistration.String(),
	)
}

func (r *TelemetryClientRegistration) Save() (err error) {
	// saving an invalid registration is not supported
	if !r.valid {
		return fmt.Errorf("client registration not valid; cannot save")
	}

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
	err = os.WriteFile(r.path, bytes, 0600)
	if err != nil {
		slog.Error(
			"failed to write client registration file",
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
	// if there are any issues os.Stat()ing the file.
	_, err = os.Stat(r.path)
	if err != nil {
		var msg string
		if errors.Is(err, fs.ErrNotExist) {
			msg = "client registration file not found"
		} else {
			msg = "unable to os.Stat() client registration file"
		}
		slog.Error(
			msg,
			slog.String("regPath", r.path),
			slog.String("err", err.Error()),
		)
		return
	}

	// retrieve the contents of the specified client registration file,
	// failing if unable to do so.
	bytes, err := os.ReadFile(r.path)
	if err != nil {
		slog.Error(
			"failed to read client registration file",
			slog.String("regPath", r.path),
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
			slog.String("regPath", r.path),
			slog.String("contents", string(bytes)),
			slog.String("err", err.Error()),
		)
		return
	}

	slog.Debug(
		"client registration loaded",
		slog.String("reg", r.String()),
	)
	return
}

func (r *TelemetryClientRegistration) Remove() (err error) {
	// mark in-memory version as invalid
	r.valid = false

	// check if registration file exists
	_, err = os.Stat(r.path)
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			// nothing to do, and not a failure, if registration file doesn't exist
			err = nil
		} else {
			slog.Error(
				"unable to os.Stat() client registration file",
				slog.String("regPath", r.path),
				slog.String("err", err.Error()),
			)
		}
		return
	}

	// remove the registration client, reporting any errors that occurr
	err = os.Remove(r.path)
	if err != nil {
		slog.Error(
			"failed to os.Remove() client registration file",
			slog.String("regPath", r.path),
			slog.String("err", err.Error()),
		)
	}

	return
}
