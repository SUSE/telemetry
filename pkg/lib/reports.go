package telemetrylib

import (
	"database/sql"
	"fmt"
	"log/slog"
	"strings"

	"github.com/SUSE/telemetry/pkg/types"
	"github.com/SUSE/telemetry/pkg/utils"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

type TelemetryReport struct {
	// NOTE: omitempty option used in json tags to support generating test scenarios
	Header           TelemetryReportHeader `json:"header" validate:"required"`
	TelemetryBundles []TelemetryBundle     `json:"telemetryBundles,omitempty" validate:"required,gt=0,dive"`
	Footer           TelemetryReportFooter `json:"footer,omitempty" validate:"omitempty"`
}

func (tr *TelemetryReport) UpdateChecksum() (err error) {
	tr.Footer.Checksum, err = utils.GetMd5Hash(tr.TelemetryBundles)
	return
}

func (tr *TelemetryReport) Validate() (err error) {

	validate := validator.New(validator.WithRequiredStructEnabled())

	err = validate.Struct(tr)
	if err != nil {
		slog.Debug(
			"report struct validation failed",
			slog.String("err", err.Error()),
		)
		err = fmt.Errorf("report struct validation check failed: %w", err)
	}

	return
}

func (tr *TelemetryReport) VerifyChecksum() (err error) {
	err = tr.verifyBundleChecksums()
	if err != nil {
		return err
	}

	// report checksums optional, so skip if not specified
	if tr.Footer.Checksum == "" {
		return
	}

	checksum, err := utils.GetMd5Hash(&tr.TelemetryBundles)
	if err != nil {
		err = fmt.Errorf("failed to generate report checksum: %w", err)
	}

	if checksum != tr.Footer.Checksum {
		err = fmt.Errorf(
			"failed to validate report checksum: %q != %q",
			checksum,
			tr.Footer.Checksum,
		)
	}
	return
}

func (tr *TelemetryReport) verifyBundleChecksums() (err error) {
	for i, bundle := range tr.TelemetryBundles {
		err = bundle.VerifyChecksum()
		if err != nil {
			return fmt.Errorf(
				"failed to verify checksum for report bundle %d: %w",
				i,
				err,
			)
		}
	}
	return
}

func NewTelemetryReport(clientId string, tags types.Tags) (*TelemetryReport, error) {
	tr := new(TelemetryReport)

	// fill in header fields
	tr.Header.ReportId = uuid.New().String()
	tr.Header.ReportTimeStamp = types.Now().String()
	tr.Header.ReportClientId = clientId
	for _, a := range tags {
		tr.Header.ReportAnnotations = append(tr.Header.ReportAnnotations, string(a))
	}

	// update the checksum
	if err := tr.UpdateChecksum(); err != nil {
		return nil, err
	}

	return tr, nil
}

type TelemetryReportHeader struct {
	// NOTE: omitempty option used in json tags to support generating test scenarios
	ReportId          string   `json:"reportId,omitempty" validate:"required,uuid4"`
	ReportTimeStamp   string   `json:"reportTimeStamp" validate:"required"`
	ReportClientId    string   `json:"reportClientId,omitempty" validate:"required,uuid4"`
	ReportAnnotations []string `json:"reportAnnotations,omitempty"`
}

type TelemetryReportFooter struct {
	// NOTE: omitempty option used in json tags to support generating test scenarios
	Checksum string `json:"checksum,omitempty" validate:"omitempty,md5"`
}

// Database
const reportsColumns = `(
	id INTEGER NOT NULL PRIMARY KEY,
	reportId VARCHAR(64) NOT NULL,
	reportTimestamp VARCHAR(32) NOT NULL,
	reportClientId VARCHAR NOT NULL,
	reportAnnotations TEXT,
	reportChecksum VARCHAR(256)
)`

type TelemetryReportRow struct {
	Id                int64
	ReportId          string
	ReportTimestamp   string
	ReportClientId    string
	ReportAnnotations string
}

func NewTelemetryReportRow(clientId string, tags types.Tags) (*TelemetryReportRow, error) {

	report, err := NewTelemetryReport(clientId, tags)
	if err != nil {
		return nil, err
	}

	reportRow := TelemetryReportRow{}
	reportRow.ReportId = report.Header.ReportId
	reportRow.ReportTimestamp = report.Header.ReportTimeStamp
	reportRow.ReportClientId = report.Header.ReportClientId
	reportRow.ReportAnnotations = strings.Join(report.Header.ReportAnnotations, ",")

	return &reportRow, nil

}

func (r *TelemetryReportRow) Exists(db *sql.DB) bool {
	row := db.QueryRow(`SELECT id FROM reports WHERE reportId = ?`, r.ReportId)
	if err := row.Scan(&r.Id); err != nil {
		if err != sql.ErrNoRows {
			slog.Error(
				"failed when checking for existence of report",
				slog.Int64("reportId", r.Id),
				slog.String("err", err.Error()),
			)
		}
		return false
	}
	return true
}

func (r *TelemetryReportRow) Insert(db *sql.DB, bundleIDs []int64) (reportId string, err error) {
	res, err := db.Exec(
		`INSERT INTO Reports(ReportId, ReportTimestamp, ReportClientId, ReportAnnotations) VALUES(?, ?, ?, ?)`,
		r.ReportId, r.ReportTimestamp, r.ReportClientId, r.ReportAnnotations,
	)
	if err != nil {
		slog.Error(
			"failed to add Report entry",
			slog.String("reportId", r.ReportId),
			slog.String("err", err.Error()),
		)
		return reportId, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		slog.Error(
			"failed to retrieve id for inserted Report",
			slog.String("reportId", r.ReportId),
			slog.String("err", err.Error()),
		)
		return reportId, err
	}
	r.Id = id

	// Update the reportId of the bundles
	for _, bundleID := range bundleIDs {
		_, err := db.Exec("UPDATE bundles SET ReportId = ? WHERE id = ?", r.Id, bundleID)
		if err != nil {
			slog.Error(
				"Failed to update reportId in bundle",
				slog.Int64("bundleId", bundleID),
				slog.String("error", err.Error()),
			)
			return "", err
		}
	}
	reportId = r.ReportId
	return
}

func (r *TelemetryReportRow) Delete(db *sql.DB) (err error) {
	_, err = db.Exec("DELETE FROM reports WHERE reportId = ?", r.ReportId)
	return
}
