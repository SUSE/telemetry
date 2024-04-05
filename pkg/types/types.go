package types

import (
	"fmt"
	"strings"
	"time"
)

// Tag is a string of the form "name" or "name=value"
type Tag string

func (t Tag) String() string {
	return string(t)
}

func (t Tag) HasValue() bool {
	return strings.Contains(string(t), "=")
}

func (t Tag) Valid() (bool, error) {
	parts := strings.Split(string(t), "=")
	if len(parts) > 2 {
		return false, fmt.Errorf("Tag names or values cannon include '='")
	}
	if len(parts[0]) < 3 {
		return false, fmt.Errorf("Tag name must be at least 3 characters long")
	}

	return true, nil
}

// Tags is a slice of Tag
type Tags []Tag

func (t *Tags) String() string {
	return fmt.Sprintf("%s", *t)
}

func (t *Tags) Set(value string) error {
	v := Tag(value)
	if valid, err := v.Valid(); !valid {
		return err
	}
	*t = append(*t, v)

	return nil
}

// TelemetryAuthToken is a string holding an encoded auth token value
type TelemetryAuthToken string

func (t TelemetryAuthToken) Valid() bool {
	return t != ""
}

// TelemetryType is a string of the format "<family>-<stream>-<subtype>"
type TelemetryType string

func (t TelemetryType) Valid() (bool, error) {
	parts := strings.Split(string(t), "-")
	if len(parts) < 3 {
		return false, fmt.Errorf("telemetry type should be of the format <family>-<stream>-<subtype>")
	}

	return true, nil
}

func (t *TelemetryType) String() string {
	return string(*t)
}

func (t *TelemetryType) Set(value string) error {
	v := TelemetryType(value)

	if valid, err := v.Valid(); !valid {
		return err
	}

	*t = v

	return nil
}

type TelemetryTimeStamp struct {
	time.Time
}

func (t TelemetryTimeStamp) String() string {
	return t.UTC().Format(time.RFC3339Nano)
}

func Now() TelemetryTimeStamp {
	t := TelemetryTimeStamp{time.Now()}
	return t
}
