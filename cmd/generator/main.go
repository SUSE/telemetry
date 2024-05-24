package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/SUSE/telemetry/pkg/client"
	"github.com/SUSE/telemetry/pkg/config"
	"github.com/SUSE/telemetry/pkg/types"
)

// options is a struct of the options
type options struct {
	config    string
	dryrun    bool
	nobundles bool
	noreports bool
	nosubmit  bool
	tags      types.Tags
	telemetry types.TelemetryType
	jsonFiles []string
}

func (o options) String() string {
	return fmt.Sprintf("config=%v, dryrun=%v, tags=%v, telemetry=%v, jsonFiles=%v", o.config, o.dryrun, o.tags, o.telemetry, o.jsonFiles)
}

var opts options

func main() {
	fmt.Printf("Generator: %s\n", opts)

	cfg, err := config.NewConfig(opts.config)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Config: %+v\n", cfg)

	tc, err := client.NewTelemetryClient(cfg)
	if err != nil {
		log.Fatal(err)
	}

	err = tc.Register()
	if err != nil {
		log.Fatal(err)
	}

	for _, jsonFile := range opts.jsonFiles {
		jsonContent, err := os.ReadFile(jsonFile)
		if err != nil {
			log.Fatal(fmt.Errorf("error reading contents of telemetry JSON file: %s", err))
		}

		err = tc.Generate(opts.telemetry, jsonContent, opts.tags)
		if err != nil {
			log.Fatal(fmt.Errorf("error generating a telemetry data item from JSON file '%s': %s", jsonFile, err))
		}
	}

	// create one or more bundles from available data items
	if !opts.nobundles {
		if err := tc.CreateBundles(opts.tags); err != nil {
			log.Fatal(fmt.Errorf("error telemetry bundles: %s", err))
		}
	}

	// create one or more reports from available bundles
	if !opts.noreports {
		if err := tc.CreateReports(opts.tags); err != nil {
			log.Fatal(fmt.Errorf("error creating telemetry reports: %s", err))
		}
	}

	// create one or more reports from available bundles and then
	// submit available reports.
	if !opts.nosubmit {
		if err := tc.Submit(); err != nil {
			log.Fatal(fmt.Errorf("error submitting telemetry: %s", err))
		}
	}
}

func init() {
	flag.StringVar(&opts.config, "config", client.CONFIG_PATH, "Path to config file to read")
	flag.BoolVar(&opts.dryrun, "dryrun", false, "Process provided JSON files but do add them to the telemetry staging area.")
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
