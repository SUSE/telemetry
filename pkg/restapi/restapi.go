package restapi

import (
	"encoding/json"
	"fmt"

	telemetrylib "github.com/SUSE/telemetry/pkg/lib"
	"github.com/SUSE/telemetry/pkg/types"
	"github.com/go-playground/validator/v10"
)

//
// Client Registration Handling via /telemetry/register POST
//

// ClientRegistrationRequest is the request payload body POST'd to the server
type ClientRegistrationRequest struct {
	ClientRegistration types.ClientRegistration `json:"clientRegistration" validate:"required"`
}

func (c *ClientRegistrationRequest) String() string {
	bytes, _ := json.Marshal(c)

	return string(bytes)
}

func (c *ClientRegistrationRequest) Validate() (err error) {
	validate := validator.New(validator.WithRequiredStructEnabled())

	err = validate.Struct(c)
	if err != nil {
		err = fmt.Errorf("client registration validation check failed: %w", err)
	}

	return
}

// ClientRegistrationResponse is the response payload body from the server
type ClientRegistrationResponse struct {
	RegistrationId   int64  `json:"registrationId" validate:"required,min=1"`
	AuthToken        string `json:"authToken" validate:"required,jwt"`
	RegistrationDate string `json:"registrationDate" validate:"required,datetime=2006-01-02T15:04:05.999999999Z07:00"`
}

func (c *ClientRegistrationResponse) String() string {
	bytes, _ := json.Marshal(c)

	return string(bytes)
}

func (c *ClientRegistrationResponse) Validate() (err error) {
	validate := validator.New()

	err = validate.Struct(c)
	if err != nil {
		err = fmt.Errorf("client credentials validation check failed: %w", err)
	}

	return
}

// Client Authenticate handling via /temelemtry/authenticate
type ClientAuthenticationRequest struct {
	RegistrationId int64                        `json:"registrationId" validate:"required,min=1"`
	RegHash        types.ClientRegistrationHash `json:"regHash" validate:"required"`
}

func (c *ClientAuthenticationRequest) String() string {
	bytes, _ := json.Marshal(c)

	return string(bytes)
}

func (c *ClientAuthenticationRequest) Validate() (err error) {
	validate := validator.New()

	err = validate.Struct(c)
	if err != nil {
		err = fmt.Errorf("client credentials validation check failed: %w", err)
	}

	return
}

// for now the /authenticate response is the same as the /register
// response
type ClientAuthenticationResponse = ClientRegistrationResponse

//
// Client Telemetry Report via /telemetry/report POST
//

// TelemetryReportRequest is the request payload body POST'd to the server
type TelemetryReportRequest struct {
	telemetrylib.TelemetryReport
}

func (t *TelemetryReportRequest) String() string {
	bytes, _ := json.Marshal(t)

	return string(bytes)
}

// TelemetryReportResponse is the response payload body from the server
type TelemetryReportResponse struct {
	ProcessingId int64                    `json:"processingId" validate:"min=0"`
	ProcessedAt  types.TelemetryTimeStamp `json:"processedAt" validate:"required"`
}

func NewTelemetryReportResponse(procId int64, procAt types.TelemetryTimeStamp) *TelemetryReportResponse {
	trr := &TelemetryReportResponse{
		ProcessingId: procId,
		ProcessedAt:  procAt,
	}

	return trr
}

func (t *TelemetryReportResponse) String() string {
	bytes, _ := json.Marshal(t)

	return string(bytes)
}

func (t *TelemetryReportResponse) ProcessingInfo() string {
	return fmt.Sprintf("%x@%s", t.ProcessingId, t.ProcessedAt)
}

func (t *TelemetryReportResponse) Validate() (err error) {
	validate := validator.New()

	err = validate.Struct(t)
	if err != nil {
		err = fmt.Errorf("report response validation check failed: %w", err)
	}

	return
}
