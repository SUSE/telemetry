package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/SUSE/telemetry/pkg/restapi"
	"github.com/SUSE/telemetry/pkg/types"
)

// Authenticate is responsible for (re)authenticating an already registered
// client with the server to ensure that it's auth token is up to date.
func (tc *TelemetryClient) Authenticate() (err error) {
	if err = tc.loadTelemetryAuth(); err != nil {
		return fmt.Errorf(
			"telemetry client (re-)authentication requires an existing "+
				"client registration: %s",
			err.Error(),
		)
	}

	// get the instanceId, failing if it can't be retrieved
	instId, err := tc.getInstanceId()
	if err != nil {
		return
	}

	// assemble the authentication request
	caReq := restapi.ClientAuthenticationRequest{
		ClientId:   tc.auth.ClientId,
		InstIdHash: *instId.Hash("default"),
	}

	reqBodyJSON, err := json.Marshal(&caReq)
	if err != nil {
		slog.Error(
			"failed to JSON marshal caReq",
			slog.String("err", err.Error()),
		)
		return
	}

	reqUrl := tc.cfg.TelemetryBaseURL + "/authenticate"
	reqBuf := bytes.NewBuffer(reqBodyJSON)
	req, err := http.NewRequest("POST", reqUrl, reqBuf)
	if err != nil {
		slog.Error(
			"failed to create new HTTP request for client authentication",
			slog.String("err", err.Error()),
		)
		return
	}

	req.Header.Add("Content-Type", "application/json")

	httpClient := http.DefaultClient
	resp, err := httpClient.Do(req)
	if err != nil {
		slog.Error(
			"failed to HTTP POST client authentication request",
			slog.String("err", err.Error()),
		)
		return
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		slog.Error(
			"failed to read client authentication response body",
			slog.String("err", err.Error()),
		)
		return
	}

	if resp.StatusCode != http.StatusOK {
		err = fmt.Errorf("client authentication failed: %s", string(respBody))
		return
	}

	var caResp restapi.ClientAuthenticationResponse
	err = json.Unmarshal(respBody, &caResp)
	if err != nil {
		slog.Error(
			"failed to JSON unmarshal client authentication response body content",
			slog.String("err", err.Error()),
		)
		return
	}

	tc.auth.ClientId = caResp.ClientId
	tc.auth.Token = types.TelemetryAuthToken(caResp.AuthToken)
	tc.auth.RegistrationDate, err = types.TimeStampFromString(caResp.RegistrationDate)
	if err != nil {
		slog.Error(
			"failed to parse registrationDate as a timestamp",
			slog.String("registrationDate", caResp.RegistrationDate),
			slog.String("err", err.Error()),
		)
		return
	}

	err = tc.saveTelemetryAuth()
	if err != nil {
		slog.Error(
			"failed to save TelemetryAuth",
			slog.String("err", err.Error()),
		)
		return
	}

	slog.Debug(
		"successfully authenticated",
		slog.Int64("clientId", tc.auth.ClientId),
	)

	return
}
