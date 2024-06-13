package telemetrylib

import (
	"github.com/SUSE/telemetry/pkg/config"
)

type TelemetryCommon interface {
	// Setup the datastore for dataitems, bundles and reports
	setup(*config.DBConfig) error

	// cleanup contents, for testing support
	cleanup() error

	// Get a count of telemetry data items that are not associated with a bundle
	ItemCount(bundleIds ...any) (int, error)

	// Get all telemetry data items from the items table
	GetItemRows(bundleIds ...any) ([]*TelemetryDataItemRow, error)

	// Delete a specified telemetry data item from the items table
	DeleteItem(dataItemRow *TelemetryDataItemRow) error

	// Get a count of telemetry bundles that are not associated with a report
	BundleCount(bundleIds ...any) (int, error)

	// Get telemetry bundles from the bundles table
	GetBundleRows(reportIds ...any) ([]*TelemetryBundleRow, error)

	// Delete a specified telemetry bundle from the bundles table
	DeleteBundle(bundleRow *TelemetryBundleRow) error

	// Get a count of telemetry reports
	ReportCount(ids ...any) (int, error)

	// Get telemetry reports from the reports table
	GetReportRows(ids ...any) ([]*TelemetryReportRow, error)

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

func (t *TelemetryCommonImpl) ItemCount(bundleIds ...any) (count int, err error) {
	// count of items matched by specified bundleIds
	count, err = t.storer.GetItemCount(bundleIds...)
	return
}

func (t *TelemetryCommonImpl) BundleCount(reportIds ...any) (count int, err error) {
	// count of bundles matched by specified reportIds
	count, err = t.storer.GetBundleCount(reportIds...)
	return
}

func (t *TelemetryCommonImpl) ReportCount(ids ...any) (count int, err error) {
	// count of reports matched by specified ids
	count, err = t.storer.GetReportCount(ids...)
	return
}

func (t *TelemetryCommonImpl) DeleteItem(itemRow *TelemetryDataItemRow) (err error) {
	err = itemRow.Delete(t.storer.Conn)
	return
}

func (t *TelemetryCommonImpl) DeleteBundle(bundleRow *TelemetryBundleRow) (err error) {
	// foreign key constraint will trigger cascaded delete of
	// associated items
	err = bundleRow.Delete(t.storer.Conn)
	return
}

func (t *TelemetryCommonImpl) DeleteReport(reportRow *TelemetryReportRow) (err error) {
	// foreign key constraints will trigger cascaded delete of
	// associated bundles, and their associated items
	err = reportRow.Delete(t.storer.Conn)
	return
}

func (t *TelemetryCommonImpl) GetItemRows(bundleIds ...any) (itemRows []*TelemetryDataItemRow, err error) {
	_, itemRows, err = t.storer.GetItems(bundleIds...)
	return
}

func (t *TelemetryCommonImpl) GetBundleRows(reportIds ...any) (bundleRows []*TelemetryBundleRow, err error) {
	_, bundleRows, err = t.storer.GetBundles(reportIds...)
	return
}

func (t *TelemetryCommonImpl) GetReportRows(ids ...any) (reportRows []*TelemetryReportRow, err error) {
	_, reportRows, err = t.storer.GetReports(ids...)
	return
}

// validate that TelemetryCommomImpl implements TelemetryCommon interface
var _ TelemetryCommon = (*TelemetryCommonImpl)(nil)
