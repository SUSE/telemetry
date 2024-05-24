package telemetrylib

import (
	"database/sql"
	"log"
	"strings"

	"github.com/SUSE/telemetry/pkg/types"
	"github.com/google/uuid"
)

type TelemetryReport struct {
	Header           TelemetryReportHeader `json:"header"`
	TelemetryBundles []TelemetryBundle     `json:"telemetryBundles"`
	Footer           TelemetryReportFooter `json:"footer"`
}

func NewTelemetryReport(clientId int64, tags types.Tags) *TelemetryReport {
	tr := new(TelemetryReport)

	// fill in header fields
	tr.Header.ReportId = uuid.New().String()
	tr.Header.ReportTimeStamp = types.Now().String()
	tr.Header.ReportClientId = clientId
	for _, a := range tags {
		tr.Header.ReportAnnotations = append(tr.Header.ReportAnnotations, string(a))
	}

	// fill in footer
	tr.Footer.Checksum = "rchecksum" // TODO

	return tr
}

type TelemetryReportHeader struct {
	ReportId          string   `json:"reportId"`
	ReportTimeStamp   string   `json:"reportTimeStamp"`
	ReportClientId    int64    `json:"reportClientId"`
	ReportAnnotations []string `json:"reportAnnotations"`
}

type TelemetryReportFooter struct {
	Checksum string `json:"checksum"`
}

// Database
const reportsColumns = `(
	id INTEGER NOT NULL PRIMARY KEY,
	reportId VARCHAR(64) NOT NULL,
	reportTimestamp VARCHAR(32) NOT NULL,
	reportClientId INTEGER NOT NULL,
	reportAnnotations TEXT,
	reportChecksum VARCHAR(256),
	FOREIGN KEY (reportId) REFERENCES bundles(id)
)`

type TelemetryReportRow struct {
	Id                int64
	ReportId          string
	ReportTimestamp   string
	ReportClientId    int64
	ReportAnnotations string
	ReportChecksum    string
}

func NewTelemetryReportRow(clientId int64, tags types.Tags) (*TelemetryReportRow, error) {

	report := NewTelemetryReport(clientId, tags)
	reportRow := TelemetryReportRow{}
	reportRow.ReportId = report.Header.ReportId
	reportRow.ReportTimestamp = report.Header.ReportTimeStamp
	reportRow.ReportClientId = report.Header.ReportClientId
	reportRow.ReportAnnotations = strings.Join(report.Header.ReportAnnotations, ",")
	reportRow.ReportChecksum = report.Footer.Checksum

	return &reportRow, nil

}

func (r *TelemetryReportRow) Exists(db *sql.DB) bool {
	row := db.QueryRow(`SELECT id FROM reports WHERE reportId = ?`, r.ReportId)
	if err := row.Scan(&r.Id); err != nil {
		if err != sql.ErrNoRows {
			log.Printf("ERR: failed when checking for existence of report id %q: %s", r.Id, err.Error())
		}
		return false
	}
	return true
}

func (r *TelemetryReportRow) Insert(db *sql.DB, bundleIDs []int64) (reportId string, err error) {
	res, err := db.Exec(
		`INSERT INTO Reports(ReportId, ReportTimestamp, ReportClientId, ReportAnnotations, ReportChecksum) VALUES(?, ?, ?, ?, ?)`,
		r.ReportId, r.ReportTimestamp, r.ReportClientId, r.ReportAnnotations, r.ReportChecksum,
	)
	if err != nil {
		log.Printf("failed to add Report entry with ReportId %q: %s", r.ReportId, err.Error())
		return reportId, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		log.Printf("ERR: failed to retrieve id for inserted Report %q: %s", r.ReportId, err.Error())
		return reportId, err
	}
	r.Id = id

	// Update the reportId of the bundles
	for _, bundleID := range bundleIDs {
		_, err := db.Exec("UPDATE bundles SET ReportId = ? WHERE id = ?", r.Id, bundleID)
		if err != nil {
			log.Fatal(err)
		}
	}
	reportId = r.ReportId
	return
}

func (r *TelemetryReportRow) Delete(db *sql.DB) (err error) {
	_, err = db.Exec("DELETE FROM reports WHERE reportId = ?", r.ReportId)
	return
}
