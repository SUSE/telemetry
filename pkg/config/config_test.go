package config

import (
	"fmt"
	"os"
	os_user "os/user"
	"path/filepath"
	"strings"
	"testing"

	"github.com/SUSE/telemetry/pkg/types"
	"github.com/stretchr/testify/suite"
)

type TestConfigTestSuite struct {
	suite.Suite

	tmpDir string
}

func (t *TestConfigTestSuite) SetupTest() {
	tmpDir, err := os.MkdirTemp("", ".cfgTest.*")
	t.Require().NoError(err, "os.MkdirTemp()")
	t.Require().NotEmpty(tmpDir, "tmpDir should be setup")

	t.tmpDir = tmpDir
}

func (t *TestConfigTestSuite) TearDownTest() {
	err := os.RemoveAll(t.tmpDir)
	t.NoError(err, "os.RemoveAll(t.tmpDir)")
}

func (t *TestConfigTestSuite) createTemp(name string) (file *os.File, err error) {
	return os.Create(filepath.Join(t.tmpDir, name))
}

func (t *TestConfigTestSuite) TestConfigDefaults() {
	// get current user
	currUser, err := os_user.Current()
	t.Require().NoError(err, "os/user.Current()")
	t.Require().NotNil(currUser, "currUser")

	// get current user's primary group
	currGroup, err := os_user.LookupGroupId(currUser.Gid)
	t.Require().NoError(err, "os/user.LookupGroupId(currUser.Gid)")
	t.Require().NotNil(currGroup, "currGroup")

	// create an empty file
	cfgName := "empty.yaml"
	emptyFile, err := t.createTemp(cfgName)
	t.Require().NoError(err, "creating empty file")
	cfgPath := emptyFile.Name()

	err = emptyFile.Close()
	t.Require().NoError(err, "closing created empty file")

	cfg, err := NewConfig(cfgPath)
	t.Require().NoError(err, "loading empty config")

	t.Equal(t.tmpDir, cfg.ConfigDir(), "Expected config dir path")
	t.Equal(cfgName, cfg.ConfigName(), "Expected config file name")
	t.Equal(cfgPath, cfg.ConfigPath(), "Expected full config file path")
	t.Equal(currUser.Username, cfg.ConfigUser(), "Expected current user")
	t.Equal(currGroup.Name, cfg.ConfigGroup(), "Expected current user's primary group")

	t.Equal(DEF_CFG_BASE_URL, cfg.TelemetryBaseURL, "TelemetryBaseURL value is not expected value")
	t.Equal(DEF_CFG_ENABLED, cfg.Enabled, "Enabled value is not expected value")
	t.Equal(DEF_CFG_CLIENT_ID, cfg.ClientId, "ClientId value is not expected value")
	t.Equal(DEF_CFG_CUSTOMER_ID, cfg.CustomerId, "CustomerId value is not expected value")

	t.Equal(DEF_CFG_DB_DRIVER, cfg.DataStores.Driver, "DataStores.Driver is not expected value")
	t.Equal(DEF_CFG_DB_PATH, cfg.DataStores.Params, "DataStores.Params is not expected value")

	t.Equal(DEF_CFG_LOG_LEVEL, cfg.Logging.Level, "Logging.Level is not expected value")
	t.Equal(DEF_CFG_LOG_LOCATION, cfg.Logging.Location, "Logging.Location is not expected value")
	t.Equal(DEF_CFG_LOG_STYLE, cfg.Logging.Style, "Logging.Style is not expected value")

	t.Equal(DEF_CFG_OPT_OUT, cfg.ClassOptions.OptOut, "ClassOptions.OptOut is not expected value")
	t.Equal(DEF_CFG_OPT_IN, cfg.ClassOptions.OptIn, "ClassOptions.OptIn is not expected value")
	t.Empty(cfg.Tags, "Tags value is expected to be empty")
	t.Empty(cfg.ClassOptions.Allow, "ClassOptions.Allow is expected to be empty")
	t.Empty(cfg.ClassOptions.Deny, "ClassOptions.Deny is expected to be empty")

	t.NotEmpty(cfg.String(), "string representation of config should be non-empty")
	t.NotEmpty(cfg.ClassOptions.String(), "string representation of class options config should be non-empty")
	t.NotEmpty(cfg.DataStores.String(), "string representation of data stores config should be non-empty")
	t.NotEmpty(cfg.Logging.String(), "string representation of logging config should be non-empty")
}

