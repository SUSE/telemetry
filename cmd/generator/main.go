package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/SUSE/telemetry/pkg/client"
	"github.com/SUSE/telemetry/pkg/config"
	"github.com/SUSE/telemetry/pkg/logging"
	"github.com/SUSE/telemetry/pkg/types"
)

// options is a struct of the options
type options struct {
	config       string
	dryrun       bool
	noregister   bool
	authenticate bool
	nobundles    bool
	noreports    bool
	nosubmit     bool
	tags         types.Tags
	telemetry    types.TelemetryType
	jsonFiles    []string
	debug        bool
}

var opts options

func main() {
	if err := logging.SetupBasicLogging(opts.debug); err != nil {
		panic(err)
	}

	slog.Debug("Generator", slog.Any("options", opts))

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

	for _, jsonFile := range opts.jsonFiles {
		jsonContent, err := os.ReadFile(jsonFile)
		if err != nil {
			slog.Error(
				"Error reading telemetry data",
				slog.String("jsonFile", jsonFile),
				slog.String("error", err.Error()),
			)
			panic(err)
		}

		err = tc.Generate(opts.telemetry, types.NewTelemetryBlob(jsonContent), opts.tags)
		if err != nil {
			slog.Error(
				"Error generating telemetry data item",
				slog.String("jsonFile", jsonFile),
				slog.String("error", err.Error()),
			)
			panic(err)
		}

		fmt.Printf(
			"Added telemetry data from %q as type %q with tags %s to local datastore\n",
			filepath.Base(jsonFile),
			opts.telemetry,
			opts.tags,
		)
	}

	// create one or more bundles from available data items
	if !opts.nobundles {
		if err := tc.CreateBundles(opts.tags); err != nil {
			slog.Error(
				"Error creating telemetry bundles",
				slog.String("error", err.Error()),
			)
			panic(err)
		}

		fmt.Println("Created telemetry bundles from pending telemetry data items")
	}

	// create one or more reports from available bundles
	if !opts.noreports {
		if err := tc.CreateReports(opts.tags); err != nil {
			slog.Error(
				"Error creating telemetry reports",
				slog.String("error", err.Error()),
			)
			panic(err)
		}

		fmt.Println("Created telemetry reports from pending telemetry bundles")
	}

	// create one or more reports from available bundles and then
	// submit available reports.
	if !opts.nosubmit {
		if err := tc.Submit(); err != nil {
			slog.Error(
				"Error submitting telemetry",
				slog.String("error", err.Error()),
			)
			panic(err)
		}

		fmt.Println("Submitted pending telemetry reports")
	}
}

func init() {
	flag.StringVar(&opts.config, "config", client.CONFIG_PATH, "Path to config file to read")
	flag.BoolVar(&opts.debug, "debug", false, "Whether to enable debug level logging.")
	flag.BoolVar(&opts.dryrun, "dryrun", false, "Process provided JSON files but do add them to the telemetry staging area.")
	flag.BoolVar(&opts.noregister, "noregister", false, "Whether to skip registering the telemetry client if it is needed.")
	flag.BoolVar(&opts.authenticate, "authenticate", false, "Whether to (re)authenticate the telemetry client.")
	flag.BoolVar(&opts.noreports, "noreports", false, "Do not create Telemetry reports")
	flag.BoolVar(&opts.nobundles, "nobundles", false, "Do not create Telemetry bundles")
	flag.BoolVar(&opts.nosubmit, "nosubmit", false, "Do not submit any Telemetry reports")
	flag.Var(&opts.tags, "tag", "Optional tags to be associated with the submitted telemetry data")
	flag.Var(&opts.telemetry, "telemetry", "The type of the telemetry being submitted")
	flag.Parse()

	if (opts.telemetry) == "" {
		fmt.Fprintln(os.Stderr, "Error: No value specified for '-telemetry'.")
		flag.Usage()
		os.Exit(1)
	}

	if flag.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "Error: No JSON files were specified as arguments.")
		flag.Usage()
		os.Exit(1)
	}
	opts.jsonFiles = flag.Args()
}
