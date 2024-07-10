package telemetrylib

import (
	"database/sql"
	"fmt"
	"log/slog"
	"strings"

	"github.com/SUSE/telemetry/pkg/types"
	"github.com/SUSE/telemetry/pkg/utils"
	"github.com/google/uuid"
)

type TelemetryDataItem struct {
	Header        TelemetryDataItemHeader `json:"header"  validate:"required"`
	TelemetryData map[string]interface{}  `json:"telemetryData"  validate:"required,dive"`
	Footer        TelemetryDataItemFooter `json:"footer" validate:"required"`
}

// func NewTelemetryDataItem(telemetry types.TelemetryType, tags types.Tags, data map[string]interface{}) *TelemetryDataItem {
func NewTelemetryDataItem(telemetry types.TelemetryType, tags types.Tags, marshaledData []byte) (*TelemetryDataItem, error) {
	data, err := utils.DeserializeMap(string(marshaledData))
	if err != nil {
		return nil, fmt.Errorf("unable to unmarshal JSON: %s", err.Error())
	}

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
	tdi.Footer.Checksum = "ichecksum" // TODO

	return tdi, nil
}

type TelemetryDataItemHeader struct {
	TelemetryId          string   `json:"telemetryId"  validate:"required"`
	TelemetryTimeStamp   string   `json:"telemetryTimeStamp"  validate:"required"`
	TelemetryType        string   `json:"telemetryType"  validate:"required"`
	TelemetryAnnotations []string `json:"telemetryAnnotations"`
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
	BundleId        sql.NullInt64
}

func NewTelemetryDataItemRow(telemetry types.TelemetryType, tags types.Tags, marshaledData []byte) (*TelemetryDataItemRow, error) {

	item, err := NewTelemetryDataItem(telemetry, tags, marshaledData)
	if err != nil {
		return nil, fmt.Errorf("unable to create a new telemetry data item: %s", err.Error())
	}

	dataItemRow := new(TelemetryDataItemRow)
	dataItemRow.ItemId = item.Header.TelemetryId
	dataItemRow.ItemType = item.Header.TelemetryType
	dataItemRow.ItemTimestamp = item.Header.TelemetryTimeStamp
	dataItemRow.ItemAnnotations = strings.Join(item.Header.TelemetryAnnotations, ",")
	dataItemRow.ItemData = marshaledData
	dataItemRow.ItemChecksum = item.Footer.Checksum

	return dataItemRow, nil

}

func (t *TelemetryDataItemRow) Exists(db *sql.DB) bool {
	row := db.QueryRow(`SELECT id FROM items WHERE telemetryId = ? AND telemetryType = ?`, t.ItemId, t.ItemType)
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
	compressedItemData, err := utils.CompressGZIP(t.ItemData)
	if err != nil {
		return
	}
	res, err := db.Exec(
		`INSERT INTO items(ItemId, ItemType, ItemTimestamp, ItemAnnotations, ItemData, ItemChecksum, BundleId) VALUES(?, ?, ?, ?, ?, ?, NULL)`,
		t.ItemId, t.ItemType, t.ItemTimestamp, t.ItemAnnotations, compressedItemData, t.ItemChecksum,
	)
	if err != nil {
		slog.Error(
			"failed to add telemetryData entry with telemetryId",
			slog.Int64("id", t.Id),
			slog.String("err", err.Error()),
		)
		return
	}
	id, err := res.LastInsertId()
	if err != nil {
		slog.Error(
			"failed to retrieve id for inserted telemetryData",
			slog.Int64("id", t.Id),
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
