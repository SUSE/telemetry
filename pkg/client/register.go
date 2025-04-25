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

func (tc *TelemetryClient) Register() (err error) {
	// get the registration, failing if it can't be retrieved
	reg, err := tc.getRegistration()
	if err != nil {
		return
	}

	// if credentials are valid, the client is already registered
	if tc.creds.Valid() {
		slog.Debug(
			"valid existing credentials loaded, client already registered, skipping",
			slog.Int64("registrationId", tc.creds.RegistrationId),
		)
		return
	}

	// register the system as a client
	var crReq restapi.ClientRegistrationRequest
	crReq.ClientRegistration = reg
	reqBodyJSON, err := json.Marshal(&crReq)
	if err != nil {
		slog.Error(
			"failed to JSON marshal crReq",
			slog.String("err", err.Error()),
		)
		return
	}

	reqUrl := tc.cfg.TelemetryBaseURL + "/register"
	reqBuf := bytes.NewBuffer(reqBodyJSON)
	req, err := http.NewRequest("POST", reqUrl, reqBuf)
	if err != nil {
		slog.Error(
			"failed to create new HTTP request for client registration",
			slog.String("err", err.Error()),
		)
		return
	}

	req.Header.Add("Content-Type", "application/json")

	httpClient := http.DefaultClient
	resp, err := httpClient.Do(req)
	if err != nil {
		slog.Error(
			"failed to HTTP POST client registration request",
			slog.String("err", err.Error()),
		)
		return
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		slog.Error(
			"failed to read client registration response body",
			slog.String("err", err.Error()),
		)
		return
	}

	// check the response status code, and handle appropriately
	switch resp.StatusCode {
	case http.StatusOK:
		// all good, nothing to do

	case http.StatusConflict:
		slog.Debug(
			"StatusConflict returned",
			slog.Int("StatusCode", resp.StatusCode),
			slog.String("error", string(respBody)),
		)
		// retry if a duplicate client registration attempt is detected
		if tc.reg.RetriesEnabled() {
			slog.Warn(
				"Duplicate client registration detected, forcing re-registration",
			)

			// delete the existing registration, forcing it to be regenerated as
			// part of the next client registration attempt
			tc.reg.Remove()

			// disable further retries
			tc.reg.DisableRetries()

			// retry client registration
			return tc.Register()
		}
		fallthrough

	default:
		// unhandled error so fail appropriately
		err = fmt.Errorf("client registration failed: %s", string(respBody))
		return
	}

	var crResp restapi.ClientRegistrationResponse
	err = json.Unmarshal(respBody, &crResp)
	if err != nil {
		slog.Error(
			"failed to JSON unmarshal client registration response body content",
			slog.String("err", err.Error()),
		)
		return
	}

	err = tc.creds.UpdateCreds(&crResp)
	if err != nil {
		slog.Error(
			"failed to update client credentials",
			slog.String("path", tc.creds.Path()),
			slog.String("err", err.Error()),
		)
		return
	}

	slog.Debug(
		"successfully registered as client",
		slog.Int64("registrationId", tc.creds.RegistrationId),
	)

	return nil
}
