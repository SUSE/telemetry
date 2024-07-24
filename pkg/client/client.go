package client

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"os"

	"github.com/SUSE/telemetry/pkg/config"
	telemetrylib "github.com/SUSE/telemetry/pkg/lib"
	"github.com/SUSE/telemetry/pkg/restapi"
	"github.com/SUSE/telemetry/pkg/types"
)

const (
	//CONFIG_DIR  = "/etc/susetelemetry"
	CONFIG_DIR      = "/tmp/susetelemetry"
	CONFIG_PATH     = CONFIG_DIR + "/config.yaml"
	AUTH_PATH       = CONFIG_DIR + "/auth.json"
	INSTANCEID_PATH = CONFIG_DIR + "/instanceid"
)

type TelemetryAuth struct {
	ClientId  int64                    `json:"clientId"`
	Token     types.TelemetryAuthToken `json:"token"`
	IssueDate types.TelemetryTimeStamp `json:"issueDate"`
}

type TelemetryClient struct {
	cfg       *config.Config
	auth      TelemetryAuth
	processor telemetrylib.TelemetryProcessor
}

func NewTelemetryClient(cfg *config.Config) (tc *TelemetryClient, err error) {
	tc = &TelemetryClient{cfg: cfg}
	tc.processor, err = telemetrylib.NewTelemetryProcessor(&cfg.DataStores)
	return
}

func checkFileExists(filePath string) bool {
	slog.Debug("checking for existence", slog.String("filePath", filePath))

	if _, err := os.Stat(filePath); err != nil {
		if !errors.Is(err, fs.ErrExist) {
			slog.Debug(
				"failed to stat path",
				slog.String("filePath", filePath),
				slog.String("error", err.Error()),
			)
			return false
		}
	}

	return true
}

func checkFileReadAccessible(filePath string) bool {
	if _, err := os.Open(filePath); err != nil {
		return false
	}
	return true
}

func ensureInstanceIdExists(instIdPath string) error {

	slog.Info("ensuring existence of instIdPath", slog.String("instIdPath", instIdPath))
	_, err := os.Stat(instIdPath)
	if !os.IsNotExist(err) {
		return nil
	}

	// For now generate an instanceId as a base64 encoded timestamp
	now := types.Now().String()
	instId := make([]byte, base64.StdEncoding.EncodedLen(len(now)))
	base64.StdEncoding.Encode(instId, []byte(now))

	err = os.WriteFile(instIdPath, instId, 0600)
	if err != nil {
		slog.Error(
			"failed to write instId to instIdPath",
			slog.String("instId", string(instId)),
			slog.String("instIdPath", instIdPath),
			slog.String("err", err.Error()),
		)
	}

	return nil
}

func (tc *TelemetryClient) AuthAccessible() bool {
	return checkFileReadAccessible(tc.AuthPath())
}

func (tc *TelemetryClient) InstanceIdAccessible() bool {
	return checkFileReadAccessible(tc.InstIdPath())
}

func (tc *TelemetryClient) HasAuth() bool {
	return checkFileExists(tc.AuthPath())
}

func (tc *TelemetryClient) HasInstanceId() bool {
	return checkFileExists(tc.InstIdPath())
}

func (tc *TelemetryClient) Processor() telemetrylib.TelemetryProcessor {
	// may want to just make the processor a public field
	return tc.processor
}

func (tc *TelemetryClient) AuthPath() string {
	// hard coded for now, possibly make a config option
	return AUTH_PATH
}

func (tc *TelemetryClient) InstIdPath() string {
	// hard coded for now, possibly make a config option
	return INSTANCEID_PATH
}

func (tc *TelemetryClient) getInstanceId() (instId types.ClientInstanceId, err error) {
	instIdPath := tc.InstIdPath()

	err = ensureInstanceIdExists(instIdPath)
	if err != nil {
		return
	}

	data, err := os.ReadFile(instIdPath)
	if err != nil {
		slog.Error(
			"failed to read instId file",
			slog.String("path", instIdPath),
			slog.String("err", err.Error()),
		)
		return
	}

	instId = types.ClientInstanceId((data))

	slog.Debug(
		"successfully read instId file",
		slog.String("path", string(instIdPath)),
		slog.String("instId", string(instId)),
	)

	return
}

func (tc *TelemetryClient) loadTelemetryAuth() (err error) {
	authPath := tc.AuthPath()

	slog.Info("Checking auth file existence", slog.String("authPath", authPath))
	_, err = os.Stat(authPath)
	if os.IsNotExist(err) {
		slog.Warn(
			"unable to find auth file",
			slog.String("authPath", authPath),
			slog.String("err", err.Error()),
		)
		return
	}

	authContent, err := os.ReadFile(authPath)
	if err != nil {
		slog.Error(
			"failed to read contents of auth file",
			slog.String("authPath", authPath),
			slog.String("err", err.Error()),
		)
		return
	}

	err = json.Unmarshal(authContent, &tc.auth)
	if err != nil {
		slog.Error(
			"failed to JSON unmarshal auth file contents",
			slog.String("authPath", authPath),
			slog.String("authContent", string(authContent)),
			slog.String("err", err.Error()),
		)
		return
	}

	if tc.auth.ClientId <= 0 {
		err = fmt.Errorf("invalid client id")
		slog.Error(
			"invalid auth",
			slog.String("authPath", authPath),
			slog.String("authContent", string(authContent)),
			slog.String("err", err.Error()),
		)
		return
	}

	if tc.auth.Token == "" {
		err = fmt.Errorf("empty token value")
		slog.Error(
			"invalid auth",
			slog.String("authPath", authPath),
			slog.String("authContent", string(authContent)),
			slog.String("err", err.Error()),
		)
		return
	}

	return
}

