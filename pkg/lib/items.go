package telemetrylib

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/SUSE/telemetry/pkg/types"
	"github.com/SUSE/telemetry/pkg/utils"
	"github.com/google/uuid"
)

type TelemetryDataItem struct {
	// NOTE: omitempty option used in json tags to support generating test scenarios
	Header        TelemetryDataItemHeader `json:"header"  validate:"required"`
	TelemetryData json.RawMessage         `json:"telemetryData"  validate:"required,dive"`
	Footer        TelemetryDataItemFooter `json:"footer,omitempty" validate:"omitempty"`
}

func (tdi *TelemetryDataItem) UpdateChecksum() (err error) {
	tdi.Footer.Checksum, err = utils.GetMd5Hash(&tdi.TelemetryData)
	if err != nil {
		err = fmt.Errorf("failed to generate data item checksum: %w", err)
	}
	return
}

func (tdi *TelemetryDataItem) VerifyChecksum() (err error) {
	// data item checksums optional, so skip if not specified
	if tdi.Footer.Checksum == "" {
		return
	}

	checksum, err := utils.GetMd5Hash(&tdi.TelemetryData)
	if err != nil {
		err = fmt.Errorf("failed to generate data item checksum: %w", err)
	}

	if checksum != tdi.Footer.Checksum {
		err = fmt.Errorf(
			"failed to validate data item checksum: %q != %q",
			checksum,
			tdi.Footer.Checksum,
		)
	}
	return
}

func NewTelemetryDataItem(telemetry types.TelemetryType, tags types.Tags, content *types.TelemetryBlob) (*TelemetryDataItem, error) {
	tdi := new(TelemetryDataItem)

	// fill in header fields
	tdi.Header.TelemetryId = uuid.New().String()
	tdi.Header.TelemetryType = string(telemetry)
	tdi.Header.TelemetryTimeStamp = types.Now().String()
	for _, a := range tags {
		tdi.Header.TelemetryAnnotations = append(tdi.Header.TelemetryAnnotations, string(a))
	}

	// fill in body
	tdi.TelemetryData = content.Bytes()

	// update the checksum
	if err := tdi.UpdateChecksum(); err != nil {
		return nil, err
	}

	return tdi, nil
}

type TelemetryDataItemHeader struct {
	TelemetryId          string   `json:"telemetryId"  validate:"required,uuid|uuid_rfc4122"`
	TelemetryTimeStamp   string   `json:"telemetryTimeStamp"  validate:"required"`
	TelemetryType        string   `json:"telemetryType"  validate:"required,min=5"`
	TelemetryAnnotations []string `json:"telemetryAnnotations,omitempty"`
}

type TelemetryDataItemFooter struct {
	Checksum string `json:"checksum"  validate:"required"`
}

//Database Mapping

const itemsColumns = `(
	id INTEGER NOT NULL PRIMARY KEY,
	itemId VARCHAR(64) NOT NULL,
	itemType VARCHAR(64) NOT NULL,
	itemTimestamp VARCHAR(32) NOT NULL,
	itemAnnotations TEXT NULL,
	itemData BLOB NOT NULL,
	itemChecksum VARCHAR(256),
	compression VARCHAR NULL,
	bundleId INTEGER NULL,
	CONSTRAINT items_bundleId
	  FOREIGN KEY (bundleId)
		REFERENCES bundles(id)
	  ON DELETE CASCADE
)`

type TelemetryDataItemRow struct {
	Id              int64
	ItemId          string
	ItemType        string
	ItemTimestamp   string
	ItemAnnotations string
	ItemData        []byte
	ItemChecksum    string
	Compression     sql.NullString
	BundleId        sql.NullInt64
}

func NewTelemetryDataItemRow(telemetry types.TelemetryType, tags types.Tags, content *types.TelemetryBlob) (itemRow *TelemetryDataItemRow, err error) {

	item, err := NewTelemetryDataItem(telemetry, tags, content)
	if err != nil {
		return
	}

	itemRow = new(TelemetryDataItemRow)
	itemRow.ItemId = item.Header.TelemetryId
	itemRow.ItemType = item.Header.TelemetryType
	itemRow.ItemTimestamp = item.Header.TelemetryTimeStamp
	itemRow.ItemAnnotations = strings.Join(item.Header.TelemetryAnnotations, ",")
	itemRow.ItemData = content.Bytes()
	itemRow.ItemChecksum = item.Footer.Checksum

	return
}

func (t *TelemetryDataItemRow) Exists(db *sql.DB) bool {
	row := db.QueryRow(`SELECT id FROM items WHERE itemId = ? AND itemType = ?`, t.ItemId, t.ItemType)
	if err := row.Scan(&t.Id); err != nil {
		if err != sql.ErrNoRows {
			slog.Error(
				"failed when checking for existence of telemetry data",
				slog.Int64("id", t.Id),
				slog.String("type", t.ItemType),
				slog.String("err", err.Error()),
			)
		}
		return false
	}
	return true
}

func (t *TelemetryDataItemRow) Insert(db *sql.DB) (err error) {
	itemData, compression, err := utils.CompressWhenNeeded(t.ItemData)
	if err != nil {
		return
	}
	res, err := db.Exec(
		`INSERT INTO items(ItemId, ItemType, ItemTimestamp, ItemAnnotations, ItemData, ItemChecksum, Compression) VALUES(?, ?, ?, ?, ?, ?, ?)`,
		t.ItemId, t.ItemType, t.ItemTimestamp, t.ItemAnnotations, itemData, t.ItemChecksum, compression,
	)
	if err != nil {
		slog.Error(
			"failed to add item entry",
			slog.String("itemId", t.ItemId),
			slog.String("err", err.Error()),
		)
		return
	}
	id, err := res.LastInsertId()
	if err != nil {
		slog.Error(
			"failed to retrieve id for inserted item",
			slog.String("itemId", t.ItemId),
			slog.String("err", err.Error()),
		)
		return
	}
	t.Id = id
	t.BundleId = sql.NullInt64{Int64: 0, Valid: false}

	return
}

func (t *TelemetryDataItemRow) Delete(db *sql.DB) (err error) {
	_, err = db.Exec("DELETE FROM items WHERE id = ?", t.Id)
	return
}
