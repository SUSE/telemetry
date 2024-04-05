package client

import (
	"log"

	"github.com/SUSE/telemetry/pkg/config"
	"github.com/SUSE/telemetry/pkg/types"
	"github.com/SUSE/telemetrylib"
)

type TelemetryAuth struct {
	Token      types.TelemetryAuthToken `json:"token"`
	ExpiryDate types.TelemetryTimeStamp `json:"issueDate"`
}

type TelemetryClient struct {
	config  config.Config
	auth    TelemetryAuth
	Items   telemetrylib.TelemetryProcessor
	Bundles telemetrylib.TelemetryProcessor
}

func Generate(telemetry types.TelemetryType, content []byte, tags types.Tags) error {
	now := types.Now()

	// Add telemetry data item to DataItem data store
	log.Printf("Generated Telemetry:\n  TimeStamp: %s\n  Name: %q\n  Content: %+q\n  Tags: %v\n",
		now, telemetry, content, tags)

	return nil
}

func Bundle(tags types.Tags) error {
	now := types.Now()

	// Bundle existing telemetry data items found in DataItem data store into one or more bundles in the Bundle data store
	log.Printf("Bundle:\n  TimeStamp: %s\n  Tags: %v\n", now, tags)

	return nil
}

func Submit(tags types.Tags) error {
	now := types.Now()

	// Submit existing telemetry bundles found in Bundle data store as one or more telemetry reports
	log.Printf("Submit:\n  TimeStamp: %s\n  Tags: %v\n", now, tags)

	return nil
}
