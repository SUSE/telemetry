package types

import (
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"hash"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
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

func (t *TelemetryAuthToken) String() string {
	return string(*t)
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

func TimeStampFromString(tsString string) (ts TelemetryTimeStamp, err error) {
	t, err := time.Parse(time.RFC3339Nano, tsString)
	ts = TelemetryTimeStamp{t}
	return
}

func Now() TelemetryTimeStamp {
	t := TelemetryTimeStamp{time.Now()}
	return t
}

type TelemetryClass int64

const (
	MANDATORY_TELEMETRY TelemetryClass = iota
	OPT_OUT_TELEMETRY
	OPT_IN_TELEMETRY
)

func (tc *TelemetryClass) String() string {
	switch *tc {
	case MANDATORY_TELEMETRY:
		return "MANDATORY"
	case OPT_OUT_TELEMETRY:
		return "OPT-OUT"
	case OPT_IN_TELEMETRY:
		return "OPT-IN"
	}
	return "UNKNOWN_TELEMETRY_CLASS"
}

type ClientRegistrationHash struct {
	Method string `json:"method"`
	Value  string `json:"value"`
}

func (c *ClientRegistrationHash) String() string {
	bytes, _ := json.Marshal(c)

	return string(bytes)
}

func (c *ClientRegistrationHash) Match(m *ClientRegistrationHash) bool {
	return (c.Method == m.Method) && (c.Value == m.Value)
}

// ClientRegistration
type ClientRegistration struct {
	ClientId   string `json:"clientId" validate:"required,uuid|uuid_rfc4122"`
	SystemUUID string `json:"systemUUID" validate:"omitempty,gt=0,uuid|uuid_rfc4122"`
	Timestamp  string `json:"timestamp" validate:"required,datetime=2006-01-02T15:04:05.999999999Z07:00"`
}

func (c *ClientRegistration) String() string {
	bytes, _ := json.Marshal(c)
	return string(bytes)
}

func (c *ClientRegistration) Validate() (err error) {
	validate := validator.New()

	err = validate.Struct(c)
	if err != nil {
		err = fmt.Errorf("client registration validation check failed: %w", err)
	}

	return
}

const DEF_INSTID_HASH_METHOD = "sha256"

func (c *ClientRegistration) Hash(inputMethod string) *ClientRegistrationHash {
	var methodHash hash.Hash

	// this routine is expected to succeed so ensure a valid method is used
	method := strings.ToLower(inputMethod)
	switch method {
	case "sha256":
		// valid supported method
	case "sha512":
		// valid supported method
	case "default":
		// use the default method
		fallthrough
	default:
		// set the method to the desired default method
		method = DEF_INSTID_HASH_METHOD
	}

	// instantiate methodHash associated with selected method and
	// write the instanced id data to it
	switch method {
	case "sha256":
		methodHash = sha256.New()
	case "sha512":
		methodHash = sha512.New()
	}
	methodHash.Write([]byte(c.String()))

	// construct the return value
	return &ClientRegistrationHash{
		Method: method,
		Value:  hex.EncodeToString(methodHash.Sum(nil)),
	}
}
