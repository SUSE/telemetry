package types

import (
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/SUSE/telemetry/pkg/limits"
)

type TelemetryBlob struct {
	bytes []byte
}

func NewTelemetryBlob(jsonBlob []byte) *TelemetryBlob {
	return &TelemetryBlob{bytes: jsonBlob}
}

func (tb *TelemetryBlob) String() string {
	return string(tb.bytes)
}

func (tb *TelemetryBlob) Bytes() []byte {
	return tb.bytes
}

func (tb *TelemetryBlob) errNotValidJson() error {
	return fmt.Errorf("not valid JSON blob")
}

func (tb *TelemetryBlob) errNotJsonObject(err error) error {
	return fmt.Errorf("not a JSON object: %s", err.Error())
}

func (tb *TelemetryBlob) errNotVersionedObject() error {
	return fmt.Errorf("missing 'version' field in JSON object")
}

func (tb *TelemetryBlob) validJson() bool {
	return json.Valid(tb.Bytes())
}

func (tb *TelemetryBlob) Valid() error {
	var data map[string]any

	if !tb.validJson() {
		return tb.errNotValidJson()
	}

	if err := json.Unmarshal(tb.Bytes(), &data); err != nil {
		slog.Debug(
			"Not a valid JSON object",
			slog.String("blob", tb.String()),
			slog.String("error", err.Error()),
		)
		newErr := tb.errNotJsonObject(err)
		return newErr
	}

	if _, found := data["version"]; !found {
		slog.Debug("Not a valid JSON object", slog.String("blob", tb.String()))
		return tb.errNotVersionedObject()
	}

	return nil
}

func (tb *TelemetryBlob) CheckLimits() error {
	return limits.NewTelemetryDataLimits().CheckLimits(tb.Bytes())
}
