package telemetrylib

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/SUSE/telemetry/pkg/config"
	"github.com/SUSE/telemetry/pkg/types"
)

type TelemetryProcessor interface {
	TelemetryCommon

	// Add telemetry data - a method to process jsonData as a byte[]
	AddData(
		telemetry types.TelemetryType,
		content *types.TelemetryBlob,
		tags types.Tags,
	) (err error)

	// Generate telemetry bundle
	GenerateBundle(
		clientId int64,
		customerId string,
		tags types.Tags,
	) (bundleRow *TelemetryBundleRow, err error)

	// Generate telemetry report
	GenerateReport(
		clientId int64,
		tags types.Tags,
	) (reportRow *TelemetryReportRow, err error)

	// Convert TelemetryReportRow structure to TelemetryReport
	ToReport(reportRow *TelemetryReportRow) (report TelemetryReport, err error)

	// Convert TelemetryBundleRow structure to TelemetryBundle
	ToBundle(bundleRow *TelemetryBundleRow) (bundle TelemetryBundle, err error)

	// Convert TelemetryDataItemRow structure to TelemetryDataItem
	ToItem(itemRow *TelemetryDataItemRow) (item TelemetryDataItem, err error)
}

// implements TelemetryProcessor interface.
type TelemetryProcessorImpl struct {
	t   TelemetryCommonImpl
	cfg *config.DBConfig
}

func (p *TelemetryProcessorImpl) setup(cfg *config.DBConfig) (err error) {
	err = p.t.setup(p.cfg)
	return
}

func (p *TelemetryProcessorImpl) cleanup() error {
	return p.t.cleanup()
}

func (p *TelemetryProcessorImpl) ItemCount(bundleIds ...any) (count int, err error) {
	return p.t.ItemCount(bundleIds...)
}

func (p *TelemetryProcessorImpl) BundleCount(reportIds ...any) (count int, err error) {
	return p.t.BundleCount(reportIds...)
}

func (p *TelemetryProcessorImpl) ReportCount(ids ...any) (count int, err error) {
	return p.t.ReportCount(ids...)
}

func (p *TelemetryProcessorImpl) GetItemRows(bundleIds ...any) (dataitemsRows []*TelemetryDataItemRow, err error) {
	return p.t.GetItemRows(bundleIds...)
}

func (p *TelemetryProcessorImpl) DeleteItem(dataItemRow *TelemetryDataItemRow) (err error) {
	return p.t.DeleteItem(dataItemRow)
}

func (p *TelemetryProcessorImpl) GetBundleRows(reportIds ...any) (bundleRows []*TelemetryBundleRow, err error) {
	return p.t.GetBundleRows(reportIds...)
}

func (p *TelemetryProcessorImpl) DeleteBundle(bundleRow *TelemetryBundleRow) (err error) {
	return p.t.DeleteBundle(bundleRow)
}

func (p *TelemetryProcessorImpl) GetReportRows(ids ...any) (reportRows []*TelemetryReportRow, err error) {
	return p.t.GetReportRows(ids...)
}

func (p *TelemetryProcessorImpl) DeleteReport(reportRow *TelemetryReportRow) (err error) {
	return p.t.DeleteReport(reportRow)
}

// validate TelemetryProcessorImpl implements the TelemetryProcessor interface
var _ TelemetryProcessor = (*TelemetryProcessorImpl)(nil)

func NewTelemetryProcessor(cfg *config.DBConfig) (TelemetryProcessor, error) {
	slog.Debug("NewTelemetryProcessor", slog.Any("cfg", cfg))
	p := TelemetryProcessorImpl{cfg: cfg}

	err := p.setup(cfg)

	return &p, err
}

func (p *TelemetryProcessorImpl) AddData(telemetry types.TelemetryType, marshaledData *types.TelemetryBlob, tags types.Tags) (err error) {
	dataItemRow := NewTelemetryDataItemRow(telemetry, tags, marshaledData)

	return dataItemRow.Insert(p.t.storer.Conn)
}

func (p *TelemetryProcessorImpl) GenerateBundle(clientId int64, customerId string, tags types.Tags) (bundleRow *TelemetryBundleRow, err error) {

	bundleRow, err = NewTelemetryBundleRow(clientId, customerId, tags)
	if err != nil {
		return bundleRow, fmt.Errorf("unable to create bundle: %s", err.Error())
	}

	//List all items that are not associated with bundle yet
	itemIDs, _, err := p.t.storer.GetItems("NULL")
	if err != nil {
		return bundleRow, fmt.Errorf("unable to get items for bundle generation: %s", err.Error())
	}

	_, err = bundleRow.Insert(p.t.storer.Conn, itemIDs)

	if err != nil {
		return bundleRow, fmt.Errorf("unable to insert bundle: %s", err.Error())
	}
	return
}

