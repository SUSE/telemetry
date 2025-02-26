package telemetrylib

import (
	"database/sql"
	"log/slog"
	"strings"

	"github.com/SUSE/telemetry/pkg/types"
	"github.com/google/uuid"
)

type TelemetryBundle struct {
	Header             TelemetryBundleHeader `json:"header" validate:"required"`
	TelemetryDataItems []TelemetryDataItem   `json:"telemetryDataItems" validate:"required,gt=0,dive"`
	Footer             TelemetryBundleFooter `json:"footer" validate:"required"`
}

func NewTelemetryBundle(clientId string, customerId string, tags types.Tags) *TelemetryBundle {
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
	tb.Footer.Checksum = "bchecksum" // TODO

	return tb
}

type TelemetryBundleHeader struct {
	BundleId          string   `json:"bundleId" validate:"required"`
	BundleTimeStamp   string   `json:"bundleTimeStamp" validate:"required"`
	BundleClientId    string   `json:"bundleClientId" validate:"required"`
	BundleCustomerId  string   `json:"bundleCustomerId" validate:"required"`
	BundleAnnotations []string `json:"bundleAnnotations"`
}

type TelemetryBundleFooter struct {
	Checksum string `json:"checksum" validate:"required"`
}

//Database Mapping

const bundlesColumns = `(
	id INTEGER NOT NULL PRIMARY KEY,
	bundleId VARCHAR(64) NOT NULL,
	bundleTimestamp VARCHAR(32) NOT NULL,
	bundleClientId VARCHAR NOT NULL,
	bundleCustomerId VARCHAR(64) NOT NULL,
	bundleAnnotations TEXT,
	bundleChecksum VARCHAR(256),
	reportId  INTEGER NULL,
	CONSTRAINT bundles_reportId
	  FOREIGN KEY (reportId)
		REFERENCES reports(id)
	  ON DELETE CASCADE
)`

type TelemetryBundleRow struct {
	Id                int64
	BundleId          string
	BundleTimestamp   string
	BundleClientId    string
	BundleCustomerId  string
	BundleAnnotations string
	BundleChecksum    string
	ReportId          sql.NullInt64
}

func NewTelemetryBundleRow(clientId string, customerId string, tags types.Tags) (*TelemetryBundleRow, error) {

	bundle := NewTelemetryBundle(clientId, customerId, tags)
	bundleRow := new(TelemetryBundleRow)

	bundleRow.BundleId = bundle.Header.BundleId
	bundleRow.BundleTimestamp = bundle.Header.BundleTimeStamp
	bundleRow.BundleClientId = bundle.Header.BundleClientId
	bundleRow.BundleCustomerId = bundle.Header.BundleCustomerId

	bundleRow.BundleAnnotations = strings.Join(bundle.Header.BundleAnnotations, ",")
	bundleRow.BundleChecksum = bundle.Footer.Checksum

	return bundleRow, nil

}

func (b *TelemetryBundleRow) Exists(db *sql.DB) bool {
	row := db.QueryRow(`SELECT id FROM bundles WHERE bundleId = ?`, b.BundleId)
	if err := row.Scan(&b.Id); err != nil {
		if err != sql.ErrNoRows {
			slog.Error(
				"failed to check bundle id existence",
				slog.Int64("bundleId", b.Id),
				slog.String("err", err.Error()),
			)
		}
		return false
	}
	return true
}

func (b *TelemetryBundleRow) Insert(db *sql.DB, itemIDs []int64) (bundleId string, err error) {
	res, err := db.Exec(
		`INSERT INTO bundles(BundleId, BundleTimestamp, BundleClientId, BundleCustomerId, BundleAnnotations, BundleChecksum, reportId) VALUES(?, ?, ?, ?, ?, ?, NULL)`,
		b.BundleId, b.BundleTimestamp, b.BundleClientId, b.BundleCustomerId, b.BundleAnnotations, b.BundleChecksum,
	)
	if err != nil {
		slog.Error(
			"failed to add bundle entry with bundleId",
			slog.String("bundleId", b.BundleId),
			slog.String("err", err.Error()),
		)
		return bundleId, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		slog.Error(
			"failed to retrieve id for inserted Bundle",
			slog.String("bundleId", b.BundleId),
			slog.String("err", err.Error()),
		)
		return bundleId, err
	}
	b.Id = id

	b.ReportId = sql.NullInt64{Int64: 0, Valid: false}

	// Update the bundleId of the items
	for _, itemID := range itemIDs {
		_, err := db.Exec("UPDATE items SET bundleId = ? WHERE id = ?", b.Id, itemID)
		if err != nil {
			slog.Error(
				"Failed to update bundleId in item",
				slog.Int64("itemId", itemID),
				slog.String("error", err.Error()),
			)
			return "", err
		}
	}

	bundleId = b.BundleId

	return
}

func (b *TelemetryBundleRow) Delete(db *sql.DB) (err error) {
	_, err = db.Exec("DELETE FROM bundles WHERE bundleId = ?", b.BundleId)
	return
}
