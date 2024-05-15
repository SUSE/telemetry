package telemetrylib

import (
	"encoding/json"
	"log"

	"github.com/SUSE/telemetry/internal/pkg/datastore"
	"github.com/SUSE/telemetry/pkg/config"
)

type TelemetryCommon interface {
	// Setup the datastore for dataitems, bundles and reports
	setup(*config.DataStoresConfig) error

	// cleanup extractor contents, for testing support
	cleanup()

	// Get a count of telemetry data items
	DataItemCount() (int, error)

	// Get a count of telemetry bundles
	BundleCount() (int, error)

	// Get a count of telemetry reports
	ReportCount() (int, error)

	// Get telemetry data items
	GetDataItems() ([]TelemetryDataItem, error)

	// Delete a specified telemetry data item from the data items datastore
	DeleteDataItem(dataItem *TelemetryDataItem) error

	// Get telemetry bundles
	GetBundles() ([]TelemetryBundle, error)

	// Delete a specified telemetry report from the report datastore
	DeleteBundle(report *TelemetryBundle) error

	// Get telemetry reports
	GetReports() ([]TelemetryReport, error)

	// Delete a specified telemetry report from the report datastore
	DeleteReport(report *TelemetryReport) error
}

type TelemetryCommonImpl struct {
	items   datastore.DataStorer
	bundles datastore.DataStorer
	reports datastore.DataStorer
}

func (t *TelemetryCommonImpl) setup(cfg *config.DataStoresConfig) (err error) {
	err = nil
	// create the telemetry data items data store if not already setup
	if t.items == nil {
		log.Printf("creating new datastore with params %q", cfg.ItemDS)

		t.items, err = datastore.NewDatabaseStore(cfg.ItemDS)

		if err != nil {
			log.Printf("failed to create an items data store with params %q", cfg.ItemDS)
			return
		}
	}

	// create the telemetry bundle data store if not already setup
	if t.bundles == nil {
		log.Printf("creating new datastore of params %q", cfg.BundleDS)

		t.bundles, err = datastore.NewDatabaseStore(cfg.BundleDS)

		if err != nil {
			log.Printf("failed to create a bundle data store with params %q", cfg.BundleDS)
			return
		}
	}

	// create the telemetry report data store if not already setup
	if t.reports == nil {

		log.Printf("creating new datastore with params %q", cfg.ReportDS)

		t.reports, err = datastore.NewDatabaseStore(cfg.ReportDS)

		if err != nil {
			log.Printf("failed to create a report data store with params %q", cfg.ReportDS)
			return
		}
	}

	return

}

func (t *TelemetryCommonImpl) cleanup() (err error) {
	err = nil
	iKeys, _ := t.items.List()
	for _, key := range iKeys {
		err = t.items.Delete(key)
	}

	bKeys, _ := t.bundles.List()
	for _, key := range bKeys {
		err = t.bundles.Delete(key)
	}

	rKeys, _ := t.reports.List()
	for _, key := range rKeys {
		err = t.reports.Delete(key)
	}

	return

}

func (t *TelemetryCommonImpl) DataItemCount() (count int, err error) {
	keys, err := t.items.List()
	if err != nil {
		log.Printf("failed to retrieve list of keys for item store: %s", err.Error())
		return
	}

	count = len(keys)

	return
}

func (t *TelemetryCommonImpl) BundleCount() (count int, err error) {
	keys, err := t.bundles.List()
	if err != nil {
		log.Printf("failed to retrieve list of keys for bundle store: %s", err.Error())
		return
	}

	count = len(keys)

	return
}

func (t *TelemetryCommonImpl) ReportCount() (count int, err error) {
	keys, err := t.reports.List()
	if err != nil {
		log.Printf("failed to retrieve list of keys for report store: %s", err.Error())
		return
	}

	count = len(keys)

	return
}

func (t *TelemetryCommonImpl) GetDataItems() (dataitems []TelemetryDataItem, err error) {

	keys, err := t.items.List()
	if err != nil {
		log.Printf("failed to retrieve list of keys for item store: %s", err.Error())
		return
	}

	for _, j := range keys {
		data, _ := t.items.Get(j)
		var item TelemetryDataItem
		err = json.Unmarshal(data, &item)
		if err != nil {
			log.Printf("failed to unmarshal data item %q: %s", j, err.Error())
			return nil, err
		}

		dataitems = append(dataitems, item)

	}

	return

}

func (t *TelemetryCommonImpl) DeleteDataItem(dataItem *TelemetryDataItem) error {
	return t.items.Delete(dataItem.Key())
}

func (t *TelemetryCommonImpl) DeleteBundle(bundle *TelemetryBundle) error {
	return t.items.Delete(bundle.Key())
}

func (t *TelemetryCommonImpl) GetBundles() (bundles []TelemetryBundle, err error) {

	keys, err := t.bundles.List()
	if err != nil {
		log.Printf("failed to retrieve list of keys for bundle store: %s", err.Error())
		return
	}

	for _, j := range keys {
		data, _ := t.bundles.Get(j)
		var bundle TelemetryBundle
		err = json.Unmarshal(data, &bundle)
		if err != nil {
			log.Printf("failed to unmarshal bundle %q: %s", j, err.Error())
			return nil, err
		}

		bundles = append(bundles, bundle)

	}

	return

}

func (t *TelemetryCommonImpl) GetReports() (reports []TelemetryReport, err error) {

	keys, err := t.reports.List()
	if err != nil {
		log.Printf("failed to retrieve list of keys for report store: %s", err.Error())
		return
	}

	for _, j := range keys {
		data, _ := t.reports.Get(j)
		var report TelemetryReport
		err = json.Unmarshal(data, &report)
		if err != nil {
			log.Printf("failed to unmarshal report %q: %s", j, err.Error())
			return nil, err
		}

		reports = append(reports, report)

	}

	return

}

func (t *TelemetryCommonImpl) DeleteReport(report *TelemetryReport) error {
	return t.reports.Delete(report.Key())
}
