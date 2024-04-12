package telemetrylib

import (
	"encoding/json"
	"fmt"
	"log"

	_ "github.com/mattn/go-sqlite3"

	"github.com/SUSE/telemetry/internal/pkg/datastore"
	"github.com/SUSE/telemetry/pkg/config"
	"github.com/SUSE/telemetry/pkg/types"
	"github.com/google/uuid"
)

type TelemetryDataItem struct {
	Header        TelemetryDataItemHeader `json:"header"`
	TelemetryData map[string]interface{}  `json:"telemetryData"`
	Footer        TelemetryDataItemFooter `json:"footer"`
}

func NewTelemetryDataItem(telemetry types.TelemetryType, tags types.Tags, data map[string]interface{}) *TelemetryDataItem {

	tdi := new(TelemetryDataItem)

	// fill in header fields
	tdi.Header.TelemetryId = uuid.New().String()
	tdi.Header.TelemetryType = string(telemetry)
	tdi.Header.TelemetryTimeStamp = types.Now().String()
	for _, a := range tags {
		tdi.Header.TelemetryAnnotations = append(tdi.Header.TelemetryAnnotations, string(a))
	}

	// fill in body
	tdi.TelemetryData = data

	// fill in footer
	tdi.Footer.Checksum = "Calculate checksum" // TODO

	return tdi
}

func (tdi *TelemetryDataItem) Key() string {
	return tdi.Header.TelemetryId + "_" + tdi.Header.TelemetryTimeStamp
}

type TelemetryDataItemHeader struct {
	TelemetryId          string   `json:"telemetryId"`
	TelemetryTimeStamp   string   `json:"telemetryTimeStamp"`
	TelemetryType        string   `json:"telemetryType"`
	TelemetryAnnotations []string `json:"telemetryAnnotations"`
}

type TelemetryDataItemFooter struct {
	Checksum string `json:"checksum"`
}

type TelemetryBundle struct {
	Header             TelemetryBundleHeader `json:"header"`
	TelemetryDataItems []TelemetryDataItem   `json:"telemetryDataItems"`
	Footer             TelemetryBundleFooter `json:"footer"`
}

func NewTelemetryBundle(clientId, customerId string, tags types.Tags) *TelemetryBundle {
	tb := new(TelemetryBundle)

	// fill in header fields
	tb.Header.BundleId = uuid.New().String()
	tb.Header.BundleTimeStamp = types.Now().String()
	tb.Header.BundleClientId = clientId
	tb.Header.BundleCustomerId = customerId
	for _, a := range tags {
		tb.Header.BundleAnnotations = append(tb.Header.BundleAnnotations, string(a))
	}

	// fill in footer
	tb.Footer.Checksum = "Calculate checksum" // TODO

	return tb
}

func (tb *TelemetryBundle) Key() string {
	return tb.Header.BundleId + "_" + tb.Header.BundleTimeStamp
}

type TelemetryBundleHeader struct {
	BundleId          string   `json:"bundleId"`
	BundleTimeStamp   string   `json:"bundleTimeStamp"`
	BundleClientId    string   `json:"bundleClientId"`
	BundleCustomerId  string   `json:"buncleCustomerId"`
	BundleAnnotations []string `json:"bundleAnnotations"`
}

type TelemetryBundleFooter struct {
	Checksum string `json:"checksum"`
}

type TelemetryReport struct {
	Header           TelemetryReportHeader `json:"header"`
	TelemetryBundles []TelemetryBundle     `json:"telemetryBundles"`
	Footer           TelemetryReportFooter `json:"footer"`
}

func NewTelemetryReport(clientId, authToken string, tags types.Tags) *TelemetryReport {
	tr := new(TelemetryReport)

	// fill in header fields
	tr.Header.ReportId = uuid.New().String()
	tr.Header.ReportTimeStamp = types.Now().String()
	tr.Header.ReportClientId = clientId
	tr.Header.ReportAuthToken = authToken
	for _, a := range tags {
		tr.Header.ReportAnnotations = append(tr.Header.ReportAnnotations, string(a))
	}

	// fill in footer
	tr.Footer.Checksum = "Calculate checksum" // TODO

	return tr
}

func (tr *TelemetryReport) Key() string {
	return tr.Header.ReportId + "_" + tr.Header.ReportTimeStamp
}

type TelemetryReportHeader struct {
	ReportId          string        `json:"reportId"`
	ReportTimeStamp   string        `json:"reportTimeStamp"`
	ReportClientId    string        `json:"reportClientId"`
	ReportAuthToken   string        `json:"reportAuthToken"`
	ReportAnnotations []interface{} `json:"reportAnnotations"`
}

type TelemetryReportFooter struct {
	Checksum string `json:"checksum"`
}

// define a method to process jsonData as a byte[]
type TelemetryProcessor interface {
	// Add telemetry data
	AddData(
		telemetry types.TelemetryType,
		content []byte,
		tags types.Tags,
	) (*TelemetryDataItem, error)

	// Get a count of telemetry data items
	DataItemCount() (int, error)

	// Generate telemetry bundle
	GenerateBundle(
		clientId,
		customerId string,
		tags types.Tags,
	) (*TelemetryBundle, error)

	// Get a count of telemetry bundles
	BundleCount() (int, error)

	// Generate telemetry report
	GenerateReport(
		clientId,
		authToken string,
		tags types.Tags,
	) (*TelemetryReport, error)

	// Get a count of telemetry reports
	ReportCount() (int, error)

	// cleanup processor contents, for testing support
	cleanup()
}