func (tc *TelemetryClient) saveTelemetryAuth() (err error) {
	authPath := tc.AuthPath()

	taJSON, err := json.Marshal(&tc.auth)
	if err != nil {
		slog.Error("failed to JSON marshal TelemetryAuth", slog.String("err", err.Error()))
		return
	}

	err = os.WriteFile(authPath, taJSON, 0600)
	if err != nil {
		slog.Error(
			"failed to write JSON marshalled TelemetryAuth",
			slog.String("authPath", authPath),
			slog.String("err", err.Error()),
		)
	}

	return
}

func (tc *TelemetryClient) submitReport(report *telemetrylib.TelemetryReport) (err error) {
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
	req.Header.Add("Authorization", "Bearer "+tc.auth.Token.String())
	req.Header.Add("X-Telemetry-Client-Id", fmt.Sprintf("%d", tc.auth.ClientId))

	httpClient := http.DefaultClient
	resp, err := httpClient.Do(req)
	if err != nil {
		slog.Error("failed HTTP POST telemetry report request", slog.String("err", err.Error()))
		return
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		slog.Error("failed to read telemetry report response body", slog.String("err", err.Error()))
		return
	}

	if resp.StatusCode != http.StatusOK {
		slog.Error("failed to submit report", slog.String("respBody", string(respBody)))
		return
	}

	var trResp restapi.TelemetryReportResponse
	err = json.Unmarshal(respBody, &trResp)
	if err != nil {
		slog.Error("failed to JSON unmarshal telemetry report response body content", slog.String("err", err.Error()))
		return
	}

	slog.Info(
		"successfully submitted report",
		slog.String("report", report.Header.ReportId),
		slog.String("processing", trResp.ProcessingInfo()),
	)
	return
}

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
	tc.auth.IssueDate, err = types.TimeStampFromString(caResp.IssueDate)
	if err != nil {
		slog.Error(
			"failed to parse issueDate as a timestamp",
			slog.String("issueDate", caResp.IssueDate),
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

	slog.Info(
		"successfully authenticated",
		slog.Int64("clientId", tc.auth.ClientId),
	)

	return
}

func (tc *TelemetryClient) Register() (err error) {
	// get the saved TelemetryAuth, returning success if found
	err = tc.loadTelemetryAuth()
	if err == nil {
		slog.Info("telemetry auth found, client already registered, skipping", slog.Int64("clientId", tc.auth.ClientId))
		return
	}

	// get the instanceId, failing if it can't be retrieved
	instId, err := tc.getInstanceId()
	if err != nil {
		return
	}

	// register the system as a client
	var crReq restapi.ClientRegistrationRequest
	crReq.ClientInstanceId = instId
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

	// TODO: Handle http.StatusConflict (409) as needing to regenerate instId
	if resp.StatusCode != http.StatusOK {
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

	tc.auth.ClientId = crResp.ClientId
	tc.auth.Token = types.TelemetryAuthToken(crResp.AuthToken)
	tc.auth.IssueDate, err = types.TimeStampFromString(crResp.IssueDate)
	if err != nil {
		slog.Error(
			"failed to parse issueDate as a timestamp",
			slog.String("issueDate", crResp.IssueDate),
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

	slog.Info(
		"successfully registered as client",
		slog.Int64("clientId", tc.auth.ClientId),
	)

	return nil
}

func (tc *TelemetryClient) Generate(telemetry types.TelemetryType, content []byte, tags types.Tags) error {
	// Enforce size limits
	tdl := telemetrylib.NewTelemetryDataLimits()
	err := tdl.CheckLimits(content)
	if err != nil {
		return err
	}

	// Add telemetry data item to DataItem data store
	slog.Info(
		"Generated Telemetry",
		slog.String("name", telemetry.String()),
		slog.String("tags", tags.String()),
		slog.String("content", string(content)),
	)

	return tc.processor.AddData(telemetry, content, tags)
}

func (tc *TelemetryClient) CreateBundles(tags types.Tags) error {
	// Bundle existing telemetry data items found in DataItem data store into one or more bundles in the Bundle data store
	slog.Info("Bundle", slog.String("Tags", tags.String()))
	tc.processor.GenerateBundle(tc.auth.ClientId, tc.cfg.CustomerID, tags)

	return nil
}

func (tc *TelemetryClient) CreateReports(tags types.Tags) (err error) {
	// Generate reports from available bundles
	slog.Info("CreateReports", slog.String("Tags", tags.String()))
	tc.processor.GenerateReport(tc.auth.ClientId, tags)

	return
}

func (tc *TelemetryClient) Submit() (err error) {
	// fail if the client is not registered

	err = tc.loadTelemetryAuth()
	if err != nil {
		return
	}

	// retrieve available reports
	reportRows, err := tc.processor.GetReportRows()
	if err != nil {
		return
	}

	// submit each available report
	for _, reportRow := range reportRows {

		report, err := tc.processor.ToReport(reportRow)
		if err != nil {
			return fmt.Errorf("failed to convert report %q: %s", reportRow.ReportId, err.Error())
		}

		if err := tc.submitReport(&report); err != nil {
			return fmt.Errorf("failed to submit report %q: %s", report.Header.ReportId, err.Error())
		}

		// delete the successfully submitted report
		tc.processor.DeleteReport(reportRow)
	}

	return nil
}
