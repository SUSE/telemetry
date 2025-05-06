package client

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/SUSE/telemetry/pkg/config"
	"github.com/SUSE/telemetry/pkg/restapi"
	"github.com/SUSE/telemetry/pkg/types"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/suite"
)

type ClientTestSuiteJWT struct {
	// JWT token generation settings
	Secret   []byte
	Method   jwt.SigningMethod
	Issuer   string
	Duration time.Duration

	// testing settings
	TokenAge time.Duration // how long ago was token generated
}

func (c *ClientTestSuiteJWT) newExpiration() *jwt.NumericDate {
	return jwt.NewNumericDate(time.Now().Add(c.Duration).Add(-c.TokenAge))
}

func (c *ClientTestSuiteJWT) CreateToken() (string, error) {
	return jwt.NewWithClaims(
		c.Method,
		jwt.MapClaims{
			"exp": c.newExpiration(),
			"iss": c.Issuer,
		},
	).SignedString(c.Secret)
}

type ClientTestSuite struct {
	suite.Suite

	tmpDir string

	// JWT token settings
	jwt ClientTestSuiteJWT

	// config will be setup once created
	cfg *config.Config

	// client will be setup once created
	client *TelemetryClient
}

func (t *ClientTestSuite) SetupTest() {
	// create a test specific temporary directory
	tmpDir, err := os.MkdirTemp("", ".clntTest.*")
	t.Require().NoError(err, "os.MkdirTemp()")
	t.Require().NotEmpty(tmpDir, "tmpDir should be setup")

	t.tmpDir = tmpDir

	// setup JWT token generation
	t.jwt = ClientTestSuiteJWT{
		Secret:   []byte("test"),
		Method:   jwt.SigningMethodHS512,
		Issuer:   "telemetry-client-test",
		Duration: time.Hour,
	}
}

func (t *ClientTestSuite) TearDownTest() {
	err := os.RemoveAll(t.tmpDir)
	t.NoError(err, "os.RemoveAll(t.tmpDir)")
}

func (t *ClientTestSuite) createTemp(name string) (file *os.File, err error) {
	return os.Create(filepath.Join(t.tmpDir, name))
}

func (t *ClientTestSuite) createTestConfig(server *httptest.Server) (string, error) {
	cfgFmt := `---
telemetry_base_url: %q
enabled: true
client_id: d19ecc03-787c-469b-8bf5-71df704f3b16
customer_id: TEST_CUSTOMER
tags: []
datastores:
  driver: sqlite3
  params: %s/client/telemetry.db
class_options:
  opt_out: true
  opt_in: false
  allow: []
  deny: []
logging:
  level: info
  location: stderr
  style: text`

	cfgFile, err := t.createTemp("config.yaml")
	t.Require().NoError(err, "should be able to create a temp config file")

	cfgContent := fmt.Sprintf(cfgFmt, server.URL, t.tmpDir)
	_, err = cfgFile.WriteString(cfgContent)
	t.Require().NoError(err, "should be able to write to temp config file")

	return cfgFile.Name(), err
}

func (t *ClientTestSuite) registerSucessHandler(w http.ResponseWriter, r *http.Request) {
	var err error
	var reqBytes, respBytes []byte
	var req restapi.ClientRegistrationRequest
	var authToken string
	var resp restapi.ClientRegistrationResponse

	// this test requires that t.cfg has been setup
	t.Require().NotNil(t.cfg, "test config should be setup")

	// verify that it was a /register POST request with JSON content
	t.Require().Equal("POST", r.Method, "should be a POST request")
	t.Require().Equal("/register", r.URL.Path, "should be a /register request")
	t.Require().Equal("application/json", r.Header.Get("Content-Type"))

	// extract the JSON payload
	reqBytes, err = io.ReadAll(r.Body)
	t.Require().NoError(err, "/register request should have a payload")
	t.Require().True(json.Valid(reqBytes))

	// unmarshal the JSON payload
	err = json.Unmarshal(reqBytes, &req)
	t.Require().NoError(err, "/register request payload should be a ClientRegistrationRequest")

	// validate the request payload
	err = req.Validate()
	t.Require().NoError(err, "/register payload should be a valid ClientRegistrationRequest")
	t.Require().Equal(t.cfg.ClientId, req.ClientRegistration.ClientId, "clientId should match what is specified in the config")

	// generate a valid auth token
	authToken, err = t.jwt.CreateToken()
	t.Require().NoError(err, "should be able to create an auth token")

	// create a mock response
	resp = restapi.ClientRegistrationResponse{
		RegistrationId:   1,
		AuthToken:        authToken,
		RegistrationDate: types.Now().String(),
	}
	err = resp.Validate()
	t.Require().NoError(err, "/register response payload should be a valid ClientRegistrationResponse")
	respBytes, err = json.Marshal(resp)
	t.Require().NoError(err, "/register response payload should be valid JSON")

	// sent the success (200) response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(respBytes)
}

