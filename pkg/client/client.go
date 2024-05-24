package client

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
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

func ensureInstanceIdExists(instIdPath string) error {

	log.Printf("ensuring existence of instIdPath %q", instIdPath)
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
		log.Printf("failed to write instId %q to instIdPath %q: %s", instId, instIdPath, err.Error())
	}

	return nil
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

func (tc *TelemetryClient) getInstanceId() (instId []byte, err error) {
	instIdPath := tc.InstIdPath()

	err = ensureInstanceIdExists(instIdPath)
	if err != nil {
		return
	}

	instId, err = os.ReadFile(instIdPath)
	if err != nil {
		log.Printf("failed to read %q: %s", instIdPath, err.Error())
	} else {
		log.Printf("instId: %q", instId)
	}

	return
}

func (tc *TelemetryClient) loadTelemetryAuth() (err error) {
	authPath := tc.AuthPath()

	log.Printf("checking for existence of authPath %q", authPath)
	_, err = os.Stat(authPath)
	if os.IsNotExist(err) {
		log.Printf("unable to find authPath %q: %s", authPath, err.Error())
		return
	}

	authContent, err := os.ReadFile(authPath)
	if err != nil {
		log.Printf("failed to read contents of authPath %q: %s", authPath, err.Error())
		return
	}

	err = json.Unmarshal(authContent, &tc.auth)
	if err != nil {
		log.Printf("failed to JSON unmarshal authPath %q content %q: %s", authPath, authContent, err.Error())
		return
	}

	if tc.auth.ClientId == 0 {
		err = fmt.Errorf("invalid authPath %q content %q: invalid client id", authPath, authContent)
		log.Print(err.Error())
		return
	}

	if tc.auth.Token == "" {
		err = fmt.Errorf("invalid authPath %q content %q: empty token value", authPath, authContent)
		log.Print(err.Error())
		return
	}

	return
}

func (tc *TelemetryClient) saveTelemetryAuth() (err error) {
	authPath := tc.AuthPath()

	taJSON, err := json.Marshal(&tc.auth)
	if err != nil {
		log.Printf("failed to JSON marshal TelemetryAuth: %s", err.Error())
		return
	}

	err = os.WriteFile(authPath, taJSON, 0600)
	if err != nil {
		log.Printf("failed to write JSON marshalled TelemetryAuth to %q: %s", authPath, err.Error())
	}

	return
}

func (tc *TelemetryClient) submitReport(report *telemetrylib.TelemetryReport) (err error) {
	// submit a telemetry report
	var trReq restapi.TelemetryReportRequest
	trReq.TelemetryReport = *report
	reqBodyJSON, err := json.Marshal(&trReq)
	if err != nil {
		log.Printf("failed to JSON marshal trReq: %s", err.Error())
		return
	}

	reqUrl := tc.cfg.TelemetryBaseURL + "/report"
	reqBuf := bytes.NewBuffer(reqBodyJSON)
	req, err := http.NewRequest("POST", reqUrl, reqBuf)
	if err != nil {
		log.Printf("failed to create new HTTP request for telemetry report: %s", err.Error())
		return
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("X-AuthToken", string(tc.auth.Token))

	httpClient := http.DefaultClient
	resp, err := httpClient.Do(req)
	if err != nil {
		log.Printf("failed to HTTP POST telemetry report request: %s", err.Error())
		return
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("failed to read telemetry report response body: %s", err.Error())
		return
	}

	if resp.StatusCode != http.StatusOK {
		err = fmt.Errorf("telemetry report failed: %s", string(respBody))
		return
	}

	var trResp restapi.TelemetryReportResponse
	err = json.Unmarshal(respBody, &trResp)
	if err != nil {
		log.Printf("failed to JSON unmarshal telemetry report response body content: %s", err.Error())
		return
	}

	log.Printf("Successfully submitted report %q: processing %q", report.Header.ReportId, trResp.ProcessingInfo())

	return
}

func (tc *TelemetryClient) Register() (err error) {
	// get the saved TelemetryAuth, returning success if found
	err = tc.loadTelemetryAuth()
	if err == nil {
		log.Printf("telemtry auth found, client already registered as id %d, skipping", tc.auth.ClientId)
		return
	}

	// get the instanceId, failing if it can't be retrieved
	instId, err := tc.getInstanceId()
	if err != nil {
		return
	}

	// register the system as a client
	var crReq restapi.ClientRegistrationRequest
	crReq.ClientInstanceId = string(instId)
	reqBodyJSON, err := json.Marshal(&crReq)
	if err != nil {
		log.Printf("failed to JSON marshal crReq: %s", err.Error())
		return
	}

	reqUrl := tc.cfg.TelemetryBaseURL + "/register"
	reqBuf := bytes.NewBuffer(reqBodyJSON)
	req, err := http.NewRequest("POST", reqUrl, reqBuf)
	if err != nil {
		log.Printf("failed to create new HTTP request for client registration: %s", err.Error())
		return
	}

	req.Header.Add("Content-Type", "application/json")

	httpClient := http.DefaultClient
	resp, err := httpClient.Do(req)
	if err != nil {
		log.Printf("failed to HTTP POST client registration request: %s", err.Error())
		return
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("failed to read client registration response body: %s", err.Error())
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
		log.Printf("failed to JSON unmarshal client registration response body content: %s", err.Error())
		return
	}

	tc.auth.ClientId = crResp.ClientId
	tc.auth.Token = types.TelemetryAuthToken(crResp.AuthToken)
	tc.auth.IssueDate, err = types.TimeStampFromString(crResp.IssueDate)
	if err != nil {
		log.Printf("failed to parse %q as a timestamp: %s", crResp.IssueDate, err.Error())
		return
	}

	err = tc.saveTelemetryAuth()
	if err != nil {
		log.Printf("failed to save TelemetryAuth: %s", err.Error())
		return
	}

	log.Printf("successfully registered as client with id %d", tc.auth.ClientId)

	return nil
}

func (tc *TelemetryClient) Generate(telemetry types.TelemetryType, content []byte, tags types.Tags) error {
	// Add telemetry data item to DataItem data store
	log.Printf("Generated Telemetry: Name: %q, Tags: %v, Content: %s\n",
		telemetry, tags, content)
	tc.processor.AddData(telemetry, content, tags)

	return nil
}

func (tc *TelemetryClient) CreateBundles(tags types.Tags) error {
	// Bundle existing telemetry data items found in DataItem data store into one or more bundles in the Bundle data store
	log.Printf("Bundle: Tags: %v", tags)
	tc.processor.GenerateBundle(tc.auth.ClientId, tc.cfg.CustomerID, tags)

	return nil
}

func (tc *TelemetryClient) CreateReports(tags types.Tags) (err error) {
	// Generate reports from available bundles
	log.Printf("CreateReports: Tags: %v", tags)
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
