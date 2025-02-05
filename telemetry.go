package telemetry

import (
	"log/slog"

	"github.com/SUSE/telemetry/pkg/client"
	"github.com/SUSE/telemetry/pkg/config"
	"github.com/SUSE/telemetry/pkg/types"
)

type TelemetryType = types.TelemetryType

type Tags = types.Tags

type TelemetryClass = types.TelemetryClass

const (
	MANDATORY_TELEMETRY = types.MANDATORY_TELEMETRY
	OPT_OUT_TELEMETRY   = types.OPT_OUT_TELEMETRY
	OPT_IN_TELEMETRY    = types.OPT_IN_TELEMETRY
)

type GenerateFlags uint64

const (
	GENERATE GenerateFlags = iota
	SUBMIT   GenerateFlags = 1 << (iota - 1)
)

func (gf *GenerateFlags) String() string {
	flags := "GENERATE"
	switch {
	case gf.FlagSet(SUBMIT):
		flags += "|SUBMIT"
		fallthrough
	default:
		// nothing to do
	}
	return flags
}

func (gf GenerateFlags) FlagSet(flag GenerateFlags) bool {
	return (gf & flag) == flag
}

func (gf GenerateFlags) SubmitRequested() bool {
	return gf.FlagSet(SUBMIT)
}

func Generate(telemetry types.TelemetryType, class TelemetryClass, content []byte, tags types.Tags, flags GenerateFlags) (err error) {
	// check that the telemetry type is valid
	if valid, err := telemetry.Valid(); !valid {
		slog.Error(
			"Invalid telemetry type",
			slog.String("type", string(telemetry)),
		)
		return err
	}

	// check that the telemetry content is valid
	blob := types.NewTelemetryBlob(content)

	// attempt to load the default config file
	cfg, err := config.NewConfig(client.CONFIG_PATH)
	if err != nil {
		slog.Error(
			"Failed to load telemetry client config",
			slog.String("path", client.CONFIG_PATH),
			slog.String("error", err.Error()),
		)
		return
	}

	// check if the telemetry client is enabled in config
	if !cfg.Enabled {
		slog.Warn("The telemetry client is disabled in the configuration; no telemetry generated")
		return
	}

	// check that the telemetry class is enabled for generation
	if !cfg.TelemetryClassEnabled(class) {
		slog.Warn(
			"Telemetry class generation is disabled",
			slog.String("class", class.String()),
		)
		return
	}

	// check that the telemetry type is enabled for generation
	if !cfg.TelemetryTypeEnabled(telemetry) {
		slog.Warn(
			"Telemetry class generation is disabled",
			slog.String("class", class.String()),
		)
		return
	}

	// instantiate a telemetry client
	tc, err := client.NewTelemetryClient(cfg)
	if err != nil {
		slog.Warn(
			"Failed to instantiate a TelemetryClient",
			slog.String("error", err.Error()),
		)
		return
	}

	// ensure the client is registered
	err = tc.Register()
	if err != nil {
		slog.Warn(
			"Failed to register TelemetryClient with upstream server",
			slog.String("error", err.Error()),
		)
		return
	}

	// generate the telemetry, storing it in the local data store
	err = tc.Generate(telemetry, blob, tags)
	if err != nil {
		slog.Warn(
			"Failed to generate telemetry",
			slog.String("error", err.Error()),
		)
		return
	}

	// check if immediate submission requested
	if flags.SubmitRequested() {
		// TODO: implement immediate submission
		slog.Info("Immediate Telemetry Submission requested")
	}

	return
}

type ClientStatus int64

const (
	CLIENT_UNINITIALIZED ClientStatus = iota
	CLIENT_CONFIG_ACCESSIBLE
	CLIENT_DISABLED
	CLIENT_MISCONFIGURED
	CLIENT_DATASTORE_ACCESSIBLE
	CLIENT_REGISTRATION_ACCESSIBLE
	CLIENT_REGISTERED
)

func (cs *ClientStatus) String() string {
	switch *cs {
	case CLIENT_UNINITIALIZED:
		return "UNITITIALIZED"
	case CLIENT_CONFIG_ACCESSIBLE:
		return "CONFIG_ACCESSIBLE"
	case CLIENT_DISABLED:
		return "DISABLED"
	case CLIENT_MISCONFIGURED:
		return "MISCONFIGURED"
	case CLIENT_DATASTORE_ACCESSIBLE:
		return "DATASTORE_ACCESSIBLE"
	case CLIENT_REGISTRATION_ACCESSIBLE:
		return "REGISTRATION_ACCESSIBLE"
	case CLIENT_REGISTERED:
		return "REGISTERED"
	}
	return "UNKNOWN_TELEMETRY_CLIENT_STATUS"
}

func Status() (status ClientStatus) {
	// default to being uninitialised
	status = CLIENT_UNINITIALIZED

	// attempt to load the default config
	cfg, err := config.NewConfig(client.CONFIG_PATH)
	if err != nil {
		slog.Warn(
			"Failed to load telemetry client config",
			slog.String("path", client.CONFIG_PATH),
			slog.String("error", err.Error()),
		)
		return
	}

	// update status to indicate that telemetry client configuration is accessible
	status = CLIENT_CONFIG_ACCESSIBLE

	// check if the telemetry client is enabled in config
	if !cfg.Enabled {
		slog.Info("The telemetry client is disabled in the configuration")
		return CLIENT_DISABLED
	}

	// instantiate a telemetry client using provided config
	tc, err := client.NewTelemetryClient(cfg)
	if err != nil {
		slog.Warn(
			"Failed to setup telemetry client using provided config",
			slog.String("path", client.CONFIG_PATH),
			slog.String("error", err.Error()),
		)
		return CLIENT_MISCONFIGURED
	}

	// update status to indicate that telemetry client datastore is accessible
	status = CLIENT_DATASTORE_ACCESSIBLE

	// check that an instance id is available
	if !tc.RegistrationAccessible() {
		slog.Warn("Telemetry client registration has not been setup", slog.String("path", tc.RegistrationPath()))
		return
	}

	// update status to indicate client has instance id
	status = CLIENT_REGISTRATION_ACCESSIBLE

	// check that we have obtained a telemetry auth token
	if !tc.AuthAccessible() {
		slog.Warn("Telemetry client has not been registered", slog.String("path", tc.AuthPath()))
		return
	}

	// update status to indicate telemetry client is registered
	status = CLIENT_REGISTERED

	return
}
