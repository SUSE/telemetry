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

func (tc *TelemetryClient) Register() (err error) {
	// get the registration, failing if it can't be retrieved
	reg, err := tc.getRegistration()
	if err != nil {
		return
	}

	// get the saved TelemetryAuth, returning success if found
	err = tc.loadTelemetryAuth()
	if err == nil {
		slog.Debug("telemetry auth found, client already registered, skipping", slog.Int64("registrationId", tc.auth.RegistrationId))
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

	tc.auth.RegistrationId = crResp.RegistrationId
	tc.auth.Token = types.TelemetryAuthToken(crResp.AuthToken)
	tc.auth.RegistrationDate, err = types.TimeStampFromString(crResp.RegistrationDate)
	if err != nil {
		slog.Error(
			"failed to parse registrationDate as a timestamp",
			slog.String("registrationDate", crResp.RegistrationDate),
			slog.String("err", err.Error()),
		)
		return
	}

	tc.authLoaded = true

	err = tc.saveTelemetryAuth()
	if err != nil {
		slog.Error(
			"failed to save TelemetryAuth",
			slog.String("err", err.Error()),
		)
		return
	}

	slog.Debug(
		"successfully registered as client",
		slog.Int64("registrationId", tc.auth.RegistrationId),
	)

	return nil
}
