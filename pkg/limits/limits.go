package limits

import (
	"errors"
	"log/slog"
)

const (
	// 5MB
	TELEMETRY_DATA_MIN_SIZE uint64 = 10
	TELEMETRY_DATA_MAX_SIZE uint64 = 5242880
)

type TelemetryDataLimits struct {
	MinSize uint64
	MaxSize uint64
}

// func NewTelemetryDataLimits(data []byte) *TelemetryDataLimits {
func NewTelemetryDataLimits() *TelemetryDataLimits {
	tdl := new(TelemetryDataLimits)
	tdl.Init(TELEMETRY_DATA_MIN_SIZE, TELEMETRY_DATA_MAX_SIZE)
	return tdl
}

// Init initiates a new TelemetryDataLimits instance with preset limits
func (t *TelemetryDataLimits) Init(min uint64, max uint64) {
	t.SetTelemetryDataLimits(min, max)
}

// SetTelemetryDataLimits sets the limits for the telemetry data.
func (t *TelemetryDataLimits) SetTelemetryDataLimits(minSize uint64, maxSize uint64) {
	t.MinSize = minSize
	t.MaxSize = maxSize
}

// GetTelemetryDataLimits gets the current limits for the telemetry data.
func (t *TelemetryDataLimits) GetTelemetryDataLimits() TelemetryDataLimits {
	return *t
}

// CheckLimits checks the telemetry data limits
func (t *TelemetryDataLimits) CheckLimits(data []byte) error {
	dataSize := uint64(len(data))
	slog.Debug(
		"Checking size limits for Telemetry Data",
		slog.Uint64("Data size", dataSize),
		slog.Uint64("Max", t.MaxSize),
		slog.Uint64("Min", t.MinSize),
	)
	switch {
	case t.MinSize > t.MaxSize:
		return errors.New("min_size cannot be greater than max_size")
	case dataSize > t.MaxSize:
		return errors.New("payload size exceeds the maximum limit")
	case dataSize < t.MinSize:
		return errors.New("payload size is below the minimum limit")
	default:
		slog.Debug(
			"Acceptable telemetry data size",
			slog.Uint64("Data size", dataSize),
		)
		return nil
	}
}
