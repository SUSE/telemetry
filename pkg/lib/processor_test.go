package telemetrylib

import (
	"fmt"
	"log"
	"strings"
	"testing"

	"github.com/SUSE/telemetry/pkg/config"
	"github.com/SUSE/telemetry/pkg/types"
	"github.com/SUSE/telemetry/pkg/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type telemetryProcessorTestEnv struct {
	cfgPath            string
	cfg                *config.Config
	telemetryprocessor TelemetryProcessor
}

func NewProcessorTestEnv(cfgFile string) (*telemetryProcessorTestEnv, error) {
	t := telemetryProcessorTestEnv{cfgPath: cfgFile}
	err := t.setup()
	return &t, err
}

func (t *telemetryProcessorTestEnv) setup() (err error) {
	t.cfg, err = config.NewConfig(t.cfgPath)
	if err != nil {
		log.Print(err.Error())
		return
	}
	processor, err := NewTelemetryProcessor(&t.cfg.DataStores)
	if err != nil {
		log.Printf("failed to setup telemetry processor for config %q: %s", t.cfgPath, err.Error())
		return
	}
	t.telemetryprocessor = processor
	return
}

func (t *telemetryProcessorTestEnv) cleanup() {

	if t.telemetryprocessor != nil {
		t.telemetryprocessor.cleanup()
	}
}

type TelemetryProcessorTestSuite struct {
	suite.Suite
	defaultEnv *telemetryProcessorTestEnv
}

func (t *TelemetryProcessorTestSuite) TearDownSuite() {
	t.defaultEnv.cleanup()
}

func (t *TelemetryProcessorTestSuite) SetupTest() {
	t.defaultEnv, _ = NewProcessorTestEnv("./testdata/config/processor/defaultEnvProcessor.yaml")
}

func (t *TelemetryProcessorTestSuite) AfterTest() {
	t.defaultEnv.cleanup()
}

func (t *TelemetryProcessorTestSuite) TestAddTelemetryDataItem() {
	telemetryType := types.TelemetryType("SLE-SERVER-Test")
	tags := types.Tags{types.Tag("key1=value1"), types.Tag("key2")}
	payload := `
       {
               "ItemA": 1,
               "ItemB": "b",
               "ItemC": "c"
       }
       `

	// test the fileEnv.yaml based datastores
	processor := t.defaultEnv.telemetryprocessor

	err := processor.AddData(telemetryType, []byte(payload), tags)
	if err != nil {
		t.Fail("Test failed to add telemetry data item to datastore")
	}

	processor.cleanup()

}

func (t *TelemetryProcessorTestSuite) TestCreateBundle() {
	// This tests adds two telemetry data items and
	// validates creation of the bundle
	telemetryprocessor := t.defaultEnv.telemetryprocessor

	telemetryType := types.TelemetryType("SLE-SERVER-Test")

	tags := types.Tags{types.Tag("key1=value1"), types.Tag("key2")}

	payload := `
	{
			"field1": "example_data",
			"field2": null,
			"field3": [1, 2, 3]
	}
	`
	err := telemetryprocessor.AddData(telemetryType, []byte(payload), tags)

	if err != nil {
		t.Fail("Test failed to add telemetry data item")
	}

	// Add another data item
	telemetryType = types.TelemetryType("SLE-SERVER-Pkg")
	newtags := types.Tags{types.Tag("key1=value1"), types.Tag("key2")}

	payload = `
	{
		"ItemA": 1,
		"ItemB": "b"
	}
	`

	err = telemetryprocessor.AddData(telemetryType, []byte(payload), newtags)

	if err != nil {
		t.Fail("Test failed to add telemetry data item")
	}

	btags := types.Tags{types.Tag("key1=value1"), types.Tag("key2")}
	bundleRow, berr := telemetryprocessor.GenerateBundle(1, "customer id", btags)

	if berr != nil {
		log.Printf("Failed to create the bundle")
		t.Fail("Test failed to create the bundle")
	}

	// Validate the item count in the bundle generated
	count, _ := telemetryprocessor.DataItemCountInBundle(bundleRow.BundleId)
	assert.Equal(t.T(), 2, count)

	telemetryprocessor.cleanup()

}

