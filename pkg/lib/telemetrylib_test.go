package telemetrylib

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"testing"

	"github.com/SUSE/telemetry/pkg/config"
	"github.com/SUSE/telemetry/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"github.com/SUSE/telemetry/internal/pkg/datastore"
)

type telemetryProcessorTestEnv struct {
	cfgPath            string
	cfg                *config.Config
	telemetryprocessor TelemetryProcessor
}

func NewTestEnv(cfgFile string) *telemetryProcessorTestEnv {
	t := telemetryProcessorTestEnv{cfgPath: cfgFile}
	t.setup()
	return &t
}

func (t *telemetryProcessorTestEnv) setup() {
	t.cfg = config.NewConfig(t.cfgPath)
	err := t.cfg.Load()
	if err != nil {
		log.Fatal(err.Error())
	}
	processor, err := NewTelemetryProcessor(t.cfg)
	if err != nil {
		log.Fatalf("failed to setup telemetry processor for config %q: %s", t.cfgPath, err.Error())
	}
	t.telemetryprocessor = processor
}

func (t *telemetryProcessorTestEnv) cleanup() {
	if t.telemetryprocessor != nil {
		t.telemetryprocessor.cleanup()
	}

	if t.cfg != nil {
		datastore.CleanAll(t.cfg.ItemDS.Type, t.cfg.ItemDS.Path)
		datastore.CleanAll(t.cfg.BundleDS.Type, t.cfg.BundleDS.Path)
	}
}

type TelemetryTestSuite struct {
	suite.Suite
	defaultEnv *telemetryProcessorTestEnv
}

func (t *TelemetryTestSuite) TearDownSuite() {
	t.defaultEnv.cleanup()
}

func (t *TelemetryTestSuite) SetupTest() {
	t.defaultEnv = NewTestEnv("./testdata/config/defaultEnv.yaml")
	t.defaultEnv.cleanup()
}

func (t *TelemetryTestSuite) AfterTest() {
	t.defaultEnv.cleanup()
}

func (t *TelemetryTestSuite) TestAddTelemetryDataItem() {
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

	_, err := processor.AddData(telemetryType, []byte(payload), tags)
	if err != nil {
		t.Fail("Test failed to add telemetry data item to datastore")
	}

}

func (t *TelemetryTestSuite) TestCreateBundle() {
	// This tests adds two telemetry data items and
	// validates creation of the bundle
	telmetryprocessor := t.defaultEnv.telemetryprocessor

	telemetryType := types.TelemetryType("SLE-SERVER-Test")

	tags := types.Tags{types.Tag("key1=value1"), types.Tag("key2")}

	payload := `
	{
			"field1": "example_data",
			"field2": null,
			"field3": [1, 2, 3]
	}
	`
	_, err := telmetryprocessor.AddData(telemetryType, []byte(payload), tags)

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

	_, err = telmetryprocessor.AddData(telemetryType, []byte(payload), newtags)

	if err != nil {
		t.Fail("Test failed to add telemetry data item")
	}

	btags := types.Tags{types.Tag("key1=value1"), types.Tag("key2")}
	b, berr := telmetryprocessor.GenerateBundle("client id", "customer id", btags)
	if berr != nil {
		log.Printf("Failed to create the bundle")
		t.Fail("Test failed to create the bundle")
	}
	assert.Equal(t.T(), 2, len(b.TelemetryDataItems))

}