func (t *ClientTestSuite) authenticateSucessHandler(w http.ResponseWriter, r *http.Request) {
	var err error
	var reqBytes, respBytes []byte
	var req restapi.ClientAuthenticationRequest
	var authToken string
	var resp restapi.ClientAuthenticationResponse

	// this test requires that t.cfg has been setup
	t.Require().NotNil(t.cfg, "test config should be setup")

	// this test requires that t.client has been setup
	t.Require().NotNil(t.client, "test client should be setup")

	// verify that it was a /authenticate POST request with JSON content
	t.Require().Equal("POST", r.Method, "should be a POST request")
	t.Require().Equal("/authenticate", r.URL.Path, "should be a /authenticate request")
	t.Require().Equal("application/json", r.Header.Get("Content-Type"))

	// extract the JSON payload
	reqBytes, err = io.ReadAll(r.Body)
	t.Require().NoError(err, "/authenticate request should have a payload")
	t.Require().True(json.Valid(reqBytes))

	// unmarshal the JSON payload
	err = json.Unmarshal(reqBytes, &req)
	t.Require().NoError(err, "/authenticate request payload should be a ClientAuthenticationRequest")

	// validate the request payload
	err = req.Validate()
	t.Require().NoError(err, "/authenticate payload should be a valid ClientAuthenticationRequest")
	t.Require().Equal(
		*t.client.reg.ClientRegistration.Hash("default"),
		req.RegHash,
		"/authenticate registration hash should match default hash",
	)

	// generate a valid auth token
	authToken, err = t.jwt.CreateToken()
	t.Require().NoError(err, "should be able to create an auth token")

	// create a mock response
	resp = restapi.ClientAuthenticationResponse{
		RegistrationId:   1,
		AuthToken:        authToken,
		RegistrationDate: types.Now().String(),
	}
	err = resp.Validate()
	t.Require().NoError(err, "/authenticate response payload should be a valid ClientAuthenticationResponse")
	respBytes, err = json.Marshal(resp)
	t.Require().NoError(err, "/authenticate response payload should be valid JSON")

	// sent the success (200) response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(respBytes)
}

func (t *ClientTestSuite) reportSucessHandler(w http.ResponseWriter, r *http.Request) {
	var err error
	var reqBytes, respBytes []byte
	var req restapi.TelemetryReportRequest
	var resp restapi.TelemetryReportResponse

	// this test requires that t.cfg has been setup
	t.Require().NotNil(t.cfg, "test config should be setup")

	// this test requires that t.client has been setup
	t.Require().NotNil(t.client, "test client should be setup")

	// verify that it was a /authenticate POST request with JSON content
	t.Require().Equal("POST", r.Method, "should be a POST request")
	t.Require().Equal("/report", r.URL.Path, "should be a /report request")
	t.Require().Equal("application/json", r.Header.Get("Content-Type"))

	// verify that the required headers are present
	t.Require().Equal(
		fmt.Sprintf("%d", t.client.creds.RegistrationId),
		r.Header.Get("X-Telemetry-Registration-Id"),
		"expected X-Telemetry-Registration-Id header to match client registration Id",
	)
	t.Require().Equal(
		`Bearer `+t.client.creds.AuthToken,
		r.Header.Get("Authorization"),
		"expected Authorization header to contain client auth token",
	)

	// extract the JSON payload
	reqBytes, err = io.ReadAll(r.Body)
	t.Require().NoError(err, "/report request should have a payload")
	t.Require().True(json.Valid(reqBytes))

	// unmarshal the JSON payload
	err = json.Unmarshal(reqBytes, &req)
	t.Require().NoError(err, "/report request payload should be a TelemetryReportRequest")

	// validate the request payload
	err = req.Validate()
	t.Require().NoError(err, "/report payload should be a valid TelemetryReportRequest")

	// create a mock response
	resp = restapi.TelemetryReportResponse{
		ProcessingId: 0,
		ProcessedAt:  types.Now(),
	}
	err = resp.Validate()
	t.Require().NoError(err, "/report response payload should be a valid TelemetryReportResponse")
	respBytes, err = json.Marshal(resp)
	t.Require().NoError(err, "/report response payload should be valid JSON")

	// sent the success (200) response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(respBytes)
}