// implements TelemetryDataItemProcessor interface.
type TelemetryProcessorImpl struct {
	cfg     *config.Config
	items   datastore.DataStorer
	bundles datastore.DataStorer
	reports datastore.DataStorer
}

func (p *TelemetryProcessorImpl) setup() (err error) {
	err = nil

	// create the telemetry data items data store if not already setup
	if p.items == nil {
		log.Printf("creating new datastore of type %q, params %q", p.cfg.ItemDS.Type, p.cfg.ItemDS.Path)
		p.items, err = datastore.NewDataStore(p.cfg.ItemDS.Type, p.cfg.ItemDS.Path)
		if err != nil {
			log.Printf("failed to create an items data store of type %q, params %q", p.cfg.ItemDS.Type, p.cfg.ItemDS.Path)
			return
		}
	}

	// create the telemetry bundle data store if not already setup
	if p.bundles == nil {
		p.bundles, err = datastore.NewDataStore(p.cfg.BundleDS.Type, p.cfg.BundleDS.Path)
		if err != nil {
			log.Printf("failed to create a bundle data store of type %q params %q", p.cfg.BundleDS.Type, p.cfg.BundleDS.Path)
			return
		}
	}

	// create the telemetry report data store if not already setup
	if p.reports == nil {
		p.reports, err = datastore.NewDataStore(p.cfg.ReportDS.Type, p.cfg.ReportDS.Path)
		if err != nil {
			log.Printf("failed to create a report data store of type %q params %q", p.cfg.ReportDS.Type, p.cfg.ReportDS.Path)
			return
		}
	}

	return
}

func (p *TelemetryProcessorImpl) cleanup() {
	iKeys, _ := p.items.List()
	for _, key := range iKeys {
		p.items.Delete(key)
	}

	bKeys, _ := p.bundles.List()
	for _, key := range bKeys {
		p.bundles.Delete(key)
	}

	rKeys, _ := p.reports.List()
	for _, key := range rKeys {
		p.reports.Delete(key)
	}
}

func NewTelemetryProcessor(cfg *config.Config) (TelemetryProcessor, error) {
	log.Printf("NewTelemetryProcessor(%+v)", cfg)
	p := TelemetryProcessorImpl{cfg: cfg}

	err := p.setup()

	return &p, err
}

func (p *TelemetryProcessorImpl) DataItemCount() (count int, err error) {
	keys, err := p.items.List()
	if err != nil {
		log.Printf("failed to retrieve list of keys for item store: %s", err.Error())
		return
	}

	count = len(keys)

	return
}

func (p *TelemetryProcessorImpl) BundleCount() (count int, err error) {
	keys, err := p.bundles.List()
	if err != nil {
		log.Printf("failed to retrieve list of keys for bundle store: %s", err.Error())
		return
	}

	count = len(keys)

	return
}

func (p *TelemetryProcessorImpl) ReportCount() (count int, err error) {
	keys, err := p.reports.List()
	if err != nil {
		log.Printf("failed to retrieve list of keys for report store: %s", err.Error())
		return
	}

	count = len(keys)

	return
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
	err = p.items.Add(iKey, jsonData)
	if err != nil {
		return nil, fmt.Errorf("failed to add the telemetry data item with telemetry id %q: %s", i.Header.TelemetryId, err.Error())
	}

	// set the returned item value
	item = i

	return
}

func (p *TelemetryProcessorImpl) GenerateBundle(clientId, customerId string, tags types.Tags) (bundle *TelemetryBundle, err error) {

	b := NewTelemetryBundle(clientId, customerId, tags)

	itemKeys, err := p.items.List()
	if err != nil {
		log.Printf("failed to retrieve list of keys from item store: %s", err.Error())
		return
	}

	items := make([]TelemetryDataItem, len(itemKeys))
	for i, key := range itemKeys {
		data, err := p.items.Get(key)
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
	err = p.bundles.Add(b.Key(), []byte(jsonData))
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
		err := p.items.Delete(key)
		if err != nil {
			log.Printf("failed to delete the data item %q: %s", key, err.Error())
		}
	}
}

func (p *TelemetryProcessorImpl) GenerateReport(clientId, authToken string, tags types.Tags) (report *TelemetryReport, err error) {

	rpt := NewTelemetryReport(clientId, authToken, tags)

	bundleKeys, err := p.bundles.List()
	if err != nil {
		log.Println("failed to retrieve the bundles", err)
		return
	}

	bundles := make([]TelemetryBundle, len(bundleKeys))

	for i, key := range bundleKeys {
		value, err := p.bundles.Get(key)
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
	err = p.reports.Add(rpt.Key(), []byte(jsonData))
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
		err := p.bundles.Delete(key)
		if err != nil {
			log.Printf("failed to delete bundle %q: %s", key, err.Error())
			continue
		}
	}
}
