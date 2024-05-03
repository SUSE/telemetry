package telemetrylib

import (
	"encoding/json"
	"log"

	"github.com/SUSE/telemetry/pkg/config"
)

type TelemetryExtractor interface {
	TelemetryCommon

	// Add Report to staging report datastore
	AddReport(report *TelemetryReport) error

	// Extract bundles from reports in the staging report datasore and store the extracted bundles in staging bundle datastore
	ReportToBundles() error

	// Extract telemetry data items from bundles in the staging bundle datastore and store the extracted items in the item datastore
	BundlesToDataItems() error

	// Get telemetry data items
	GetDataItems() ([]TelemetryDataItem, error)

	// Get telemetry bundles
	GetBundles() ([]TelemetryBundle, error)

	// Get telemetry reports
	GetReports() ([]TelemetryReport, error)
}

// implements TelemetryExtractor interface.
type TelemetryExtractorImpl struct {
	t   TelemetryCommonImpl
	cfg *config.Config // this will be server side configuration
}

func (e *TelemetryExtractorImpl) setup(*config.Config) (err error) {
	err = e.t.setup(e.cfg)
	return
}

func (e *TelemetryExtractorImpl) cleanup() {
	e.t.cleanup()
}

func (e *TelemetryExtractorImpl) DataItemCount() (count int, err error) {
	return e.t.DataItemCount()
}

func (e *TelemetryExtractorImpl) BundleCount() (count int, err error) {
	return e.t.BundleCount()
}

func (e *TelemetryExtractorImpl) ReportCount() (count int, err error) {
	return e.t.ReportCount()
}

func (e *TelemetryExtractorImpl) GetDataItems() (dataitems []TelemetryDataItem, err error) {
	return e.t.GetDataItems()
}

func (e *TelemetryExtractorImpl) DeleteDataItem(dataItem *TelemetryDataItem) (err error) {
	return e.t.DeleteDataItem(dataItem)
}

func (e *TelemetryExtractorImpl) GetBundles() (bundles []TelemetryBundle, err error) {
	return e.t.GetBundles()
}

func (e *TelemetryExtractorImpl) GetReports() (reports []TelemetryReport, err error) {
	return e.t.GetReports()
}

func (e *TelemetryExtractorImpl) DeleteReport(report *TelemetryReport) (err error) {
	return e.t.DeleteReport(report)
}

func NewTelemetryExtractor(cfg *config.Config) (TelemetryExtractor, error) {
	log.Printf("NewTelemetryExtractor(%+v)", cfg)
	e := TelemetryExtractorImpl{cfg: cfg}

	err := e.setup(cfg)

	return &e, err
}

func (e *TelemetryExtractorImpl) AddReport(report *TelemetryReport) (err error) {
	rKey := report.Key()
	log.Printf("adding a report with ID %s\n", rKey)

	jsonData, err := json.Marshal(report)
	if err != nil {
		log.Println("error marshalling report:", err)
		return
	}

	err = e.t.reports.Add(rKey, []byte(jsonData))
	if err != nil {
		log.Println("error adding report:", err)
		return
	}

	return nil
}

func (e *TelemetryExtractorImpl) ReportToBundles() error {

	// list all the reports in the report datastore
	reportKeys, err := e.t.reports.List()
	if err != nil {
		log.Println("failed to retrieve the reports", err)
		return err
	}

	// Get each report in the report datastore and extract bundles from that report into bundle datastore
	for _, key := range reportKeys {
		value, err := e.t.reports.Get(key)
		if err != nil {
			log.Printf("unable to get report %q: %s", key, err.Error())
			return err
		}

		var report TelemetryReport
		err = json.Unmarshal(value, &report)
		if err != nil {
			log.Printf("unable to marshal the report %q: %s", key, err.Error())
			return err
		}

		for _, bundle := range report.TelemetryBundles {

			bKey := bundle.Key()
			log.Printf("adding a bundle with ID %s\n", bKey)

			jsonData, err := json.Marshal(bundle)
			if err != nil {
				log.Println("error marshalling bundle:", err)
				return err
			}

			err = e.t.bundles.Add(bKey, []byte(jsonData))
			if err != nil {
				log.Println("error adding bundle:", err)
				return err
			}
		}

		// Delete already handled report
		e.t.reports.Delete(report.Key())

	}

	return nil
}

func (e *TelemetryExtractorImpl) BundlesToDataItems() error {

	// list all the bundles in the bundle datastore
	bundleKeys, err := e.t.bundles.List()
	if err != nil {
		log.Println("failed to retrieve the bundles", err)
		return err
	}

	// Get each bundle in the bundle datastore and extract items from that bundle into staging item datastore
	for _, key := range bundleKeys {
		value, err := e.t.bundles.Get(key)
		if err != nil {
			log.Printf("unable to get bundle %q: %s", key, err.Error())
			return err
		}

		var bundle TelemetryBundle
		err = json.Unmarshal(value, &bundle)
		if err != nil {
			log.Printf("unable to marshal the bundle %q: %s", key, err.Error())
			return err
		}

		for _, item := range bundle.TelemetryDataItems {

			iKey := item.Key()
			log.Printf("adding a data item with ID %s\n", iKey)

			jsonData, err := json.Marshal(item)
			if err != nil {
				log.Println("error marshalling dataitem:", err)
				return err
			}

			err = e.t.items.Add(iKey, []byte(jsonData))
			if err != nil {
				log.Println("error adding dataitem:", err)
				return err
			}
		}

		// Delete already handled bundle
		e.t.bundles.Delete(bundle.Key())

	}

	return nil

}
