package telemetrylib

import (
	"fmt"
	"log/slog"
	"testing"

	"github.com/SUSE/telemetry/pkg/config"
	"github.com/SUSE/telemetry/pkg/types"
	"github.com/google/uuid"
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
		slog.Error(
			"Failed to load config",
			slog.String("cfgPath", t.cfgPath),
			slog.String("error", err.Error()),
		)
		return
	}
	t.cfg.ClientId = uuid.NewString()
	processor, err := NewTelemetryProcessor(&t.cfg.DataStores)
	if err != nil {
		slog.Error(
			"Failed to setup telemetry processor",
			slog.String("cfgPath", t.cfgPath),
			slog.String("error", err.Error()),
		)
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
	payload := types.NewTelemetryBlob([]byte(`{
		"ItemA": 1,
		"ItemB": "b",
		"ItemC": "c"
	}`))

	// test the fileEnv.yaml based datastores
	processor := t.defaultEnv.telemetryprocessor

	err := processor.AddData(telemetryType, payload, tags)
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

	payload := types.NewTelemetryBlob([]byte(`{
		"field1": "example_data",
		"field2": null,
		"field3": [1, 2, 3]
	}`))
	err := telemetryprocessor.AddData(telemetryType, payload, tags)

	if err != nil {
		t.Fail("Test failed to add telemetry data item")
	}

	// Add another data item
	telemetryType = types.TelemetryType("SLE-SERVER-Pkg")
	newtags := types.Tags{types.Tag("key1=value1"), types.Tag("key2")}

	payload = types.NewTelemetryBlob([]byte(`{
		"ItemA": 1,
		"ItemB": "b"
	}`))

	err = telemetryprocessor.AddData(telemetryType, payload, newtags)

	if err != nil {
		t.Fail("Test failed to add telemetry data item")
	}

	btags := types.Tags{types.Tag("key1=value1"), types.Tag("key2")}
	bundleRow, berr := telemetryprocessor.GenerateBundle(t.defaultEnv.cfg.ClientId, "customer id", btags)

	if berr != nil {
		t.Fail("Test failed to create the bundle")
	}

	// Validate the item count in the bundle generated
	count, _ := telemetryprocessor.ItemCount(bundleRow.Id)
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

			// validate that there are no items present
			allItemsCount, _ := telemetryprocessor.ItemCount()
			assert.Equal(t.T(), 0, allItemsCount)

			// validate that there are no unassigned items present
			unassignedItemsCount, _ := telemetryprocessor.ItemCount("NULL")
			assert.Equal(t.T(), 0, unassignedItemsCount)

			// validate that there are no bundles present
			allBundlesCount, _ := telemetryprocessor.BundleCount()
			assert.Equal(t.T(), 0, allBundlesCount)

			// validate that there are no unassigned bundles present
			unassignedBundlesCount, _ := telemetryprocessor.BundleCount("NULL")
			assert.Equal(t.T(), 0, unassignedBundlesCount)

			// validate that there are no reports present
			reportsCount, _ := telemetryprocessor.ReportCount()
			assert.Equal(t.T(), 0, reportsCount)

			// add some unassigned data items
			err := addDataItems(5, telemetryprocessor)
			assert.NoError(t.T(), err, "Adding first set of dataitems failed")

			// validate the total and unassigned item counts
			allItemsCount, _ = telemetryprocessor.ItemCount()
			assert.Equal(t.T(), 5, allItemsCount)
			unassignedItemsCount, _ = telemetryprocessor.ItemCount("NULL")
			assert.Equal(t.T(), 5, unassignedItemsCount)

			// validate that we get the expected number of unassigned item rows
			itemRows, err := telemetryprocessor.GetItemRows("NULL")
			assert.NoError(t.T(), err)
			assert.Equal(t.T(), 5, len(itemRows))

			// validate that we can render items
			for _, itemRow := range itemRows {
				item, err := telemetryprocessor.ToItem(itemRow)
				assert.NoError(t.T(), err)
				assert.Equal(t.T(), item.Header.TelemetryId, itemRow.ItemId)
			}

			// generate a bundle to hold unassigned items
			btags := types.Tags{types.Tag("key1=value1"), types.Tag("key2")}
			bundleRow, berr := telemetryprocessor.GenerateBundle(env.cfg.ClientId, "customer id", btags)
			if berr != nil {
				t.Fail("Test failed to create the bundle")
			}

			// validate the total and unassigned item counts
			allItemsCount, _ = telemetryprocessor.ItemCount()
			assert.Equal(t.T(), 5, allItemsCount)
			unassignedItemsCount, _ = telemetryprocessor.ItemCount("NULL")
			assert.Equal(t.T(), 0, unassignedItemsCount)

			// Validate the item count in the bundle generated
			count, _ := telemetryprocessor.ItemCount(bundleRow.Id)
			assert.Equal(t.T(), 5, count)

			// validate the total and unassigned bundle counts
			allBundlesCount, _ = telemetryprocessor.BundleCount()
			assert.Equal(t.T(), 1, allBundlesCount)
			unassignedBundlesCount, _ = telemetryprocessor.BundleCount("NULL")
			assert.Equal(t.T(), 1, unassignedBundlesCount)

			// Add more items
			err = addDataItems(5, telemetryprocessor)
			assert.NoError(t.T(), err, "Adding second set of dataitems failed")

			// validate the total and unassigned item counts
			allItemsCount, _ = telemetryprocessor.ItemCount()
			assert.Equal(t.T(), 10, allItemsCount)
			unassignedItemsCount, _ = telemetryprocessor.ItemCount("NULL")
			assert.Equal(t.T(), 5, unassignedItemsCount)

			// generate a second bundle
			btags1 := types.Tags{types.Tag("key3=value3"), types.Tag("key4")}
			bundleRow, berr = telemetryprocessor.GenerateBundle(env.cfg.ClientId, "customer id", btags1)
			if berr != nil {
				t.Fail("Test failed to create the bundle")
			}

			// validate the total and unassigned item counts
			allItemsCount, _ = telemetryprocessor.ItemCount()
			assert.Equal(t.T(), 10, allItemsCount)
			unassignedItemsCount, _ = telemetryprocessor.ItemCount("NULL")
			assert.Equal(t.T(), 0, unassignedItemsCount)

			// Validate the item count in the bundle generated
			count, _ = telemetryprocessor.ItemCount(bundleRow.Id)
			assert.Equal(t.T(), 5, count)

			// validate the total and unassigned bundle counts
			allBundlesCount, _ = telemetryprocessor.BundleCount()
			assert.Equal(t.T(), 2, allBundlesCount)
			unassignedBundlesCount, _ = telemetryprocessor.BundleCount("NULL")
			assert.Equal(t.T(), 2, unassignedBundlesCount)

			// validate that we get the expected number of unassigned bundle rows
			bundleRows, err := telemetryprocessor.GetBundleRows("NULL")
			assert.NoError(t.T(), err)
			assert.Equal(t.T(), 2, len(bundleRows))

			// validate that we can render bundles
			for _, bundleRow := range bundleRows {
				bundle, err := telemetryprocessor.ToBundle(bundleRow)
				assert.NoError(t.T(), err)
				assert.Equal(t.T(), bundle.Header.BundleId, bundleRow.BundleId)
			}

			// generate a report consuming available bundles
			rtags := types.Tags{types.Tag("key5=value5"), types.Tag("key6")}
			reportRow, err := telemetryprocessor.GenerateReport(env.cfg.ClientId, rtags)
			assert.NoError(t.T(), err, "Report failed")

			// validate the total and unassigned bundle counts
			allBundlesCount, _ = telemetryprocessor.BundleCount()
			assert.Equal(t.T(), 2, allBundlesCount)
			unassignedBundlesCount, _ = telemetryprocessor.BundleCount("NULL")
			assert.Equal(t.T(), 0, unassignedBundlesCount)

			// validate the bundle count in the report generated
			count, _ = telemetryprocessor.BundleCount(reportRow.Id)
			assert.Equal(t.T(), 2, count)

			// validate the total number of reports
			reportsCount, _ = telemetryprocessor.ReportCount()
			assert.Equal(t.T(), 1, reportsCount)

			// validate that we can get the report rows and that the count is as expected
			reportRows, err := telemetryprocessor.GetReportRows()
			assert.NoError(t.T(), err)
			assert.Equal(t.T(), 1, len(reportRows))

			// validate that we can render reports
			for _, reportRow := range reportRows {
				report, err := telemetryprocessor.ToReport(reportRow)
				assert.NoError(t.T(), err)
				assert.Equal(t.T(), report.Header.ReportId, reportRow.ReportId)

				// delete the report
				telemetryprocessor.DeleteReport(reportRow)
			}

			// validate that there are no reports present after deleting reports
			reportsCount, _ = telemetryprocessor.ReportCount()
			assert.Equal(t.T(), 0, reportsCount)

			// validate that there are no bundles present after cascaded delete
			allBundlesCount, _ = telemetryprocessor.BundleCount()
			assert.Equal(t.T(), 0, allBundlesCount)

			// validate that there are no items present after cascaded delete
			allItemsCount, _ = telemetryprocessor.ItemCount()
			assert.Equal(t.T(), 0, allItemsCount)

			env.cleanup()
		})
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
		formattedJSON := types.NewTelemetryBlob([]byte(fmt.Sprintf(payload, uuid.New().String())))
		err := processor.AddData(telemetryType, formattedJSON, tags)
		if err != nil {
			slog.Error(
				"Failed to add the item",
				slog.Int("numItems", numItems),
				slog.String("error", err.Error()),
			)
			return err
		}

		numItems = numItems + 1
	}
	return nil
}

func TestTelemetryProcessorTestSuite(t *testing.T) {
	suite.Run(t, new(TelemetryProcessorTestSuite))
}
