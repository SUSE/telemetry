package client

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	telemetrylib "github.com/SUSE/telemetry/pkg/lib"
	"github.com/SUSE/telemetry/pkg/restapi"
)

func (tc *TelemetryClient) submitReportInternal(report *telemetrylib.TelemetryReport) (err error) {
	// submit a telemetry report
	var trReq restapi.TelemetryReportRequest
	trReq.TelemetryReport = *report
	reqBodyJSON, err := json.Marshal(&trReq)
	if err != nil {
		slog.Error("failed to JSON marshal trReq", slog.String("err", err.Error()))
		return
	}

	reqUrl := tc.cfg.TelemetryBaseURL + "/report"
	reqBuf := bytes.NewBuffer(reqBodyJSON)
	req, err := http.NewRequest("POST", reqUrl, reqBuf)
	if err != nil {
		slog.Error("failed to create new HTTP request for telemetry report", slog.String("err", err.Error()))
		return
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", "Bearer "+tc.creds.AuthToken)
	req.Header.Add("X-Telemetry-Registration-Id", fmt.Sprintf("%d", tc.creds.RegistrationId))

	httpClient := http.DefaultClient
	resp, err := httpClient.Do(req)
	if err != nil {
		slog.Error("failed HTTP POST telemetry report request", slog.String("err", err.Error()))
		return
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		slog.Error(
			"failed to read telemetry report response body",
			slog.Int("StatusCode", resp.StatusCode),
			slog.String("error", err.Error()),
		)
		return
	}

	switch resp.StatusCode {
	case http.StatusOK:
		// nothing to do
	case http.StatusUnauthorized:
		return unauthorizedError(resp)
	default:
		slog.Error(
			"failed to submit report",
			slog.Int("StatusCode", resp.StatusCode),
			slog.String("respBody", string(respBody)),
		)
		return
	}

	var trResp restapi.TelemetryReportResponse
	err = json.Unmarshal(respBody, &trResp)
	if err != nil {
		slog.Error("failed to JSON unmarshal telemetry report response body content", slog.String("err", err.Error()))
		return
	}

	slog.Debug(
		"successfully submitted report",
		slog.String("report", report.Header.ReportId),
		slog.String("processing", trResp.ProcessingInfo()),
	)
	return
}

func (tc *TelemetryClient) submitReportRetry(
	report *telemetrylib.TelemetryReport,
	maxTries int,
	delay time.Duration,
) (err error) {
	// retry at most MaxTries times
	for retry := maxTries; retry > 0; retry -= 1 {

		// handle panic() calls as well as return
		func() {
			defer func() {
				if r := recover(); r != nil {
					switch rType := r.(type) {
					case string:
						err = errors.New(rType)
					case error:
						err = rType
					default:
						err = fmt.Errorf("unexpected recovery type: %s", rType)
					}
				}
			}()
			err = tc.submitReportInternal(report)
		}()

		if err == nil {
			break
		}

		switch {
		// check if we need to register again?
		case errors.Is(err, ErrRegistrationRequired):
			slog.Info(
				"Telemetry Client Registration Required",
				slog.String("error", err.Error()),
			)

			// force a (re-)registration by deleting any existing
			// client creds bundle
			err = tc.creds.Remove()
			if err != nil {
				slog.Warn(
					"Failed to delete existing telemetry auth bundle",
					slog.String("error", err.Error()),
				)
			}

			// register the telemetry client
			err = tc.Register()
			if err != nil {
				// if registration failed, for now don't re-try
				return
			}

			slog.Info(
				"Telemetry Client Registration Successful",
			)

		// check if we need to authenticate again?
		case errors.Is(err, ErrAuthenticationRequired):
			slog.Info(
				"Telemetry Client Authentication Required",
				slog.String("error", err.Error()),
			)

			// attempt to (re-)autenticate
			err = tc.Authenticate()
			if err != nil {
				// if authentication failed, for now don't re-try
				return
			}

			slog.Info(
				"Telemetry Client Authentication Successful",
			)

		// TODO: handle server busy backoff and retry appropriately

		default:
			slog.Debug(
				"Unhandled error",
				slog.String("error", err.Error()),
			)
		}

		// sleep between retries
		if retry > 0 {
			time.Sleep(delay)
		}

	}
	return
}

func (tc *TelemetryClient) submitReport(report *telemetrylib.TelemetryReport) (err error) {
	if err = report.Validate(); err != nil {
		slog.Error(
			"validation failure",
			slog.String("err", err.Error()),
		)
		return
	}

	// TODO: make delay configurable, or possibly supplied by the request response
	err = tc.submitReportRetry(report, 3, time.Duration(500*time.Millisecond))
	return
}
