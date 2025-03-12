package main

import (
	"flag"
	"fmt"
	"log/slog"

	"github.com/SUSE/telemetry/pkg/client"
	"github.com/SUSE/telemetry/pkg/config"
	"github.com/SUSE/telemetry/pkg/logging"
)

// options is a struct of the options
type options struct {
	config       string
	dryrun       bool
	noregister   bool
	authenticate bool
	debug        bool
}

var opts options

func main() {
	if err := logging.SetupBasicLogging(opts.debug); err != nil {
		panic(err)
	}

	slog.Debug("Authenticator", slog.Any("options", opts))

	cfg, err := config.NewConfig(opts.config)
	if err != nil {
		slog.Error(
			"Failed to load config",
			slog.String("config", opts.config),
			slog.String("error", err.Error()),
		)
		panic(err)
	}

	// setup logging based upon config settings
	lm := logging.NewLogManager()
	if err := lm.Config(&cfg.Logging); err != nil {
		panic(err)
	}

	// override config log level to debug if option specified
	if opts.debug {
		lm.SetLevel("DEBUG")
		slog.Debug("Debug mode enabled")
	}

	if err := lm.Setup(); err != nil {
		panic(err)
	}

	tc, err := client.NewTelemetryClient(cfg)
	if err != nil {
		slog.Error(
			"Failed to instantiate TelemetryClient",
			slog.String("config", opts.config),
			slog.String("error", err.Error()),
		)
		panic(err)
	}

	if !opts.noregister {
		err = tc.Register()
		if err != nil {
			slog.Error(
				"Failed to register TelemetryClient",
				slog.String("error", err.Error()),
			)
			panic(err)
		}
	}

	if opts.authenticate {
		err = tc.Authenticate()
		if err != nil {
			slog.Error(
				"Failed to (re)uthenticate TelemetryClient",
				slog.String("error", err.Error()),
			)
			panic(err)
		}
	}

	issuer, err := tc.AuthIssuer()
	if err != nil {
		slog.Error(
			"AuthIssuer() failed",
			slog.String("err", err.Error()),
		)
	}

	expiration, err := tc.AuthExpiration()
	if err != nil {
		slog.Error(
			"AuthExpiration() failed",
			slog.String("err", err.Error()),
		)
	}

	fmt.Printf(
		"Current Auth Token:\n  %-[1]*[2]s %[3]s\n  %-[1]*[4]s %[5]s\n  %-[1]*[6]s %[7]s\n",
		19,
		"Issuer:",
		issuer,
		"Expiration (UTC):",
		expiration.UTC().Format("2006-01-02T15:04:05.000000"),
		"Expiration (local):",
		expiration.Format("2006-01-02T15:04:05.000000Z07:00"),
	)
}

func init() {
	flag.StringVar(&opts.config, "config", config.DEF_CFG_PATH, "Path to config file to read")
	flag.BoolVar(&opts.debug, "debug", false, "Whether to enable debug level logging.")
	flag.BoolVar(&opts.dryrun, "dryrun", false, "Process provided JSON files but do add them to the telemetry staging area.")
	flag.BoolVar(&opts.noregister, "noregister", false, "Whether to skip registering the telemetry client if it is needed.")
	flag.BoolVar(&opts.authenticate, "authenticate", false, "Whether to (re)authenticate the telemetry client.")
	flag.Parse()
}