type telemetryTestServerHandler struct {
	Method string
	Path   string
	Func   http.HandlerFunc
}

func (t *ClientTestSuite) telemetryTestServer(handlers ...telemetryTestServerHandler) (server *httptest.Server) {
	mux := http.NewServeMux()

	for _, handler := range handlers {
		pattern := fmt.Sprintf("%s %s", handler.Method, handler.Path)
		mux.HandleFunc(pattern, handler.Func)
	}

	server = httptest.NewServer(mux)

	return
}

func (t *ClientTestSuite) Test_RegisterAuthenticateSubmit() {
	var err error
	var cfgPath string

	// setup test server instance
	server := t.telemetryTestServer(
		telemetryTestServerHandler{
			Method: "POST",
			Path:   "/register",
			Func:   t.registerSucessHandler,
		},
		telemetryTestServerHandler{
			Method: "POST",
			Path:   "/authenticate",
			Func:   t.authenticateSucessHandler,
		},
		telemetryTestServerHandler{
			Method: "POST",
			Path:   "/report",
			Func:   t.reportSucessHandler,
		},
	)

	// generate a testing config file targetted at the test server
	cfgPath, err = t.createTestConfig(server)
	t.Require().NoError(err, "should have created config for test server")
	t.Require().FileExists(cfgPath, "test config file should exist")

	// load the generated testing config
	t.cfg, err = config.NewConfig(cfgPath)
	t.Require().NoError(err, "should be able to create test config object from test config file")
	t.Require().NotNil(t.cfg, "a valid config should have been returned")

	// create an initial client, register and re-authenticate it
	t.client, err = NewTelemetryClient(t.cfg)
	t.Require().NoError(err, "should be able to create test client object from test config object")

	// verify registration works
	err = t.client.Register()
	t.Require().NoError(err, "client registration should succeed")

	// verify re-authentication works
	err = t.client.Authenticate()
	t.Require().NoError(err, "client authentication should succeed")

	// create a second client instance, using previously created config,
	// and verify that it can re-authenticate as well
	t.cfg, err = config.NewConfig(cfgPath)
	t.Require().NoError(err, "should stll be able to create test config object from test config file")
	t.Require().NotNil(t.cfg, "a valid config should have been returned again")

	t.client, err = NewTelemetryClient(t.cfg)
	t.Require().NoError(err, "should still be able to create test client object from test config object")

	err = t.client.Authenticate()
	t.Require().NoError(err, "client authentication should succeed again")

	// generate a data item
	err = t.client.Generate(
		"TELEMETRY-UNIT-TEST",
		types.NewTelemetryBlob([]byte(`{"version":1,"data":{}}`)),
		types.Tags{},
	)
	t.Require().NoError(err, "data item generation should have worked")

	// create a bundle containing the generated data item
	err = t.client.CreateBundles(types.Tags{})
	t.Require().NoError(err, "bundle creation should have worked")

	// create a report containing the bundle containing the generated data item
	err = t.client.CreateReports(types.Tags{})
	t.Require().NoError(err, "report creation should have worked")

	// submit the report to the server
	err = t.client.Submit()
	t.Require().NoError(err, "report submission should have worked")
}

func TestTelemetryClientTestSuite(t *testing.T) {
	suite.Run(t, new(ClientTestSuite))
}
