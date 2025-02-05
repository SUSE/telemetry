package client

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/SUSE/telemetry/pkg/config"
	telemetrylib "github.com/SUSE/telemetry/pkg/lib"
	"github.com/SUSE/telemetry/pkg/types"
	"github.com/golang-jwt/jwt/v5"
)

const (
	//CONFIG_DIR  = "/etc/susetelemetry"
	CONFIG_DIR  = "/tmp/susetelemetry"
	CONFIG_PATH = CONFIG_DIR + "/config.yaml"
	AUTH_PATH   = CONFIG_DIR + "/auth.json"
	//INSTANCEID_PATH = CONFIG_DIR + "/instanceid"
)

// TODO: unify with restapi.ClientRegistrationResponse
type TelemetryAuth struct {
	// TODO: unify these fields with restapi.ClientRegistrationResponse
	RegistrationId   int64                    `json:"registrationId"`
	Token            types.TelemetryAuthToken `json:"authToken"`
	RegistrationDate types.TelemetryTimeStamp `json:"registrationDate"`
}

type TelemetryClient struct {
	cfg        *config.Config
	reg        *TelemetryClientRegistration
	auth       TelemetryAuth
	authLoaded bool
	processor  telemetrylib.TelemetryProcessor
}

func NewTelemetryClient(cfg *config.Config) (tc *TelemetryClient, err error) {
	tc = &TelemetryClient{cfg: cfg}
	tc.reg = NewTelemetryClientRegistration()
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

func (tc *TelemetryClient) ensureRegistrationExists() (err error) {

	// if the client registration is valid and accessible then nothing to do
	if tc.reg.Valid() && tc.reg.Accessible() {
		slog.Debug(
			"client registration exists",
			slog.String("reg", tc.reg.String()),
		)
		return nil
	}

	slog.Debug(
		"ensuring existence of client registration",
		slog.String("regPath", tc.reg.path),
	)

	// generate a new client registration if needed
	if !tc.reg.Valid() {
		tc.reg.Generate()
		slog.Debug(
			"client registration generated",
			slog.String("reg", tc.reg.String()),
		)
	}

	err = tc.reg.Save()
	if err != nil {
		slog.Debug(
			"failed to save client registration generated",
			slog.String("reg", tc.reg.String()),
		)
		return err
	}

	slog.Debug(
		"saved client registration",
		slog.String("reg", tc.reg.String()),
	)

	return
}

func (tc *TelemetryClient) authParsedToken() (token *jwt.Token, err error) {
	if !tc.authLoaded {
		if err = tc.loadTelemetryAuth(); err != nil {
			slog.Error(
				"Failed to load authToken",
				slog.String("error", err.Error()),
			)
			return
		}
	}

	// only the server can validate the signing key, so parse unverified
	token, _, err = jwt.NewParser().ParseUnverified(
		string(tc.auth.Token), jwt.MapClaims{},
	)

	if err != nil {
		slog.Error(
			"Failed to parse JWT",
			slog.String("error", err.Error()),
		)
		return
	}

	return
}

func (tc *TelemetryClient) AuthIssuer() (issuer string, err error) {
	token, err := tc.authParsedToken()
	if err != nil {
		return
	}

	issuer, err = token.Claims.GetIssuer()
	if err != nil {
		slog.Error(
			"Filed to retrieve issuer from token claims",
			slog.String("err", err.Error()),
		)
		return
	}

	return
}

func (tc *TelemetryClient) AuthExpiration() (expiration time.Time, err error) {
	token, err := tc.authParsedToken()
	if err != nil {
		return
	}

	expTime, err := token.Claims.GetExpirationTime()
	if err != nil {
		slog.Error(
			"Filed to retrieve expiration date from token claims",
			slog.String("err", err.Error()),
		)
		return
	}

	expiration = expTime.Time

	return
}

func (tc *TelemetryClient) AuthAccessible() bool {
	return checkFileReadAccessible(tc.AuthPath())
}

func (tc *TelemetryClient) HasAuth() bool {
	return checkFileExists(tc.AuthPath())
}

func (tc *TelemetryClient) HasInstanceId() bool {
	return checkFileExists(tc.RegistrationPath())
}

func (tc *TelemetryClient) Processor() telemetrylib.TelemetryProcessor {
	// may want to just make the processor a public field
	return tc.processor
}

func (tc *TelemetryClient) AuthPath() string {
	// hard coded for now, possibly make a config option
	return AUTH_PATH
}

func (tc *TelemetryClient) ClientId() string {
	return tc.reg.ClientId
}

func (tc *TelemetryClient) RegistrationPath() string {
	return tc.reg.path
}

func (tc *TelemetryClient) RegistrationAccessible() bool {
	return tc.reg.Accessible()
}

func (tc *TelemetryClient) getRegistration() (reg types.ClientRegistration, err error) {

	err = tc.ensureRegistrationExists()
	if err != nil {
		return
	}

	err = tc.reg.Load()
	if err != nil {
		slog.Debug(
			"failed to load client registration",
			slog.String("reg", tc.reg.Path()),
			slog.String("err", err.Error()),
		)
		return
	}

	slog.Debug(
		"successfully loaded client registration",
		slog.String("reg", tc.reg.String()),
	)

	reg = tc.reg.ClientRegistration

	return
}

func (tc *TelemetryClient) deleteTelemetryAuth() (err error) {
	if err = os.Remove(tc.AuthPath()); err != nil {
		slog.Error(
			"Failed to delete existing client creds",
			slog.String("error", err.Error()),
		)
		return
	}

	// clear previous in memory auth settings
	tc.auth = TelemetryAuth{}
	tc.authLoaded = false

	return
}

func (tc *TelemetryClient) loadTelemetryAuth() (err error) {
	authPath := tc.AuthPath()

	slog.Debug("Checking auth file existence", slog.String("authPath", authPath))
	_, err = os.Stat(authPath)
	if os.IsNotExist(err) {
		slog.Debug(
			"Unable to find auth file",
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

	if tc.auth.RegistrationId <= 0 {
		err = fmt.Errorf("invalid registration id")
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

	tc.authLoaded = true

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

func errClientNotAuthorized() error {
	return errors.New("client not authorized")
}

func errRegistrationRequired() error {
	return errors.New("client registration required")
}

func errAuthenticationRequired() error {
	return errors.New("client authentication required")
}

var (
	ErrClientNotAuthorized    = errClientNotAuthorized()    // general authorization failure
	ErrRegistrationRequired   = errRegistrationRequired()   // need to (re-)register
	ErrAuthenticationRequired = errAuthenticationRequired() // need to (re-authenticate)
)

func parseQuotedAssignment(assignment string) (field, value string, found bool) {
	// split assignment on '='
	field, value, found = strings.Cut(assignment, "=")
	if found {
		// if split was successful string quote and inner wrapping spaces
		value = strings.TrimSpace(strings.Trim(value, `"'`))
	}
	return
}

func unauthorizedError(resp *http.Response) (err error) {
	// default to general authorization failure
	err = ErrClientNotAuthorized

	// retrieve the WWW-Authenticate response header
	hdrWwwAuthenticate, found := resp.Header[http.CanonicalHeaderKey("WWW-Authenticate")]
	if !found {
		slog.Error(
			"Unauthorized response lacks WWW-Authenticate header",
			slog.Int("StatusCode", resp.StatusCode),
		)
		return
	}

	// joing possible multiple header values with ","
	wwwAuthenticate := strings.Join(hdrWwwAuthenticate, ",")

	if wwwAuthenticate == "" {
		slog.Error(
			"Unauthorized response WWW-Authenticate header empty",
			slog.Int("StatusCode", resp.StatusCode),
			slog.String("WWW-Authenticate", wwwAuthenticate),
		)
		return
	}

	// the WWW-Authenticate header should have the following format
	//   <challenge> realm="<realm>" scope="<scope>"
	// where:
	//   <challenge> is "Bearer"
	//   <realm> is "suse-telemetry-service"
	//   <scope> is either "authenticate" or "register"
	fields := strings.Fields(wwwAuthenticate)

	// validate the WWW-Authenticate header value
	if len(fields) < 3 {
		slog.Error(
			"Unauthorized response WWW-Authenticate header invalid format",
			slog.Int("StatusCode", resp.StatusCode),
			slog.String("WWW-Authenticate", wwwAuthenticate),
		)
		return
	}

	// first field specifies the challenge, value validated below
	challenge := fields[0]

	// second field should be realm="<realm>", value validated below
	fieldName, realm, found := parseQuotedAssignment(fields[1])
	if !found || (fieldName != "realm") {
		slog.Error(
			"Unauthorized response WWW-Authenticate header missing realm",
			slog.Int("StatusCode", resp.StatusCode),
			slog.String("WWW-Authenticate", wwwAuthenticate),
		)
		return
	}

	// third field should be scope="<scope>", value validated below
	fieldName, scope, found := parseQuotedAssignment(fields[2])
	if !found || (fieldName != "scope") {
		slog.Error(
			"Unauthorized response WWW-Authenticate header missing scope",
			slog.Int("StatusCode", resp.StatusCode),
			slog.String("WWW-Authenticate", wwwAuthenticate),
		)
		return
	}

	// only Bearer challenge type is accepted
	if challenge != "Bearer" {
		slog.Error(
			"Unauthorized response WWW-Authenticate header invalid challenge",
			slog.Int("StatusCode", resp.StatusCode),
			slog.String("WWW-Authenticate", wwwAuthenticate),
			slog.String("challenge", challenge),
		)
		return
	}

	// only suse-telemetry-service realm type is accepted
	switch realm {
	case "suse-telemetry-service":
		// valid
	default:
		slog.Error(
			"Unauthorized response WWW-Authenticate header invalid realm",
			slog.Int("StatusCode", resp.StatusCode),
			slog.String("WWW-Authenticate", wwwAuthenticate),
			slog.String("realm", realm),
		)
		return
	}

	// only authenticate and register scope types are accepted
	switch scope {
	case "authenticate":
		slog.Debug("Client (re-)authentication required")
		err = ErrAuthenticationRequired
	case "register":
		slog.Debug("Client (re-)registration required")
		err = ErrRegistrationRequired
	default:
		slog.Error(
			"Unauthorized response WWW-Authenticate header invalid scope",
			slog.Int("StatusCode", resp.StatusCode),
			slog.String("WWW-Authenticate", wwwAuthenticate),
			slog.String("scope", scope),
		)
	}

	return
}

func (tc *TelemetryClient) Generate(telemetry types.TelemetryType, content *types.TelemetryBlob, tags types.Tags) error {
	// Enforce valid versioned JSON object
	if err := content.Valid(); err != nil {
		slog.Debug(
			"Supplied content is not a versioned JSON object",
			slog.String("error", err.Error()),
		)
		return err
	}

	// Enforce content size limits
	if err := content.CheckLimits(); err != nil {
		slog.Debug(
			"Supplied JSON blob failed limits check",
			slog.String("error", err.Error()),
		)
		return err
	}

	// Add telemetry data item to DataItem data store
	slog.Debug(
		"Generated Telemetry",
		slog.String("name", telemetry.String()),
		slog.String("tags", tags.String()),
		slog.String("content", content.String()),
	)

	return tc.processor.AddData(telemetry, content, tags)
}

func (tc *TelemetryClient) CreateBundles(tags types.Tags) error {
	// Bundle existing telemetry data items found in DataItem data store into one or more bundles in the Bundle data store
	slog.Debug("Bundle", slog.String("Tags", tags.String()))
	tc.processor.GenerateBundle(tc.ClientId(), tc.cfg.CustomerID, tags)

	return nil
}

func (tc *TelemetryClient) CreateReports(tags types.Tags) (err error) {
	// Generate reports from available bundles
	slog.Debug("CreateReports", slog.String("Tags", tags.String()))
	tc.processor.GenerateReport(tc.ClientId(), tags)

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
			return fmt.Errorf("failed to convert report %q: %w", reportRow.ReportId, err)
		}

		if err := tc.submitReport(&report); err != nil {
			return fmt.Errorf("failed to submit report %q: %w", report.Header.ReportId, err)
		}

		// delete the successfully submitted report
		tc.processor.DeleteReport(reportRow)
	}

	return nil
}
