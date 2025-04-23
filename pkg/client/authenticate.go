package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/SUSE/telemetry/pkg/restapi"
)

// Authenticate is responsible for (re)authenticating an already registered
// client with the server to ensure that it's auth token is up to date.
func (tc *TelemetryClient) Authenticate() (err error) {
	// get the registration, failing if it can't be retrieved
	regId, err := tc.getRegistration()
	if err != nil {
		return
	}

	// confirm that credentials already exist
	if !tc.creds.Valid() {
		return fmt.Errorf(
			"telemetry client (re-)authentication requires an existing " +
				"set of client credentials",
		)
	}

	// assemble the authentication request
	caReq := restapi.ClientAuthenticationRequest{
		RegistrationId: tc.creds.RegistrationId,
		RegHash:        *regId.Hash("default"),
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

	// check the response status code, and handle appropriately
	switch resp.StatusCode {
	case http.StatusOK:
		// all good, nothing to do
		slog.Debug(
			"Authentication request succeeded",
			slog.Int("StatusCode", resp.StatusCode),
		)

	case http.StatusUnauthorized:
		slog.Debug(
			"StatusUnauthorized returned",
			slog.Int("StatusCode", resp.StatusCode),
			slog.String("error", string(respBody)),
		)
		// retry if a unregistered client authentication attempt is detected
		if tc.reg.RetriesEnabled() {
			slog.Warn(
				"Unregistered client authentication detected, forcing registration",
			)

			// delete the existing creds, and trigger a registration attempty
			tc.creds.Remove()

			// disable further retries
			tc.creds.DisableRetries()

			// retry client registration
			return tc.Register()
		}
		fallthrough

	default:
		// unhandled error so fail appropriately
		err = fmt.Errorf("client registration failed: %s", string(respBody))
		return
	}

	// extract the creds from the response and save them
	var caResp restapi.ClientAuthenticationResponse
	err = json.Unmarshal(respBody, &caResp)
	if err != nil {
		slog.Error(
			"failed to JSON unmarshal client authentication response body content",
			slog.String("err", err.Error()),
		)
		return
	}

	// save the extracted creds
	err = tc.creds.UpdateCreds(&caResp)
	if err != nil {
		slog.Error(
			"failed to updated client credentials",
			slog.String("err", err.Error()),
		)
		return
	}

	slog.Debug(
		"successfully authenticated",
		slog.Int64("registrationId", tc.creds.RegistrationId),
	)

	return
}
