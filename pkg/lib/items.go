package telemetrylib

import (
	"database/sql"
	"fmt"
	"log"
	"strings"

	"github.com/SUSE/telemetry/pkg/types"
	"github.com/SUSE/telemetry/pkg/utils"
	"github.com/google/uuid"
)

type TelemetryDataItem struct {
	Header        TelemetryDataItemHeader `json:"header"`
	TelemetryData map[string]interface{}  `json:"telemetryData"`
	Footer        TelemetryDataItemFooter `json:"footer"`
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
	TelemetryId          string   `json:"telemetryId"`
	TelemetryTimeStamp   string   `json:"telemetryTimeStamp"`
	TelemetryType        string   `json:"telemetryType"`
	TelemetryAnnotations []string `json:"telemetryAnnotations"`
}

type TelemetryDataItemFooter struct {
	Checksum string `json:"checksum"`
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
	FOREIGN KEY (bundleId) REFERENCES bundles(id)
)`

type TelemetryDataItemRow struct {
	Id              int64
	ItemId          string
	ItemType        string
	ItemTimestamp   string
	ItemAnnotations string
	ItemData        string
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
	dataItemRow.ItemData = string(marshaledData)
	dataItemRow.ItemChecksum = item.Footer.Checksum

	return dataItemRow, nil

}

func (t *TelemetryDataItemRow) Exists(db *sql.DB) bool {
	row := db.QueryRow(`SELECT id FROM items WHERE telemetryId = ? AND telemetryType = ?`, t.ItemId, t.ItemType)
	if err := row.Scan(&t.Id); err != nil {
		if err != sql.ErrNoRows {
			log.Printf("ERR: failed when checking for existence of telemetry data id %q, type %q: %s", t.Id, t.ItemType, err.Error())
		}
		return false
	}
	return true
}

func (t *TelemetryDataItemRow) Insert(db *sql.DB) (err error) {
	res, err := db.Exec(
		`INSERT INTO items(ItemId, ItemType, ItemTimestamp, ItemAnnotations, ItemData, ItemChecksum, BundleId) VALUES(?, ?, ?, ?, ?, ?, NULL)`,
		t.ItemId, t.ItemType, t.ItemTimestamp, t.ItemAnnotations, fmt.Sprint(t.ItemData), t.ItemChecksum,
	)
	if err != nil {
		log.Printf("failed to add telemetryData entry with telemetryId %q: %s", t.ItemId, err.Error())
		return err
	}
	id, err := res.LastInsertId()
	if err != nil {
		log.Printf("ERR: failed to retrieve id for inserted telemetryData %q: %s", t.ItemId, err.Error())
		return err
	}
	t.Id = id
	t.BundleId = sql.NullInt64{Int64: 0, Valid: false}

	return
}

func (t *TelemetryDataItemRow) Delete(db *sql.DB) (err error) {
	_, err = db.Exec("DELETE FROM items WHERE id = ?", t.Id)
	return
}