func (t *TelemetryProcessorTestSuite) TestReport() {
	// This tests creates two bundles with 5 telemetry data items each
	// creates a report with the two bundles generated, testing for each
	// of the various possible datastore combinations defined in the yaml
	// files in testdata/config folder

	tests := []struct {
		cfgPath string
	}{
		{"./testdata/config/processor/defaultEnvProcessor.yaml"},
	}

	for _, tt := range tests {
		t.Run("validating creation of report with env from "+tt.cfgPath, func() {

			env, _ := NewProcessorTestEnv(tt.cfgPath)
			env.cleanup()
			env.setup()
			telemetryprocessor := env.telemetryprocessor
			itemsCount, _ := telemetryprocessor.DataItemCount()
			assert.Equal(t.T(), 0, itemsCount)
			bundlesCount, _ := telemetryprocessor.BundleCount()
			assert.Equal(t.T(), 0, bundlesCount)
			reportsCount, _ := telemetryprocessor.ReportCount()
			assert.Equal(t.T(), 0, reportsCount)

			err := addDataItems(5, telemetryprocessor)
			assert.NoError(t.T(), err, "Adding first set of dataitems failed")
			itemsCount, _ = telemetryprocessor.DataItemCount()
			assert.Equal(t.T(), 5, itemsCount)

			btags := types.Tags{types.Tag("key1=value1"), types.Tag("key2")}

			bundleRow, berr := telemetryprocessor.GenerateBundle(1, "customer id", btags)
			if berr != nil {
				log.Printf("Failed to create the bundle")
				t.Fail("Test failed to create the bundle")
			}

			// Validate the item count in the bundle generated
			count, _ := telemetryprocessor.DataItemCountInBundle(bundleRow.BundleId)
			assert.Equal(t.T(), 5, count)

			err = addDataItems(5, telemetryprocessor)
			assert.NoError(t.T(), err, "Adding second set of dataitems failed")
			itemsCount, _ = telemetryprocessor.DataItemCount()
			assert.Equal(t.T(), 5, itemsCount)

			btags1 := types.Tags{types.Tag("key3=value3"), types.Tag("key4")}
			bundleRow, berr = telemetryprocessor.GenerateBundle(1, "customer id", btags1)
			if berr != nil {
				log.Printf("Failed to create the bundle")
				t.Fail("Test failed to create the bundle")
			}
			// Validate the item count in the bundle generated
			count, _ = telemetryprocessor.DataItemCountInBundle(bundleRow.BundleId)
			assert.Equal(t.T(), 5, count)

			rtags := types.Tags{types.Tag("key5=value5"), types.Tag("key6")}
			reportRow, err := telemetryprocessor.GenerateReport(123456, rtags)
			assert.NoError(t.T(), err, "Report failed")

			// Validate the bundle count in the report generated
			count, _ = telemetryprocessor.BundleCountInReport(reportRow.ReportId)
			assert.Equal(t.T(), 2, count)

			bundlesCount, _ = telemetryprocessor.BundleCount()
			assert.Equal(t.T(), 0, bundlesCount)

			reportRows, err := telemetryprocessor.GetReportRows()
			assert.NoError(t.T(), err)
			assert.Equal(t.T(), 1, len(reportRows))

			report, err := telemetryprocessor.ToReport(reportRow)
			assert.NoError(t.T(), err)
			assert.Equal(t.T(), report.Header.ReportId, reportRow.ReportId)

			env.cleanup()
		})
	}

}

func (t *TelemetryProcessorTestSuite) TestAddTelemetryDataItemInvalidPayload() {

	payload := `
	{
			"field1": "example_data",
			"field2": null
			"field3": [1, 2, 3]
	}
	`
	telemetryType := types.TelemetryType("SLE-SERVER-Pkg")
	var tags types.Tags

	processor := t.defaultEnv.telemetryprocessor
	err := processor.AddData(telemetryType, []byte(payload), tags)

	expectedmsg := "unable to unmarshal JSON"

	// Check if the string contains the substring
	if !strings.Contains(err.Error(), expectedmsg) {
		t.T().Errorf("String '%s' does not contain substring '%s'", err.Error(), expectedmsg)
	}

}

func addDataItems(totalItems int, processor TelemetryProcessor) error {

	telemetryType := types.TelemetryType("SLE-SERVER-Test")
	var tags types.Tags

	tag1 := types.Tag("key1=value1")
	tag2 := types.Tag("key2")
	tags = append(tags, tag1, tag2)

	var payload = `
		{
			"ItemA": 1,
			"ItemB": "%s"
		}
		`
	numItems := 1
	for numItems <= totalItems {
		formattedJSON := fmt.Sprintf(payload, utils.GenerateRandomString(3))
		err := processor.AddData(telemetryType, []byte(formattedJSON), tags)
		if err != nil {
			log.Printf("Failed to add the item %d", numItems)
			return err
		}

		numItems = numItems + 1
	}
	return nil
}

func TestTelemetryProcessorTestSuite(t *testing.T) {
	suite.Run(t, new(TelemetryProcessorTestSuite))
}
