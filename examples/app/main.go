package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"os"

	"github.com/SUSE/telemetry"
)

func telemetry_ready_check() {
	status := telemetry.Status()

	// if the client is ready then return
	if status.Ready() {
		goto clientReady
	}

	// if telemetry is disabled, exit with success
	if status.Disabled() {
		slog.Info(
			"Telemetry client disabled, exiting",
			slog.String("status", status.String()),
		)
		os.Exit(0)
	}

	// attempt to register if needed
	if status.RegistrationRequired() {
		slog.Warn(
			"Telemetry client registration required",
			slog.String("status", status.String()),
		)
		if err := telemetry.Register(); err != nil {
			slog.Error(
				"Failed to register as telemetry client",
				slog.String("error", err.Error()),
			)
			os.Exit(1)
		}
		slog.Info("Telemetry client registered")
		status = telemetry.Status()
	}

	// exit if the client is not ready
	if !status.Ready() {
		slog.Error(
			"Telemetry client not ready",
			slog.String("status", status.String()),
		)
		os.Exit(1)
	}

clientReady:
	slog.Info(
		"Telemetry client ready",
		slog.String("status", status.String()),
	)
}

type SomeTelemetryType struct {
	Version     int      `json:"version"`
	Application string   `json:"application"`
	Fields      []string `josn:"fields"`
}

func (stt *SomeTelemetryType) String() string {
	return fmt.Sprintf("%+v", *stt)
}

func telemetry_content() []byte {
	appTelemetry := SomeTelemetryType{
		Version:     1,
		Application: "specific",
		Fields: []string{
			"Of",
			"Data",
		},
	}

	content, err := json.Marshal(appTelemetry)
	if err != nil {
		slog.Error(
			"Failed to json.Marshal() content",
			slog.Any("telemetry", appTelemetry),
			slog.String("error", err.Error()),
		)
		panic(fmt.Errorf("json.Marshal() failed: %s", err.Error()))
	}

	return content
}

func telemetry_generate() {
	telemetryType := telemetry.TelemetryType("SOME-TELEMETRY-TYPE")
	class := telemetry.MANDATORY_TELEMETRY
	content := telemetry_content()
	tags := telemetry.Tags{}
	flags := telemetry.GENERATE | telemetry.SUBMIT

	// verify the telemetry type
	if valid, err := telemetryType.Valid(); !valid {
		slog.Error(
			"Invalid telemetry type",
			slog.Any("type", telemetryType),
			slog.String("error", err.Error()),
		)
		return
	}

	err := telemetry.Generate(
		telemetryType,
		class,
		content,
		tags,
		flags)
	if err != nil {
		slog.Error(
			"Generate() failed",
			slog.String("type", telemetryType.String()),
			slog.String("class", class.String()),
			slog.Any("content", content),
			slog.String("tags", tags.String()),
			slog.String("flags", flags.String()),
			slog.String("error", err.Error()),
		)
		os.Exit(1)
	}
}

type options struct {
	config string
}

func parse_opts(opts *options) {
	flag.StringVar(&opts.config, "config", telemetry.DefaultConfigPath(), "Path to alternate config file")
	flag.Parse()
}

func main() {
	// parse the config option if specified
	opts := new(options)
	parse_opts(opts)
	if opts.config != telemetry.DefaultConfigPath() {
		// tell the telemetry library to use specified config file
		slog.Info(
			"Setting customer telemetry config path",
			slog.String("config", opts.config),
		)
		telemetry.SetConfigPath(opts.config)
	}

	// verify that the telemetry subsystem is ready to use
	telemetry_ready_check()

	// generate and submit telemetry
	telemetry_generate()
}
