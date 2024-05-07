package telemetrylib

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/SUSE/telemetry/pkg/config"
	"github.com/SUSE/telemetry/pkg/types"
)

type TelemetryProcessor interface {
	TelemetryCommon
	// Add telemetry data - a method to process jsonData as a byte[]
	AddData(
		telemetry types.TelemetryType,
		content []byte,
		tags types.Tags,
	) (*TelemetryDataItem, error)

	// Generate telemetry bundle
	GenerateBundle(
		clientId,
		customerId string,
		tags types.Tags,
	) (*TelemetryBundle, error)

	// Generate telemetry report
	GenerateReport(
		clientId,
		authToken string,
		tags types.Tags,
	) (*TelemetryReport, error)

	// Get telemetry data items
	GetDataItems() ([]TelemetryDataItem, error)

	// Get telemetry bundles
	GetBundles() ([]TelemetryBundle, error)

	// Get telemetry reports
	GetReports() ([]TelemetryReport, error)
}

// implements TelemetryProcessor interface.
type TelemetryProcessorImpl struct {
	t   TelemetryCommonImpl
	cfg *config.DataStoresConfig
}

func (p *TelemetryProcessorImpl) setup(*config.DataStoresConfig) (err error) {
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

func (p *TelemetryProcessorImpl) GetDataItems() (dataitems []TelemetryDataItem, err error) {
	return p.t.GetDataItems()
}

func (p *TelemetryProcessorImpl) DeleteDataItem(dataItem *TelemetryDataItem) (err error) {
	return p.t.DeleteDataItem(dataItem)
}

func (p *TelemetryProcessorImpl) GetBundles() (bundles []TelemetryBundle, err error) {
	return p.t.GetBundles()
}

func (p *TelemetryProcessorImpl) GetReports() (reports []TelemetryReport, err error) {
	return p.t.GetReports()
}

func (p *TelemetryProcessorImpl) DeleteReport(report *TelemetryReport) (err error) {
	return p.t.DeleteReport(report)
}

func NewTelemetryProcessor(cfg *config.DataStoresConfig) (TelemetryProcessor, error) {
	log.Printf("NewTelemetryProcessor(%+v)", cfg)
	p := TelemetryProcessorImpl{cfg: cfg}

	err := p.setup(cfg)

	return &p, err
}

func (p *TelemetryProcessorImpl) AddData(telemetry types.TelemetryType, marshaledData []byte, tags types.Tags) (item *TelemetryDataItem, err error) {

	var data map[string]interface{}

	err = json.Unmarshal([]byte(marshaledData), &data)
	if err != nil {
		return nil, fmt.Errorf("unable to unmarshal JSON: %s", err.Error())
	}

	i := NewTelemetryDataItem(telemetry, tags, data)

	jsonData, err := json.Marshal(i)
	if err != nil {
		log.Println("error marshalling JSON:", err)
		return
	}

	iKey := i.Key()
	err = p.t.items.Add(iKey, jsonData)
	if err != nil {
		return nil, fmt.Errorf("failed to add the telemetry data item with telemetry id %q: %s", i.Header.TelemetryId, err.Error())
	}

	// set the returned item value
	item = i

	return
}

func (p *TelemetryProcessorImpl) GenerateBundle(clientId, customerId string, tags types.Tags) (bundle *TelemetryBundle, err error) {

	b := NewTelemetryBundle(clientId, customerId, tags)

	itemKeys, err := p.t.items.List()
	if err != nil {
		log.Printf("failed to retrieve list of keys from item store: %s", err.Error())
		return
	}

	if len(itemKeys) < 1 {
		log.Print("no data items to bundle up, skipping")
		return
	}

	items := make([]TelemetryDataItem, len(itemKeys))
	for i, key := range itemKeys {
		data, err := p.t.items.Get(key)
		if err != nil {
			log.Printf("failed to retrieve key %q from item store: %s", key, err.Error())
			return nil, err
		}

		var item TelemetryDataItem
		err = json.Unmarshal(data, &item)
		if err != nil {
			log.Printf("failed to unmarshal data item %q: %s", key, err.Error())
			return nil, err
		}

		items[i] = item
	}

	b.TelemetryDataItems = items
	b.Footer.Checksum = "calculatechecksum" //TO DO

	// Create the bundle
	jsonData, err := json.Marshal(b)
	if err != nil {
		log.Println("error marshalling JSON:", err)
		return
	}
	bKey := b.Key()
	log.Printf("generating a bundle with ID %s\n", bKey)
	err = p.t.bundles.Add(b.Key(), []byte(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to add the bundle with bundle id %q: %s", b.Header.BundleId, err.Error())
	}

	log.Printf("generated a bundle with ID %s successfully\n", b.Header.BundleId)

	// Delete the processed dataitems
	p.deleteAlreadyProcessedDataItems(itemKeys)

	// set the returned bundle value
	bundle = b

	return
}

func (p *TelemetryProcessorImpl) deleteAlreadyProcessedDataItems(itemKeys []string) {
	for _, key := range itemKeys {
		err := p.t.items.Delete(key)
		if err != nil {
			log.Printf("failed to delete the data item %q: %s", key, err.Error())
		}
	}
}

func (p *TelemetryProcessorImpl) GenerateReport(clientId, authToken string, tags types.Tags) (report *TelemetryReport, err error) {

	rpt := NewTelemetryReport(clientId, authToken, tags)

	bundleKeys, err := p.t.bundles.List()
	if err != nil {
		log.Println("failed to retrieve the bundles", err)
		return
	}

	if len(bundleKeys) < 1 {
		log.Print("no bundles available to add to reports, skipping")
		return
	}

	bundles := make([]TelemetryBundle, len(bundleKeys))

	for i, key := range bundleKeys {
		value, err := p.t.bundles.Get(key)
		if err != nil {
			log.Printf("unable to marshal the data item %q: %s", key, err.Error())
			return nil, err
		}

		var bundle TelemetryBundle
		err = json.Unmarshal(value, &bundle)
		if err != nil {
			log.Printf("unable to marshal the data item %q: %s", key, err.Error())
			return nil, err
		}

		bundles[i] = bundle
	}

	rpt.TelemetryBundles = bundles

	jsonData, err := json.Marshal(rpt)
	if err != nil {
		log.Println("error marshalling JSON:", err)
		return
	}

	rKey := rpt.Key()
	log.Printf("generating a report with ID %s\n", rKey)
	err = p.t.reports.Add(rpt.Key(), []byte(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to add the report with report id %q: %s", rpt.Header.ReportId, err.Error())
	}

	log.Printf("generated a report with ID %s successfully\n", rpt.Header.ReportId)

	// Delete the processed bundles
	p.deleteAlreadyProcessedBundles(bundleKeys)

	return rpt, nil
}

func (p *TelemetryProcessorImpl) deleteAlreadyProcessedBundles(bundleKeys []string) {
	for _, key := range bundleKeys {
		err := p.t.bundles.Delete(key)
		if err != nil {
			log.Printf("failed to delete bundle %q: %s", key, err.Error())
			continue
		}
	}
}
