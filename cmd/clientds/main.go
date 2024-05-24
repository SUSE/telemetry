package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/SUSE/telemetry/pkg/client"
	"github.com/SUSE/telemetry/pkg/config"
)

// options is a struct of the options
type options struct {
	config  string
	items   bool
	bundles bool
	reports bool
}

func (o options) String() string {
	return fmt.Sprintf("config=%v, items=%v, bundles=%v, reports=%v", o.config, o.items, o.bundles, o.reports)
}

var opts options

func main() {
	fmt.Printf("clientds: %s\n", opts)

	cfg, err := config.NewConfig(opts.config)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Config: %+v\n", cfg)

	tc, err := client.NewTelemetryClient(cfg)
	if err != nil {
		log.Fatal(err)
	}

	processor := tc.Processor()

	if opts.items {
		itemRows, err := processor.GetDataItemRows()
		if err != nil {
			log.Fatal(err.Error())
		}

		itemCount := len(itemRows)
		if itemCount > 0 {
			fmt.Printf("%d Telemetry data items found.\n", len(itemRows))
			for i, dataItemRow := range itemRows {
				fmt.Printf("Data Item[%d]: %q\n", i, dataItemRow.ItemId)
			}
		}
	}

	if opts.bundles {
		bundleRows, err := processor.GetBundleRows()
		if err != nil {
			log.Fatal(err.Error())
		}

		bundleCount := len(bundleRows)
		if bundleCount > 0 {
			fmt.Printf("%d Telemetry bundles found.\n", len(bundleRows))
			for i, bundleRow := range bundleRows {
				fmt.Printf("Bundle[%d]: %q\n", i, bundleRow.BundleId)
			}
		}
	}

	if opts.reports {
		reportRows, err := processor.GetReportRows()
		if err != nil {
			log.Fatal(err.Error())
		}

		reportCount := len(reportRows)
		if reportCount > 0 {
			fmt.Printf("%d Telemetry reports found.\n", len(reportRows))
			for i, reportRow := range reportRows {
				fmt.Printf("Reports[%d]: %q\n", i, reportRow.ReportId)
			}
		}
	}
}

func init() {
	flag.StringVar(&opts.config, "config", client.CONFIG_PATH, "Path to config file to read")
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
