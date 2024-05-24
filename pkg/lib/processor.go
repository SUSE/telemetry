package telemetrylib

import (
	"fmt"
	"log"
	"strings"

	"github.com/SUSE/telemetry/pkg/config"
	"github.com/SUSE/telemetry/pkg/types"
	"github.com/SUSE/telemetry/pkg/utils"
)

type TelemetryProcessor interface {
	TelemetryCommon

	// Add telemetry data - a method to process jsonData as a byte[]
	AddData(
		telemetry types.TelemetryType,
		content []byte,
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

func (p *TelemetryProcessorImpl) cleanup() {
	p.t.cleanup()
}

func (p *TelemetryProcessorImpl) DataItemCount() (count int, err error) {
	return p.t.DataItemCount()
}

func (p *TelemetryProcessorImpl) BundleCount() (count int, err error) {
	return p.t.BundleCount()
}

func (p *TelemetryProcessorImpl) ReportCount() (count int, err error) {
	return p.t.ReportCount()
}

func (p *TelemetryProcessorImpl) GetDataItemRows() (dataitemsRows []*TelemetryDataItemRow, err error) {
	return p.t.GetDataItems()
}

func (p *TelemetryProcessorImpl) DeleteDataItem(dataItemRow *TelemetryDataItemRow) (err error) {
	return p.t.DeleteDataItem(dataItemRow)
}

func (p *TelemetryProcessorImpl) GetBundleRows() (bundleRows []*TelemetryBundleRow, err error) {
	return p.t.GetBundles()
}

func (p *TelemetryProcessorImpl) DeleteBundle(bundleRow *TelemetryBundleRow) (err error) {
	return p.t.DeleteBundle(bundleRow)
}

func (p *TelemetryProcessorImpl) GetReportRows() (reportRows []*TelemetryReportRow, err error) {
	return p.t.GetReports()
}

func (p *TelemetryProcessorImpl) DeleteReport(reportRow *TelemetryReportRow) (err error) {
	return p.t.DeleteReport(reportRow)
}

func NewTelemetryProcessor(cfg *config.DBConfig) (TelemetryProcessor, error) {
	log.Printf("NewTelemetryProcessor(%+v)", cfg)
	p := TelemetryProcessorImpl{cfg: cfg}

	err := p.setup(cfg)

	return &p, err
}

func (p *TelemetryProcessorImpl) AddData(telemetry types.TelemetryType, marshaledData []byte, tags types.Tags) (err error) {
	dataItemRow, err := NewTelemetryDataItemRow(telemetry, tags, marshaledData)
	if err != nil {
		return fmt.Errorf("unable to create telemetry data: %s", err.Error())
	}

	err = dataItemRow.Insert(p.t.storer.Conn)
	return
}

func (p *TelemetryProcessorImpl) GenerateBundle(clientId int64, customerId string, tags types.Tags) (bundleRow *TelemetryBundleRow, err error) {

	bundleRow, err = NewTelemetryBundleRow(clientId, customerId, tags)
	if err != nil {
		return bundleRow, fmt.Errorf("unable to create bundle: %s", err.Error())
	}

	//List all items that are not associated with bundle yet
	itemIDs, _, err := p.t.storer.GetItemsWithNoBundleAssociation()
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
	bundleIDs, _, err := p.t.storer.GetBundlesWithNoReportAssociation()

	if err != nil {
		return reportRow, fmt.Errorf("unable to get bundles for the report generation: %s", err.Error())
	}

	_, err = reportRow.Insert(p.t.storer.Conn, bundleIDs)

	if err != nil {
		return reportRow, fmt.Errorf("unable to insert report: %s", err.Error())
	}

	return

}

func (p *TelemetryProcessorImpl) DataItemCountInBundle(bundleId string) (count int, err error) {
	// Get a count of telemetry data items that is associated with a bundle
	return p.t.DataItemCountInBundle(bundleId)
}

func (p *TelemetryProcessorImpl) BundleCountInReport(reportId string) (count int, err error) {
	// Get a count of telemetry bundles that is associated with a report
	return p.t.BundleCountInReport(reportId)
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

	bundleRows, err := p.t.storer.GetBundleRowsInAReport(reportRow.ReportId)

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

	itemRows, err := p.t.storer.GetDataItemRowsInABundle(bundleRow.BundleId)
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

	data, err := utils.DeserializeMap(itemRow.ItemData)

	item = TelemetryDataItem{
		Header:        itemHeader,
		TelemetryData: data,
		Footer:        itemFooter,
	}

	return

}