func (p *TelemetryProcessorImpl) GenerateReport(clientId int64, tags types.Tags) (reportRow *TelemetryReportRow, err error) {

	reportRow, err = NewTelemetryReportRow(clientId, tags)
	if err != nil {
		return reportRow, fmt.Errorf("unable to create report: %s", err.Error())
	}

	//List all bundles that are not associated with report yet
	bundleIDs, _, err := p.t.storer.GetBundles("NULL")

	if err != nil {
		return reportRow, fmt.Errorf("unable to get bundles for the report generation: %s", err.Error())
	}

	_, err = reportRow.Insert(p.t.storer.Conn, bundleIDs)

	if err != nil {
		return reportRow, fmt.Errorf("unable to insert report: %s", err.Error())
	}

	return

}

func (p *TelemetryProcessorImpl) ToReport(reportRow *TelemetryReportRow) (report TelemetryReport, err error) {
	// Convert TelemetryReportRow structure to TelemetryReport

	annotations := strings.Split(reportRow.ReportAnnotations, ",")

	reportHeader := TelemetryReportHeader{
		ReportId:          reportRow.ReportId,
		ReportTimeStamp:   reportRow.ReportTimestamp,
		ReportClientId:    reportRow.ReportClientId,
		ReportAnnotations: annotations,
	}

	reportFooter := TelemetryReportFooter{
		Checksum: reportRow.ReportChecksum,
	}

	_, bundleRows, err := p.t.storer.GetBundles(reportRow.Id)

	var bundles []TelemetryBundle

	for i := 0; i < len(bundleRows); i++ {
		bundleRow := bundleRows[i]
		var bundle TelemetryBundle
		bundle, err = p.ToBundle(bundleRow)
		bundles = append(bundles, bundle)

	}

	report = TelemetryReport{
		Header:           reportHeader,
		TelemetryBundles: bundles,
		Footer:           reportFooter,
	}

	return

}

func (p *TelemetryProcessorImpl) ToBundle(bundleRow *TelemetryBundleRow) (bundle TelemetryBundle, err error) {
	// Convert TelemetryBundleRow structure to TelemetryBundle
	annotations := strings.Split(bundleRow.BundleAnnotations, ",")

	bundleHeader := TelemetryBundleHeader{
		BundleId:          bundleRow.BundleId,
		BundleTimeStamp:   bundleRow.BundleTimestamp,
		BundleClientId:    bundleRow.BundleClientId,
		BundleCustomerId:  bundleRow.BundleCustomerId,
		BundleAnnotations: annotations,
	}

	bundleFooter := TelemetryBundleFooter{
		Checksum: bundleRow.BundleChecksum,
	}

	_, itemRows, err := p.t.storer.GetItems(bundleRow.Id)
	var items []TelemetryDataItem

	for j := 0; j < len(itemRows); j++ {
		var item TelemetryDataItem
		itemRow := itemRows[j]
		item, err = p.ToItem(itemRow)
		items = append(items, item)

	}

	bundle = TelemetryBundle{
		Header:             bundleHeader,
		TelemetryDataItems: items,
		Footer:             bundleFooter,
	}

	return

}

func (p *TelemetryProcessorImpl) ToItem(itemRow *TelemetryDataItemRow) (item TelemetryDataItem, err error) {
	// Convert TelemetryDataItemRow structure to TelemetryDataItem
	annotations := strings.Split(itemRow.ItemAnnotations, ",")
	itemHeader := TelemetryDataItemHeader{
		TelemetryId:          itemRow.ItemId,
		TelemetryTimeStamp:   itemRow.ItemTimestamp,
		TelemetryType:        itemRow.ItemType,
		TelemetryAnnotations: annotations,
	}

	itemFooter := TelemetryDataItemFooter{
		Checksum: itemRow.ItemChecksum,
	}

	//data, err := utils.DeserializeMap(string(itemRow.ItemData))

	item = TelemetryDataItem{
		Header:        itemHeader,
		TelemetryData: itemRow.ItemData,
		Footer:        itemFooter,
	}

	return

}