func (t *TelemetryTestSuite) TestReport() {
	// This tests creates two bundles with 5 telemetry data items each
	// creates a report with the two bundles generated, testing for each
	// of the various possible datastore combinations defined in the yaml
	// files in testdata/config folder

	tests := []struct {
		cfgPath string
	}{
		{"./testdata/config/itemdb_bundledb_reportfile_Env.yaml"},
		{"./testdata/config/itemdb_bundlefile_reportfile_Env.yaml"},
		{"./testdata/config/itemdb_bundlemem_reportfile_Env.yaml"},
		{"./testdata/config/itemfile_bundledb_reportfile_Env.yaml"},
		{"./testdata/config/itemfile_bundlefile_reportfile_Env.yaml"},
		{"./testdata/config/itemfile_bundlemem_reportfile_Env.yaml"},
		{"./testdata/config/itemmem_bundledb_reportfile_Env.yaml"},
		{"./testdata/config/itemmem_bundlefile_reportfile_Env.yaml"},
		{"./testdata/config/itemmem_bundlemem_reportfile_Env.yaml"},
		{"./testdata/config/itemdb_bundledb_reportdb_Env.yaml"},
		{"./testdata/config/itemdb_bundlefile_reportdb_Env.yaml"},
		{"./testdata/config/itemdb_bundlemem_reportdb_Env.yaml"},
		{"./testdata/config/itemfile_bundledb_reportdb_Env.yaml"},
		{"./testdata/config/itemfile_bundlefile_reportdb_Env.yaml"},
		{"./testdata/config/itemfile_bundlemem_reportdb_Env.yaml"},
		{"./testdata/config/itemmem_bundledb_reportdb_Env.yaml"},
		{"./testdata/config/itemmem_bundlefile_reportdb_Env.yaml"},
		{"./testdata/config/itemmem_bundlemem_reportdb_Env.yaml"},
		{"./testdata/config/itemdb_bundledb_reportmem_Env.yaml"},
		{"./testdata/config/itemdb_bundlefile_reportmem_Env.yaml"},
		{"./testdata/config/itemdb_bundlemem_reportmem_Env.yaml"},
		{"./testdata/config/itemfile_bundledb_reportmem_Env.yaml"},
		{"./testdata/config/itemfile_bundlefile_reportmem_Env.yaml"},
		{"./testdata/config/itemfile_bundlemem_reportmem_Env.yaml"},
		{"./testdata/config/itemmem_bundledb_reportmem_Env.yaml"},
		{"./testdata/config/itemmem_bundlefile_reportmem_Env.yaml"},
		{"./testdata/config/itemmem_bundlemem_reportmem_Env.yaml"},
	}

	for _, tt := range tests {
		t.Run("validating creation of report with env from "+tt.cfgPath, func() {
			env := telemetryProcessorTestEnv{cfgPath: tt.cfgPath}
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
			bundle, berr := telemetryprocessor.GenerateBundle("client id", "customer id", btags)
			if berr != nil {
				log.Printf("Failed to create the bundle")
				t.Fail("Test failed to create the bundle")
			}
			assert.Equal(t.T(), 5, len(bundle.TelemetryDataItems))

			err = addDataItems(5, telemetryprocessor)
			assert.NoError(t.T(), err, "Adding second set of dataitems failed")
			itemsCount, _ = telemetryprocessor.DataItemCount()
			assert.Equal(t.T(), 5, itemsCount)

			btags1 := types.Tags{types.Tag("key3=value3"), types.Tag("key4")}
			bundle1, berr := telemetryprocessor.GenerateBundle("client id", "customer id", btags1)
			if berr != nil {
				log.Printf("Failed to create the bundle")
				t.Fail("Test failed to create the bundle")
			}
			assert.Equal(t.T(), 5, len(bundle1.TelemetryDataItems))

			rtags := types.Tags{types.Tag("key5=value5"), types.Tag("key6")}
			report, err := telemetryprocessor.GenerateReport("client id", "auth token", rtags)
			assert.NoError(t.T(), err, "Report failed")

			assert.Equal(t.T(), 2, len(report.TelemetryBundles))
			bundlesCount, _ = telemetryprocessor.BundleCount()
			assert.Equal(t.T(), 0, bundlesCount)

			env.cleanup()
		})
	}

}

func (t *TelemetryTestSuite) TestAddTelemetryDataItemInvalidPayload() {

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
	_, err := processor.AddData(telemetryType, []byte(payload), tags)

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
	var err error
	numItems := 1
	for numItems <= totalItems {
		formattedJSON := fmt.Sprintf(payload, datastore.GenerateRandomString(3))

		// Decode JSON string into a map
		var data map[string]interface{}
		err = json.Unmarshal([]byte(formattedJSON), &data)
		if err != nil {
			log.Println("Error:", err)
			return err
		}

		// Encode Data struct back to JSON format
		formattedData, err := json.MarshalIndent(data, "", "    ")
		if err != nil {
			log.Println("Error:", err)
			return err
		}

		_, err = processor.AddData(telemetryType, []byte(string(formattedData)), tags)
		if err != nil {
			log.Printf("Failed to add the item %d", numItems)
			return err
		}

		numItems = numItems + 1
	}
	return nil
}

func TestTelemetryTestSuite(t *testing.T) {
	suite.Run(t, new(TelemetryTestSuite))
}
