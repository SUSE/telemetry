package telemetrylib

import (
	"encoding/json"
	"log"
	"strings"
	"testing"

	"github.com/SUSE/telemetry/pkg/config"
	"github.com/SUSE/telemetry/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"github.com/SUSE/telemetry/internal/pkg/datastore"
)

type telemetryExtractorTestEnv struct {
	cfgPath            string
	cfg                *config.Config
	telemetryExtractor TelemetryExtractor
}

func NewExtractorTestEnv(cfgFile string) *telemetryExtractorTestEnv {
	t := telemetryExtractorTestEnv{cfgPath: cfgFile}
	t.setup()
	return &t
}

func (t *telemetryExtractorTestEnv) setup() {
	t.cfg = config.NewConfig(t.cfgPath)
	err := t.cfg.Load()
	if err != nil {
		log.Fatal(err.Error())
	}
	Extractor, err := NewTelemetryExtractor(&t.cfg.DataStores)
	if err != nil {
		log.Fatalf("failed to setup telemetry Extractor for config %q: %s", t.cfgPath, err.Error())
	}
	t.telemetryExtractor = Extractor
}

func (t *telemetryExtractorTestEnv) cleanup() {
	if t.telemetryExtractor != nil {
		t.telemetryExtractor.cleanup()
	}

	if t.cfg != nil {

		itemDS := strings.Split(t.cfg.DataStores.ItemDS, "|")
		bundleDS := strings.Split(t.cfg.DataStores.BundleDS, "|")
		reportDS := strings.Split(t.cfg.DataStores.ReportDS, "|")

		datastore.CleanAll(itemDS[0], itemDS[1])
		datastore.CleanAll(bundleDS[0], bundleDS[1])
		datastore.CleanAll(reportDS[0], reportDS[1])
	}
}

type TelemetryExtractorTestSuite struct {
	suite.Suite
	defaultEnv *telemetryExtractorTestEnv
}

func (t *TelemetryExtractorTestSuite) TearDownSuite() {
	t.defaultEnv.cleanup()
}

func (t *TelemetryExtractorTestSuite) SetupTest() {
	t.defaultEnv = NewExtractorTestEnv("./testdata/config/extractor/defaultEnvExtractor.yaml")
	t.defaultEnv.cleanup()
}

func (t *TelemetryExtractorTestSuite) AfterTest() {
	t.defaultEnv.cleanup()
}

