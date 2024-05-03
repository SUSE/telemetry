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

	cfg := config.NewConfig(opts.config)
	err := cfg.Load()
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
		items, err := processor.GetDataItems()
		if err != nil {
			log.Fatal(err.Error())
		}

		itemCount := len(items)
		if itemCount > 0 {
			fmt.Printf("%d Telemetry data items found.\n", len(items))
			for i, dataItem := range items {
				fmt.Printf("Data Item[%d]: %q\n", i, dataItem.Key())
			}
		}
	}

	if opts.bundles {
		bundles, err := processor.GetBundles()
		if err != nil {
			log.Fatal(err.Error())
		}

		bundleCount := len(bundles)
		if bundleCount > 0 {
			fmt.Printf("%d Telemetry bundles found.\n", len(bundles))
			for i, bundle := range bundles {
				fmt.Printf("Bundle[%d]: %q\n", i, bundle.Key())
			}
		}
	}

	if opts.reports {
		reports, err := processor.GetReports()
		if err != nil {
			log.Fatal(err.Error())
		}

		reportCount := len(reports)
		if reportCount > 0 {
			fmt.Printf("%d Telemetry reports found.\n", len(reports))
			for i, report := range reports {
				fmt.Printf("Reports[%d]: %q\n", i, report.Key())
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
