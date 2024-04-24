package telemetrylib

import (
	"log"

	"github.com/SUSE/telemetry/internal/pkg/datastore"
	"github.com/SUSE/telemetry/pkg/config"
)

type TelemetryCommon interface {
	// Setup the datastore for dataitems, bundles and reports
	setup(*config.Config) error

	// cleanup extractor contents, for testing support
	cleanup()

	// Get a count of telemetry data items
	DataItemCount() (int, error)

	// Get a count of telemetry bundles
	BundleCount() (int, error)

	// Get a count of telemetry reports
	ReportCount() (int, error)
}

type TelemetryCommonImpl struct {
	items   datastore.DataStorer
	bundles datastore.DataStorer
	reports datastore.DataStorer
}

func (t *TelemetryCommonImpl) setup(cfg *config.Config) (err error) {
	err = nil
	// create the telemetry data items data store if not already setup
	if t.items == nil {
		log.Printf("creating new datastore of type %q, params %q", cfg.ItemDS.Type, cfg.ItemDS.Path)
		t.items, err = datastore.NewDataStore(cfg.ItemDS.Type, cfg.ItemDS.Path)
		if err != nil {
			log.Printf("failed to create an items data store of type %q, params %q", cfg.ItemDS.Type, cfg.ItemDS.Path)
			return
		}
	}

	// create the telemetry bundle data store if not already setup
	if t.bundles == nil {
		t.bundles, err = datastore.NewDataStore(cfg.BundleDS.Type, cfg.BundleDS.Path)
		if err != nil {
			log.Printf("failed to create a bundle data store of type %q params %q", cfg.BundleDS.Type, cfg.BundleDS.Path)
			return
		}
	}

	// create the telemetry report data store if not already setup
	if t.reports == nil {
		t.reports, err = datastore.NewDataStore(cfg.ReportDS.Type, cfg.ReportDS.Path)
		if err != nil {
			log.Printf("failed to create a report data store of type %q params %q", cfg.ReportDS.Type, cfg.ReportDS.Path)
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
