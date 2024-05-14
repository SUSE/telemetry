package telemetrylib

import (
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

func NewTelemetryBundle(clientId int64, customerId string, tags types.Tags) *TelemetryBundle {
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
	BundleClientId    int64    `json:"bundleClientId"`
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