func (t *TestConfigTestSuite) TestConfigLoadSaveUpdate() {
	emptyFile, err := t.createTemp("config.yaml")
	t.Require().NoError(err, "creating empty file")
	cfgPath := emptyFile.Name()

	err = emptyFile.Close()
	t.Require().NoError(err, "closing created empty file")

	defCfg, err := NewConfig(cfgPath)
	t.Require().NoError(err, "loading empty config")

	err = defCfg.Save()
	t.Require().NoError(err, "saving default config")

	loadCfg, err := NewConfig(cfgPath)
	t.Require().NoError(err, "loading default config")

	t.Equal(defCfg.String(), loadCfg.String(), "unmodified config should match default")

	// toggle the enabled setting
	loadCfg.Enabled = !loadCfg.Enabled

	t.NotEqual(defCfg.String(), loadCfg.String(), "updated config shouldn't match default")

	err = loadCfg.Save()
	t.Require().NoError(err, "saving updated config")

	updatedCfg, err := NewConfig(cfgPath)
	t.Require().NoError(err, "loading updated config")

	t.Equal(loadCfg.Enabled, updatedCfg.Enabled, "enabled setting should match in updated and reloaded configs")
	t.NotEqual(defCfg.String(), updatedCfg.String(), "reloaded config shouldn't match default")
	t.Equal(loadCfg.String(), updatedCfg.String(), "reloaded config should match updated")
}

func (t *TestConfigTestSuite) TestConfigFileDoesntExist() {
	// skip the test if the default config user exists
	user, err := os_user.Lookup(DEF_CFG_USER)
	if err == nil {
		t.T().Skipf("Skipping test as user %q exists with id %v", user.Username, user.Uid)
	}

	// skip the test if the default config user exists
	group, err := os_user.LookupGroup(DEF_CFG_GROUP)
	if err == nil {
		t.T().Skipf("Skipping test as group %q exists with id %v", group.Name, group.Gid)
	}

	cfgPath := filepath.Join(t.tmpDir, "config.yaml")

	_, err = NewConfig(cfgPath)
	t.Error(err, "loading non-existent file should fail")
}

func (t *TestConfigTestSuite) TestConfigFileFound() {
	tmpfile, err := t.createTemp("config.yaml")
	t.Require().NoError(err)
	defer os.Remove(tmpfile.Name())

	url := "http://localhost:9999/telemetry"
	driver := "sqlite3"
	params := "/tmp/telemetry/testcfg/telemetry.db"

	content := `
telemetry_base_url: %s
enabled: true
customer_id: SOME_CUSTOMER
tags: []
datastores:
  driver: %s
  params: %s
logging:
  level: info
  location: stderr
  style: text
class_options:
  opt_out: true
  opt_in: false
  allow: []
  deny: []
`

	formattedContents := fmt.Sprintf(content, url, driver, params)
	_, err = tmpfile.Write([]byte(formattedContents))
	t.Require().NoError(err)
	t.Require().NoError(tmpfile.Close())

	config, err := NewConfig(tmpfile.Name())
	t.Require().NoError(err)

	t.Equal(url, config.TelemetryBaseURL, "TelemetryBaseURL value is not the expected")
	t.Equal(driver, config.DataStores.Driver, "DataStores.Driver is not the expected")
	t.Equal(params, config.DataStores.Params, "DataStores.Params is not the expected")
}

func (t *TestConfigTestSuite) TestConfigFileFoundButUnparsable() {
	tmpfile, err := t.createTemp("config.yaml")
	t.Require().NoError(err)
	defer os.Remove(tmpfile.Name())

	url := "http://localhost:9999/telemetry"
	driver := "sqlite3"
	params := "/tmp/telemetry/testcfg/telemetry.db"

	content := `
telemetry_base_url: %s
enabled true
customer_id: SOME_CUSTOMER
tags: []
datastores:
  driver: %s
  params: %s
logging:
  level: info
  location: stderr
  style: text
class_options:
  opt_out: true
  opt_in: false
  allow: []
  deny: []
`

	formattedContents := fmt.Sprintf(content, url, driver, params)
	_, err = tmpfile.Write([]byte(formattedContents))
	t.Require().NoError(err)
	t.Require().NoError(tmpfile.Close())

	_, err = NewConfig(tmpfile.Name())
	if !strings.Contains(err.Error(), "failed to yaml.Unmarshal() contents of config") {
		t.T().Errorf("String '%s' does not contain substring '%s'", err.Error(), "failed to parse contents of config file")
	}

}

