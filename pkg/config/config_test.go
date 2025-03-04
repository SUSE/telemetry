package config

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/SUSE/telemetry/pkg/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type TestConfigTestSuite struct {
	suite.Suite
}

func (t *TestConfigTestSuite) TestConfigFileNotFound() {
	config, err := NewConfig("nonexisting.yaml")
	assert.NoError(t.T(), err, "error using the defaults for non existent config file")

	assert.Equal(t.T(), DefaultCfg.TelemetryBaseURL, config.TelemetryBaseURL, "TelemetryBaseURL value is not the expected")
	assert.Equal(t.T(), DefaultCfg.Enabled, config.Enabled, "Enabled value is not the expected")
	assert.Equal(t.T(), DefaultCfg.ClientId, config.ClientId, "ClientId value is not the expected")
	assert.Equal(t.T(), DefaultCfg.CustomerId, config.CustomerId, "CustomerId value is not the expected")
	assert.Equal(t.T(), DefaultCfg.Tags, config.Tags, "Tags value is not the expected")
	assert.Equal(t.T(), DefaultCfg.DataStores.Driver, config.DataStores.Driver, "DataStores.Driver is not the expected")
	assert.Equal(t.T(), DefaultCfg.DataStores.Params, config.DataStores.Params, "DataStores.Params is not the expected")
	assert.Equal(t.T(), DefaultCfg.Logging.Level, config.Logging.Level, "Logging.Level is not the expected")
	assert.Equal(t.T(), DefaultCfg.Logging.Location, config.Logging.Location, "Logging.Location is not the expected")
	assert.Equal(t.T(), DefaultCfg.Logging.Style, config.Logging.Style, "Logging.Style is not the expected")
	assert.Equal(t.T(), DefaultCfg.ClassOptions.OptOut, config.ClassOptions.OptOut, "ClassOptions.OptOut is not the expected")
	assert.Equal(t.T(), DefaultCfg.ClassOptions.OptIn, config.ClassOptions.OptIn, "ClassOptions.OptIn is not the expected")
	assert.Equal(t.T(), DefaultCfg.ClassOptions.Allow, config.ClassOptions.Allow, "ClassOptions.Allow is not the expected")
	assert.Equal(t.T(), DefaultCfg.ClassOptions.Deny, config.ClassOptions.Deny, "ClassOptions.Deny is not the expected")
}

func (t *TestConfigTestSuite) TestConfigFileFound() {
	tmpfile, err := os.CreateTemp("/tmp", "temp-cfg"+utils.GenerateRandomString(5)+".yaml")
	require.NoError(t.T(), err)
	defer os.Remove(tmpfile.Name())

	url := "http://localhost:9999/telemetry"
	driver := "sqlite3"
	params := "/tmp/telemetry/testcfg/telemetry.db"

	content := `
telemetry_base_url: %s
enabled: true
customer_id: 01234
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
	require.NoError(t.T(), err)
	require.NoError(t.T(), tmpfile.Close())

	config, err := NewConfig(tmpfile.Name())
	require.NoError(t.T(), err)

	assert.Equal(t.T(), url, config.TelemetryBaseURL, "TelemetryBaseURL value is not the expected")
	assert.Equal(t.T(), driver, config.DataStores.Driver, "DataStores.Driver is not the expected")
	assert.Equal(t.T(), params, config.DataStores.Params, "DataStores.Params is not the expected")
}

func (t *TestConfigTestSuite) TestConfigFileFoundButUnparsable() {
	tmpfile, err := os.CreateTemp("/tmp", "temp-cfg"+utils.GenerateRandomString(5)+".yaml")
	require.NoError(t.T(), err)
	defer os.Remove(tmpfile.Name())

	url := "http://localhost:9999/telemetry"
	driver := "sqlite3"
	params := "/tmp/telemetry/testcfg/telemetry.db"

	content := `
telemetry_base_url: %s
enabled true
customer_id: 01234
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
	require.NoError(t.T(), err)
	require.NoError(t.T(), tmpfile.Close())

	_, err = NewConfig(tmpfile.Name())
	if !strings.Contains(err.Error(), "failed to parse contents of config file") {
		t.T().Errorf("String '%s' does not contain substring '%s'", err.Error(), "failed to parse contents of config file")
	}

}

func TestTelemetryClientConfigTestSuite(t *testing.T) {
	suite.Run(t, new(TestConfigTestSuite))
}