func (t *TelemetryExtractorTestSuite) TestExtractor() {
	var err error

	tests := []struct {
		cfgPath string
	}{
		{"./testdata/config/extractor/itemdb_bundledb_reportfile_Env.yaml"},
		{"./testdata/config/extractor/itemdb_bundlefile_reportfile_Env.yaml"},
		{"./testdata/config/extractor/itemdb_bundlemem_reportfile_Env.yaml"},
		{"./testdata/config/extractor/itemfile_bundledb_reportfile_Env.yaml"},
		{"./testdata/config/extractor/itemfile_bundlefile_reportfile_Env.yaml"},
		{"./testdata/config/extractor/itemfile_bundlemem_reportfile_Env.yaml"},
		{"./testdata/config/extractor/itemmem_bundledb_reportfile_Env.yaml"},
		{"./testdata/config/extractor/itemmem_bundlefile_reportfile_Env.yaml"},
		{"./testdata/config/extractor/itemmem_bundlemem_reportfile_Env.yaml"},
		{"./testdata/config/extractor/itemdb_bundledb_reportdb_Env.yaml"},
		{"./testdata/config/extractor/itemdb_bundlefile_reportdb_Env.yaml"},
		{"./testdata/config/extractor/itemdb_bundlemem_reportdb_Env.yaml"},
		{"./testdata/config/extractor/itemfile_bundledb_reportdb_Env.yaml"},
		{"./testdata/config/extractor/itemfile_bundlefile_reportdb_Env.yaml"},
		{"./testdata/config/extractor/itemfile_bundlemem_reportdb_Env.yaml"},
		{"./testdata/config/extractor/itemmem_bundledb_reportdb_Env.yaml"},
		{"./testdata/config/extractor/itemmem_bundlefile_reportdb_Env.yaml"},
		{"./testdata/config/extractor/itemmem_bundlemem_reportdb_Env.yaml"},
		{"./testdata/config/extractor/itemdb_bundledb_reportmem_Env.yaml"},
		{"./testdata/config/extractor/itemdb_bundlefile_reportmem_Env.yaml"},
		{"./testdata/config/extractor/itemdb_bundlemem_reportmem_Env.yaml"},
		{"./testdata/config/extractor/itemfile_bundledb_reportmem_Env.yaml"},
		{"./testdata/config/extractor/itemfile_bundlefile_reportmem_Env.yaml"},
		{"./testdata/config/extractor/itemfile_bundlemem_reportmem_Env.yaml"},
		{"./testdata/config/extractor/itemmem_bundledb_reportmem_Env.yaml"},
		{"./testdata/config/extractor/itemmem_bundlefile_reportmem_Env.yaml"},
		{"./testdata/config/extractor/itemmem_bundlemem_reportmem_Env.yaml"},
	}

	for _, tt := range tests {
		t.Run("validating creation of report with env from "+tt.cfgPath, func() {

			env := telemetryExtractorTestEnv{cfgPath: tt.cfgPath}
			env.cleanup()
			env.setup()
			extractor := env.telemetryExtractor
			itemsCount, _ := extractor.DataItemCount()
			assert.Equal(t.T(), 0, itemsCount)
			bundlesCount, _ := extractor.BundleCount()
			assert.Equal(t.T(), 0, bundlesCount)
			reportsCount, _ := extractor.ReportCount()
			assert.Equal(t.T(), 0, reportsCount)

			/*
				In order to create a report to be submitted to the extractor, first
				create 2 dataitems, add those dataitems to a bundle, and create a report
				with that bundle and then submit the report to the extractor using
				extractor's AddReport method
			*/

			// Create 2 dataitems
			telemetryType := types.TelemetryType("SLE-SERVER-Test")
			itags1 := types.Tags{types.Tag("ikey1=ivalue1"), types.Tag("ikey2")}
			itags2 := types.Tags{types.Tag("ikey1=ivalue1")}
			payload := `
			{
							"ItemA": 1,
							"ItemB": "b",
							"ItemC": "c"
			}
			`
			var data map[string]interface{}
			err = json.Unmarshal([]byte(payload), &data)
			assert.NoError(t.T(), err, "unmarshalling of dataitem failed")
			item1 := NewTelemetryDataItem(telemetryType, itags1, data)
			item2 := NewTelemetryDataItem(telemetryType, itags2, data)

			// Create 1 bundle
			btags1 := types.Tags{types.Tag("bkey1=bvalue1"), types.Tag("bkey2")}
			bundle1 := NewTelemetryBundle(1, "customer id", btags1)

			// add the two items to the bundle
			bundle1.TelemetryDataItems = append(bundle1.TelemetryDataItems, *item1)
			bundle1.TelemetryDataItems = append(bundle1.TelemetryDataItems, *item2)

			// Create 1 report
			rtags1 := types.Tags{types.Tag("rkey1=rvalue1"), types.Tag("rkey2")}
			report1 := NewTelemetryReport("client id", "auth token", rtags1)

			report1.TelemetryBundles = append(report1.TelemetryBundles, *bundle1)

			// Call the extractor AddReport implementation
			err = extractor.AddReport(report1)
			assert.NoError(t.T(), err, "failed to add telemetry report1 to extractor datastore")
			reportsCount, _ = extractor.ReportCount()
			assert.Equal(t.T(), 1, reportsCount)
			bundlesCount, _ = extractor.BundleCount()
			assert.Equal(t.T(), 0, bundlesCount)
			itemsCount, _ = extractor.DataItemCount()
			assert.Equal(t.T(), 0, itemsCount)

			bundles := report1.TelemetryBundles

			assert.Equal(t.T(), 1, len(bundles))
			for i := 0; i < len(bundles); i++ {
				assert.Equal(t.T(), 2, len(bundles[i].TelemetryDataItems))
			}

			err = extractor.ReportsToBundles()
			assert.NoError(t.T(), err, "failed to add telemetry bundles to extractor datastore")
			bundlesCount, _ = extractor.BundleCount()
			assert.Equal(t.T(), 1, bundlesCount)
			reportsCount, _ = extractor.ReportCount()
			assert.Equal(t.T(), 0, reportsCount)
			itemsCount, _ = extractor.DataItemCount()
			assert.Equal(t.T(), 0, itemsCount)

			bundles, berr := extractor.GetBundles()
			assert.NoError(t.T(), berr, "failed to get telemetry bundles from extractor datastore")
			assert.Equal(t.T(), 1, len(bundles))
			for i := 0; i < len(bundles); i++ {
				assert.Equal(t.T(), 2, len(bundles[i].TelemetryDataItems))
			}

			err = extractor.BundlesToDataItems()
			assert.NoError(t.T(), err, "failed to add telemetry dataitems to extractor datastore")
			itemsCount, _ = extractor.DataItemCount()
			assert.Equal(t.T(), 2, itemsCount)
			bundlesCount, _ = extractor.BundleCount()
			assert.Equal(t.T(), 0, bundlesCount)
			reportsCount, _ = extractor.ReportCount()
			assert.Equal(t.T(), 0, reportsCount)

			items, ierr := extractor.GetDataItems()
			assert.NoError(t.T(), ierr, "failed to get telemetry dataitems from extractor datastore")
			assert.Equal(t.T(), 2, len(items))

			ierr = extractor.DeleteDataItem(&items[0])
			assert.NoError(t.T(), ierr, "failed to delete telemetry dataitems from extractor datastore")
			itemsCount, _ = extractor.DataItemCount()
			assert.Equal(t.T(), 1, itemsCount)

			env.cleanup()
		})
	}

}

func TestTelemetryExtractorTestSuite(t *testing.T) {
	suite.Run(t, new(TelemetryExtractorTestSuite))
}
