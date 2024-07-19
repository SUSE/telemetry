package main

import (
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/SUSE/telemetry"
)

func status_check_example() {
	status := telemetry.Status()
	slog.Info("Telemetry Client", slog.String("status", status.String()))
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

func generate_telemetry_example() {
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
	}
}

func main() {
	status_check_example()
	generate_telemetry_example()
}
