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

const (
	DEF_CONFIG = "/etc/susetelemetry/config.yaml"
)

// options is a struct of the options
type options struct {
	config    string
	dryrun    bool
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

	cfg := config.NewConfig()
	err := config.Load(opts.config, cfg)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Config: %+v\n", cfg)

	for _, jsonFile := range opts.jsonFiles {
		jsonContent, err := os.ReadFile(jsonFile)
		if err != nil {
			log.Fatal(fmt.Errorf("error reading contents of telemetry JSON file: %s", err))
		}

		err = client.Generate(opts.telemetry, jsonContent, opts.tags)
		if err != nil {
			log.Fatal(fmt.Errorf("error generating a telemetry data item from JSON file '%s': %s", jsonFile, err))
		}
	}
}

func init() {
	flag.StringVar(&opts.config, "config", DEF_CONFIG, "Path to config file to read")
	flag.BoolVar(&opts.dryrun, "dryrun", false, "Process provided JSON files but do add them to the telemetry staging area.")
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
