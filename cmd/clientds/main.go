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
	config  string
	items   bool
	bundles bool
	reports bool
	debug   bool
}

var opts options

func main() {
	slog.Debug(
		"clientds",
		slog.Any("options", opts),
	)

	if err := logging.SetupBasicLogging(opts.debug); err != nil {
		panic(err)
	}

	cfg, err := config.NewConfig(opts.config)
	if err != nil {
		slog.Error(
			"Failed to load specified config",
			slog.String("config", opts.config),
			slog.String("Error", err.Error()),
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
			slog.String("Error", err.Error()),
		)
		panic(err)
	}

	processor := tc.Processor()

	// this will be toggled to true if items, bundles or reports were found
	foundEntries := false

	if opts.items {
		itemRows, err := processor.GetItemRows()
		if err != nil {
			slog.Error(
				"Failed to retrieve items from client datastore",
				slog.String("error", err.Error()),
			)
			panic(err)
		}

		itemCount := len(itemRows)
		if itemCount > 0 {
			fmt.Printf("%d Telemetry data items found.\n", len(itemRows))
			for i, dataItemRow := range itemRows {
				fmt.Printf("Data Item[%d]: %q\n", i, dataItemRow.ItemId)
			}

			foundEntries = true
		}
	}

	if opts.bundles {
		bundleRows, err := processor.GetBundleRows()
		if err != nil {
			slog.Error(
				"Failed to retrieve bundles from client datastore",
				slog.String("error", err.Error()),
			)
			panic(err)
		}

		bundleCount := len(bundleRows)
		if bundleCount > 0 {
			fmt.Printf("%d Telemetry bundles found.\n", len(bundleRows))
			for i, bundleRow := range bundleRows {
				fmt.Printf("Bundle[%d]: %q\n", i, bundleRow.BundleId)
			}

			foundEntries = true
		}
	}

	if opts.reports {
		reportRows, err := processor.GetReportRows()
		if err != nil {
			slog.Error(
				"Failed to retrieve reports from client datastore",
				slog.String("error", err.Error()),
			)
			panic(err)
		}

		reportCount := len(reportRows)
		if reportCount > 0 {
			fmt.Printf("%d Telemetry reports found.\n", len(reportRows))
			for i, reportRow := range reportRows {
				fmt.Printf("Reports[%d]: %q\n", i, reportRow.ReportId)
			}

			foundEntries = true
		}
	}

	if !foundEntries {
		fmt.Println("No items, bundles or reports found in client datastore")
	}
}

func init() {
	flag.BoolVar(&opts.debug, "debug", false, "Enable debug level logging")
	flag.StringVar(&opts.config, "config", config.DEF_CFG_PATH, "Path to config file to read")
	flag.BoolVar(&opts.items, "items", false, "Report details on telemetry data items datastore")
	flag.BoolVar(&opts.bundles, "bundles", false, "Report details on telemetry bundles datastore")
	flag.BoolVar(&opts.reports, "reports", false, "Report details on telemetry reports datastore")
	flag.Parse()

	if !(opts.items || opts.bundles || opts.reports) {
		opts.items = true
		opts.bundles = true
		opts.reports = true
	}
}
