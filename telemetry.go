package telemetry

import (
	"fmt"
	"log/slog"

	"github.com/SUSE/telemetry/pkg/client"
	"github.com/SUSE/telemetry/pkg/config"
	"github.com/SUSE/telemetry/pkg/types"
	"github.com/SUSE/telemetry/pkg/utils"
)

//
// config path management
//

// the active config path to use
var activeConfigPath string = config.DEF_CFG_PATH

// retrieve the default config path
func DefaultConfigPath() string {
	return config.DEF_CFG_PATH
}

// get the active config path
func ConfigPath() string {
	return activeConfigPath
}

// set the active config path
func SetConfigPath(path string) {
	activeConfigPath = path
}

// Telemetry Class, Type and Tags
type TelemetryType = types.TelemetryType

type TelemetryClass = types.TelemetryClass

const (
	MANDATORY_TELEMETRY = types.MANDATORY_TELEMETRY
	OPT_OUT_TELEMETRY   = types.OPT_OUT_TELEMETRY
	OPT_IN_TELEMETRY    = types.OPT_IN_TELEMETRY
)

type Tags = types.Tags

//
// GenerateFlags type
//

type GenerateFlags uint64

const (
	GENERATE GenerateFlags = iota
	SUBMIT   GenerateFlags = 1 << (iota - 1)
)

func (gf GenerateFlags) IsFlagSet(flag GenerateFlags) bool {
	return (gf & flag) == flag
}

func (gf *GenerateFlags) String() string {
	flags := "GENERATE"
	switch {
	case gf.IsFlagSet(SUBMIT):
		flags += "|SUBMIT"
		fallthrough
	default:
		// nothing to do
	}
	return flags
}

func (gf GenerateFlags) SubmitRequested() bool {
	return gf.IsFlagSet(SUBMIT)
}

//
// ClientStatus type
//

type ClientStatus int64

const (
	CLIENT_UNINITIALIZED ClientStatus = iota
	CLIENT_CONFIG_MISSING
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
	case CLIENT_CONFIG_MISSING:
		return "CONFIG_MISSING"
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

//
// Helper routines
//

func getClientConfig() (cfg *config.Config, err error) {
	// attempt to load the active config
	cfg, err = config.NewConfig(activeConfigPath)
	if err != nil {
		slog.Error(
			"Failed to load telemetry client config",
			slog.String("path", activeConfigPath),
			slog.String("error", err.Error()),
		)
		err = fmt.Errorf(
			"failed to load telemetry client config %q: %w",
			activeConfigPath,
			err,
		)
		return nil, err
	}

	return cfg, nil
}

//
// Telemetry Generation and Submission
//

func Generate(
	telemetry types.TelemetryType,
	class TelemetryClass,
	content []byte,
	tags types.Tags,
	flags GenerateFlags,
) (err error) {
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
	if err = blob.Valid(); err != nil {
		slog.Error(
			"Telemetry JSON blob is not valid",
			slog.String("content", string(content)),
			slog.String("error", err.Error()),
		)
		return fmt.Errorf("invalid Telemetry JSON blob: %w", err)
	}

	// attempt to load the active config file
	cfg, err := getClientConfig()
	if err != nil {
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
			"Telemetry type generation is disabled",
			slog.String("telemetry type", telemetry.String()),
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

//
// Telemetry Client Status Check
//

func Status() (status ClientStatus) {
	var exists bool
	var cfg *config.Config
	var tc *client.TelemetryClient
	var err error

	// default to being uninitialised
	status = CLIENT_UNINITIALIZED

	// check that active config exists
	exists = utils.CheckPathExists(activeConfigPath)
	if !exists {
		slog.Error(
			"Specified telemetry client config doesn't exist",
			slog.String("path", activeConfigPath),
		)
		return CLIENT_CONFIG_MISSING
	}

	// attempt to load the active config
	cfg, err = getClientConfig()
	if err != nil {
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
	tc, err = client.NewTelemetryClient(cfg)
	if err != nil {
		slog.Error(
			"Failed to setup telemetry client using provided config",
			slog.String("path", activeConfigPath),
			slog.String("error", err.Error()),
		)
		return CLIENT_MISCONFIGURED
	}

	// update status to indicate that telemetry client datastore is accessible
	status = CLIENT_DATASTORE_ACCESSIBLE

	// check that an registration is available
	if !tc.RegistrationAccessible() {
		slog.Warn("Telemetry client registration has not been setup", slog.String("path", tc.RegistrationPath()))
		return
	}

	// update status to indicate client has registration
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

//
// Telemetry Client Id Management
//

func GetTelemetryClientId() (clientId string, err error) {
	// attempt to load the active config
	cfg, err := getClientConfig()
	if err != nil {
		return
	}

	return cfg.ClientId, nil
}

func UpdateTelemetryClientId(clientId string) (err error) {
	// attempt to load the active config
	cfg, err := getClientConfig()
	if err != nil {
		return
	}

	cfg.ClientId = clientId

	if err = cfg.Save(); err != nil {
		slog.Error(
			"Failed to update telemetry clientId",
			slog.String("path", activeConfigPath),
			slog.String("clientId", clientId),
			slog.String("error", err.Error()),
		)
		err = fmt.Errorf(
			"failed to update telemetry client id %q in config %q: %w",
			clientId,
			activeConfigPath,
			err,
		)
	}

	return
}

//
// Telemetry Customer Id Management
//

func GetTelemetryCustomerId() (clientId string, err error) {
	// attempt to load the active config
	cfg, err := getClientConfig()
	if err != nil {
		return
	}

	return cfg.CustomerId, nil
}

func UpdateTelemetryCustomerId(customerId string) (err error) {
	// attempt to load the active config
	cfg, err := getClientConfig()
	if err != nil {
		return
	}

	cfg.CustomerId = customerId

	if err = cfg.Save(); err != nil {
		slog.Error(
			"Failed to update telemetry customerId",
			slog.String("path", activeConfigPath),
			slog.String("customerId", customerId),
			slog.String("error", err.Error()),
		)
		err = fmt.Errorf(
			"failed to update telemetry customer id %q in config %q: %w",
			customerId,
			activeConfigPath,
			err,
		)
	}

	return
}
