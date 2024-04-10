package restapi

import (
	"encoding/json"

	"github.com/SUSE/telemetry/pkg/types"
	"github.com/SUSE/telemetrylib"
)

//
// Client Registration Handling via /telemetry/register POST
//

// ClientRegistrationRequest is the request payload body POST'd to the server
type ClientRegistrationRequest struct {
	ClientInstanceId string `json:"clientInstanceId"`
}

func (c *ClientRegistrationRequest) String() string {
	bytes, _ := json.Marshal(c)

	return string(bytes)
}

// ClientRegistrationResponse is the response payload body from the server
type ClientRegistrationResponse struct {
	ClientId  int64  `json:"clientId"`
	AuthToken string `json:"authToken"`
	IssueDate string `json:"issueDate"`
}

func (c *ClientRegistrationResponse) String() string {
	bytes, _ := json.Marshal(c)

	return string(bytes)
}

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
	ProcessingId int64                    `json:"processingId"`
	ProcessedAt  types.TelemetryTimeStamp `json:"processedAt"`
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
