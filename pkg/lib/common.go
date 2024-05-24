package telemetrylib

import (
	"github.com/SUSE/telemetry/pkg/config"
)

type TelemetryCommon interface {
	// Setup the datastore for dataitems, bundles and reports
	setup(*config.DBConfig) error

	// cleanup contents, for testing support
	cleanup()

	// Get a count of telemetry data items that are not associated with a bundle
	DataItemCount() (int, error)

	// Get a count of telemetry data items that are associated with a specific bundle
	DataItemCountInBundle(bundleId string) (int, error)

	// Get a count of telemetry bundles that are not associated with a report
	BundleCount() (int, error)

	// Get a count of telemetry bundles that are associated with a specific report
	BundleCountInReport(reportId string) (int, error)

	// Get a count of telemetry reports
	ReportCount() (int, error)

	// Get telemetry data items from the items table
	GetDataItemRows() ([]*TelemetryDataItemRow, error)

	// Delete a specified telemetry data item from the items table
	DeleteDataItem(dataItemRow *TelemetryDataItemRow) error

	// Get telemetry bundles from the bundles table
	GetBundleRows() ([]*TelemetryBundleRow, error)

	// Delete a specified telemetry bundle from the bundles table
	DeleteBundle(bundleRow *TelemetryBundleRow) error

	// Get telemetry reports from the reports table
	GetReportRows() ([]*TelemetryReportRow, error)

	// Delete a specified telemetry report from the reports table
	DeleteReport(reportRow *TelemetryReportRow) error
}

type TelemetryCommonImpl struct {
	storer *DatabaseStore
}

func (t *TelemetryCommonImpl) setup(cfg *config.DBConfig) (err error) {
	t.storer, err = NewDatabaseStore(*cfg)
	return
}

func (t *TelemetryCommonImpl) cleanup() (err error) {
	err = t.storer.dropTables()
	return

}

func (t *TelemetryCommonImpl) DataItemCount() (count int, err error) {
	// count of items that not have been associated with bundle yet
	//_, dataitemRows, err := t.storer.GetItemsWithNoBundleAssociation()
	//count = len(dataitemRows)
	count, err = t.storer.GetDataItemCount()
	return
}

func (t *TelemetryCommonImpl) BundleCount() (count int, err error) {
	// count of bundles that not have been associated with report yet
	count, err = t.storer.GetBundleCount()
	return
}

func (t *TelemetryCommonImpl) ReportCount() (count int, err error) {
	// count of reports
	count, err = t.storer.GetReportCount()
	return
}

func (t *TelemetryCommonImpl) GetDataItems() (dataitemRows []*TelemetryDataItemRow, err error) {
	_, dataitemRows, err = t.storer.GetItemsWithNoBundleAssociation()
	return
}

func (t *TelemetryCommonImpl) DeleteDataItem(dataItemRow *TelemetryDataItemRow) (err error) {
	err = dataItemRow.Delete(t.storer.Conn)
	return
}

func (t *TelemetryCommonImpl) DeleteBundle(bundleRow *TelemetryBundleRow) (err error) {
	err = bundleRow.Delete(t.storer.Conn)
	return
}

func (t *TelemetryCommonImpl) GetBundles() (bundleRows []*TelemetryBundleRow, err error) {
	_, bundleRows, err = t.storer.GetBundlesWithNoReportAssociation()
	return
}

func (t *TelemetryCommonImpl) GetReports() (reportRows []*TelemetryReportRow, err error) {
	_, reportRows, err = t.storer.GetReports()
	return
}

func (t *TelemetryCommonImpl) DeleteReport(reportRow *TelemetryReportRow) (err error) {
	//Delete the report
	err = reportRow.Delete(t.storer.Conn)
	return
}

func (t *TelemetryCommonImpl) DataItemCountInBundle(bundleId string) (count int, err error) {
	// Get a count of telemetry data items that is associated with a bundle
	itemRows, err := t.storer.GetDataItemRowsInABundle(bundleId)
	count = len(itemRows)
	return
}

func (t *TelemetryCommonImpl) BundleCountInReport(reportId string) (count int, err error) {
	// Get a count of telemetry bundles that is associated with a report
	bundleRows, err := t.storer.GetBundleRowsInAReport(reportId)
	count = len(bundleRows)
	return
}

func (t *TelemetryCommonImpl) GetBundleRowsInReport(reportId string) (bundleRows []*TelemetryBundleRow, err error) {
	bundleRows, err = t.storer.GetBundleRowsInAReport(reportId)
	return
}