func (t *TestConfigTestSuite) TestConfigTelemetryClasses() {
	cfgFile, err := t.createTemp("config.yaml")
	t.Require().NoError(err, "creating config file")
	cfgPath := cfgFile.Name()

	err = cfgFile.Close()
	t.Require().NoError(err, "closing created config file")

	cfg, err := NewConfig(cfgPath)
	t.Require().NoError(err, "loading config")

	// ensure telemetry collection is enabled
	cfg.Enabled = true

	// enable opt-in and opt out telemetry
	cfg.ClassOptions.OptIn = true
	cfg.ClassOptions.OptOut = true

	// all should be enabled
	t.True(cfg.TelemetryClassEnabled(types.MANDATORY_TELEMETRY), "mandatory always enabled")
	t.True(cfg.TelemetryClassEnabled(types.OPT_IN_TELEMETRY), "opt-in should be enabled")
	t.True(cfg.TelemetryClassEnabled(types.OPT_OUT_TELEMETRY), "opt-out should be enabled")

	// disable opt-in and opt out telemetry
	cfg.ClassOptions.OptIn = false
	cfg.ClassOptions.OptOut = false

	// only mandatory should be enabled
	t.True(cfg.TelemetryClassEnabled(types.MANDATORY_TELEMETRY), "mandatory always enabled")
	t.False(cfg.TelemetryClassEnabled(types.OPT_IN_TELEMETRY), "opt-in should be disabled")
	t.False(cfg.TelemetryClassEnabled(types.OPT_OUT_TELEMETRY), "opt-out should be disabled")

	// unrecognised class should be disabled
	t.False(cfg.TelemetryClassEnabled(999999999), "unrecognised class should be disabled")

	// disable telemetry collection
	cfg.Enabled = false

	// all should be disabled
	t.False(cfg.TelemetryClassEnabled(types.MANDATORY_TELEMETRY), "telemetry gathering disabled")
	t.False(cfg.TelemetryClassEnabled(types.OPT_IN_TELEMETRY), "telemetry gathering disabled")
	t.False(cfg.TelemetryClassEnabled(types.OPT_OUT_TELEMETRY), "telemetry gathering disabled")
}

func (t *TestConfigTestSuite) TestConfigTelemetryTypes() {
	cfgFile, err := t.createTemp("config.yaml")
	t.Require().NoError(err, "creating config file")
	cfgPath := cfgFile.Name()

	err = cfgFile.Close()
	t.Require().NoError(err, "closing created config file")

	cfg, err := NewConfig(cfgPath)
	t.Require().NoError(err, "loading config")

	// ensure telemetry collection is enabled
	cfg.Enabled = true

	// confirm any telemetry permitted when allow and deny lists are empty
	t.True(cfg.TelemetryTypeEnabled("TYPE1"), "allowed if lists are empty")
	t.True(cfg.TelemetryTypeEnabled("TYPE2"), "allowed if lists are empty")

	// add test type to the allow list
	cfg.ClassOptions.Allow = append(cfg.ClassOptions.Allow, "TYPE1")

	// only allowed telemetry types should be permitted
	t.True(cfg.TelemetryTypeEnabled("TYPE1"), "allowed telemetry should be enabled")
	t.False(cfg.TelemetryTypeEnabled("TYPE2"), "only allowed telemetry should be enabled")

	// add test type to the deny list
	cfg.ClassOptions.Deny = append(cfg.ClassOptions.Deny, "TYPE1")

	// deny list overrides allow list
	t.False(cfg.TelemetryTypeEnabled("TYPE1"), "deny list overrides allow list")
	t.False(cfg.TelemetryTypeEnabled("TYPE2"), "if not denied still needs to be allowed")

	// tests should be enabled if in allow list but not in deny list
	cfg.ClassOptions.Allow = append(cfg.ClassOptions.Allow, "TYPE2")
	t.True(cfg.TelemetryTypeEnabled("TYPE2"), "allowed but not denied telemetry should be enabled")

	// clear allow list so only deny list active
	cfg.ClassOptions.Allow = []types.TelemetryType{}

	// tests should be enabled so long as not in deny list if allow list is empty
	t.False(cfg.TelemetryTypeEnabled("TYPE1"), "denied if in deny list")
	t.True(cfg.TelemetryTypeEnabled("TYPE2"), "allowed if not in deny list")

	// confirm tests still enabled if both lists are empty
	cfg.ClassOptions.Deny = []types.TelemetryType{}
	t.True(cfg.TelemetryTypeEnabled("TYPE1"), "allowed if lists are empty")
	t.True(cfg.TelemetryTypeEnabled("TYPE2"), "allowed if lists are empty")

	// disable telemetry collection
	cfg.Enabled = false

	// tests should be disabled if collection disabled
	t.False(cfg.TelemetryTypeEnabled("TYPE1"), "denied if collection disabled")
	t.False(cfg.TelemetryTypeEnabled("TYPE2"), "denied if collection disabled")
}

func TestTelemetryClientConfigTestSuite(t *testing.T) {
	suite.Run(t, new(TestConfigTestSuite))
}
