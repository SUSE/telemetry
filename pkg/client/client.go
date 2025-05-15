package client

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/SUSE/telemetry/pkg/config"
	telemetrylib "github.com/SUSE/telemetry/pkg/lib"
	"github.com/SUSE/telemetry/pkg/types"
	"github.com/golang-jwt/jwt/v5"
)

type TelemetryClient struct {
	cfg       *config.Config
	reg       *TelemetryClientRegistration
	creds     *TelemetryClientCredentials
	processor telemetrylib.TelemetryProcessor
}

func NewTelemetryClient(cfg *config.Config) (tc *TelemetryClient, err error) {
	tc = &TelemetryClient{cfg: cfg}

	// create client registration manager
	tc.reg, err = NewTelemetryClientRegistration(cfg)
	if err != nil {
		slog.Debug(
			"failed to create a new client registration",
			slog.String("configDir", cfg.ConfigDir()),
			slog.String("err", err.Error()),
		)
		return nil, fmt.Errorf("failed to create a new client registration: %w", err)
	}

	// load the client registration if it exists
	if tc.reg.Exists() {
		if err = tc.reg.Load(); err != nil {
			slog.Warn(
				"failed to load existing client registration, regeneration required",
				slog.String("registration", tc.reg.Path()),
				slog.String("err", err.Error()),
			)
		}
	}

	// create client credentials manager
	tc.creds, err = NewTelemetryClientCredentials(cfg)
	if err != nil {
		slog.Debug(
			"failed to create a new client credentials",
			slog.String("configDir", cfg.ConfigDir()),
			slog.String("err", err.Error()),
		)
		return nil, fmt.Errorf("failed to create a new client credentials: %w", err)
	}

	// load the client credentials if it exists
	if tc.creds.Exists() {
		if err = tc.creds.Load(); err != nil {
			slog.Warn(
				"failed to load existing client credentials, registration required",
				slog.String("credentials", tc.creds.Path()),
				slog.String("err", err.Error()),
			)
		}
	}

	tc.processor, err = telemetrylib.NewTelemetryProcessor(&cfg.DataStores)
	if err != nil {
		slog.Debug(
			"failed to setup datastore management",
			slog.String("DataStores", cfg.DataStores.String()),
			slog.String("err", err.Error()),
		)
		return nil, fmt.Errorf("failed to setup data store manager: %w", err)
	}

	return tc, nil
}

func (tc *TelemetryClient) getRegistration() (reg types.ClientRegistration, err error) {
	// ensure that a registration exists, creating one if needed
	err = tc.ensureRegistrationExists()
	if err != nil {
		return
	}

	// if a registration is already loaded then nothing more to do
	if tc.reg.Valid() {
		reg = tc.reg.Registration()
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

func (tc *TelemetryClient) ensureRegistrationExists() (err error) {
	// if the existing client registration is valid then nothing to do
	if tc.reg.Valid() {
		slog.Debug(
			"client registration already loaded",
			slog.String("reg", tc.reg.String()),
		)
		return nil
	}

	slog.Debug(
		"ensuring existence of client registration",
		slog.String("path", tc.reg.Path()),
	)

	if tc.reg.Accessible() {
		slog.Debug(
			"client registration exists, needs to be loaded",
			slog.String("path", tc.reg.Path()),
		)
		return nil
	}

	// generate a new client registration if needed
	if err = tc.reg.Generate(); err != nil {
		slog.Error(
			"failed to generate a new registration",
			slog.String("path", tc.reg.Path()),
			slog.String("err", err.Error()),
		)
		return
	}

	slog.Debug(
		"client registration saved",
		slog.String("path", tc.reg.Path()),
		slog.String("reg", tc.reg.String()),
	)

	return
}

func (tc *TelemetryClient) authParsedToken() (token *jwt.Token, err error) {
	// load the client credentials
	if err = tc.creds.Load(); err != nil {
		slog.Error(
			"Failed to load credentials",
			slog.String("error", err.Error()),
		)
		return
	}

	// only the server can validate the signing key, so parse unverified
	token, _, err = jwt.NewParser().ParseUnverified(
		string(tc.creds.AuthToken), jwt.MapClaims{},
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

func (tc *TelemetryClient) ConfigTags() types.Tags {
	tags := make(types.Tags, len(tc.cfg.Tags))
	copy(tags, tc.cfg.Tags)
	return tags
}

func (tc *TelemetryClient) ServerURL() string {
	return tc.cfg.TelemetryBaseURL
}

func (tc *TelemetryClient) CredentialsAccessible() bool {
	return tc.creds.Accessible()
}

func (tc *TelemetryClient) HasCredentials() bool {
	return tc.creds.Exists()
}

func (tc *TelemetryClient) Processor() telemetrylib.TelemetryProcessor {
	// may want to just make the processor a public field
	return tc.processor
}

func (tc *TelemetryClient) CredentialsPath() string {
	return tc.creds.Path()
}

func (tc *TelemetryClient) ClientId() string {
	return tc.reg.ClientId
}

func (tc *TelemetryClient) RegistrationPath() string {
	return tc.reg.Path()
}

func (tc *TelemetryClient) RegistrationAccessible() bool {
	return tc.reg.Accessible()
}

func (tc *TelemetryClient) PersistentDatastore() bool {
	return tc.processor.Persistent()
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
	tc.processor.GenerateBundle(tc.ClientId(), tc.cfg.CustomerId, tags)

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
	err = tc.creds.Load()
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
			return fmt.Errorf("failed to generate report %q: %w", reportRow.ReportId, err)
		}

		if err := tc.submitReport(report); err != nil {
			return fmt.Errorf("failed to submit report %q: %w", report.Header.ReportId, err)
		}

		// delete the successfully submitted report
		tc.processor.DeleteReport(reportRow)
	}

	return nil
}
