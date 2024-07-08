package telemetrylib

import "errors"

type TelemetryDataLimits struct {
	MaxSize uint64
	MinSize uint64
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
