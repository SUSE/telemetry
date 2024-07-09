package telemetrylib

import (
	"errors"
	"log"
)

const (
	// 5MB
	TELEMETRY_DATA_MAX_SIZE = 5242880
	TELEMETRY_DATA_MIN_SIZE = 10
)

type TelemetryDataLimits struct {
	MaxSize uint64
	MinSize uint64
}

// func NewTelemetryDataLimits(data []byte) *TelemetryDataLimits {
func NewTelemetryDataLimits(data []byte) (*TelemetryDataLimits, error) {
	tdl := new(TelemetryDataLimits)

	err := tdl.Init(TELEMETRY_DATA_MIN_SIZE, TELEMETRY_DATA_MAX_SIZE, tdl, data)
	if err != nil {
		return tdl, err
	}

	return tdl, nil
}

// Init initiates a new TelemetryDataLimits instance with preset limits
func (t *TelemetryDataLimits) Init(min uint64, max uint64, tdl *TelemetryDataLimits, data []byte) error {
	tdl.SetTelemetryDataLimits(min, max)
	log.Println("Checking size limits for Telemetry Data")
	err := tdl.CheckLimits(data)
	if err != nil {
		return err
	}
	log.Println("Checks passed")
	return nil
}

// SetTelemetryDataLimits sets the limits for the telemetry data.
func (t *TelemetryDataLimits) SetTelemetryDataLimits(maxSize uint64, minSize uint64) {
	t.MaxSize = maxSize
	t.MinSize = minSize
}

// GetTelemetryDataLimits gets the current limits for the telemetry data.
func (t *TelemetryDataLimits) GetTelemetryDataLimits() TelemetryDataLimits {
	return *t
}

// CheckLimits checks the telemetry data limits
func (t *TelemetryDataLimits) CheckLimits(data []byte) error {
	dataSize := uint64(len(data))
	switch {
	case t.MinSize > t.MaxSize:
		return errors.New("min_size cannot be greater than max_size")
	case dataSize > t.MaxSize:
		return errors.New("payload size exceeds the maximum limit")
	case dataSize < t.MinSize:
		return errors.New("payload size is below the minimum limit")
	default:
		return nil
	}
}
