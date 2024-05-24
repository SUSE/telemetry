package telemetrylib

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/SUSE/telemetry/pkg/config"
	_ "github.com/mattn/go-sqlite3"
)

// DatabaseStorer is an implementation for storing data in a database.
type DatabaseStore struct {
	Conn       *sql.DB
	Driver     string
	DataSource string
}

func NewDatabaseStore(dbConfig config.DBConfig) (ds *DatabaseStore, err error) {
	ds = &DatabaseStore{}

	switch dbConfig.Driver {
	case "sqlite3":
		if _, err := os.Stat(dbConfig.Params); os.IsNotExist(err) {
			dirPath := filepath.Dir(dbConfig.Params)
			os.MkdirAll(dirPath, 0700)

			file, err := os.Create(dbConfig.Params)
			if err != nil {
				log.Fatal(err.Error())
				return nil, err
			}
			file.Close()
			log.Println("created SQLite file", dbConfig.Params)
		}

		ds.Setup(dbConfig)
		ds.Connect()

	default:
		err = fmt.Errorf("unsupported database type %q", dbConfig.Driver)
		log.Print(err.Error())
		return nil, err
	}

	err = ds.EnsureTablesExist()
	if err != nil {
		log.Print(err.Error())
		return nil, err
	}
	return ds, nil
}

func (d DatabaseStore) String() string {
	return fmt.Sprintf("%p:%s:%s", d.Conn, d.Driver, d.DataSource)
}

func (d *DatabaseStore) Setup(dbcfg config.DBConfig) {
	d.Driver, d.DataSource = dbcfg.Driver, dbcfg.Params
}

func (d *DatabaseStore) Connect() (err error) {
	d.Conn, err = sql.Open(d.Driver, d.DataSource)
	if err != nil {
		log.Printf("Failed to connect to DB '%s:%s': %s", d.Driver, d.DataSource, err.Error())
	}

	return
}

func (d *DatabaseStore) EnsureTablesExist() (err error) {
	for name, columns := range dbTables {
		createCmd := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s %s", name, columns)
		_, err = d.Conn.Exec(createCmd)
		if err != nil {
			log.Printf("failed to create table %q: %s", name, err.Error())
			return
		}
	}

	return
}

// list of predefined tables
var dbTables = map[string]string{
	"items":   itemsColumns,
	"bundles": bundlesColumns,
	"reports": reportsColumns,
}

func (d *DatabaseStore) GetItemsWithNoBundleAssociation() (itemIDs []int64, dataitemRows []*TelemetryDataItemRow, err error) {
	rows, err := d.Conn.Query("SELECT * FROM items WHERE bundleId IS NULL")
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	for rows.Next() {
		var dataitemRow TelemetryDataItemRow

		if err := rows.Scan(
			&dataitemRow.Id,
			&dataitemRow.ItemId,
			&dataitemRow.ItemType,
			&dataitemRow.ItemTimestamp,
			&dataitemRow.ItemAnnotations,
			&dataitemRow.ItemData,
			&dataitemRow.ItemChecksum,
			&dataitemRow.BundleId); err != nil {
			log.Fatal(err)
		}
		dataitemRows = append(dataitemRows, &dataitemRow)
		itemIDs = append(itemIDs, dataitemRow.Id)
	}

	if err := rows.Err(); err != nil {
		log.Fatal(err)
	}

	return itemIDs, dataitemRows, err

}

func (d *DatabaseStore) GetBundlesWithNoReportAssociation() (bundleIDs []int64, bundleRows []*TelemetryBundleRow, err error) {
	rows, err := d.Conn.Query("SELECT * FROM bundles WHERE reportId IS NULL")
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	for rows.Next() {
		var bundleRow TelemetryBundleRow

		if err := rows.Scan(
			&bundleRow.Id,
			&bundleRow.BundleId,
			&bundleRow.BundleTimestamp,
			&bundleRow.BundleClientId,
			&bundleRow.BundleCustomerId,
			&bundleRow.BundleAnnotations,
			&bundleRow.BundleChecksum,
			&bundleRow.ReportId); err != nil {
			log.Fatal(err)
		}
		bundleRows = append(bundleRows, &bundleRow)
		bundleIDs = append(bundleIDs, bundleRow.Id)
	}

	if err := rows.Err(); err != nil {
		log.Fatal(err)
	}

	return bundleIDs, bundleRows, err

}

func (d *DatabaseStore) GetDataItemRowsInABundle(bundleId string) (itemRows []*TelemetryDataItemRow, err error) {
	//perform a join between the items table and the bundle table to filter the items by the bundle ID.
	rows, err := d.Conn.Query(`SELECT items.id, items.itemId, items.itemType, items.itemTimestamp, items.itemAnnotations, items.itemData, items.itemChecksum, items.bundleId FROM items JOIN bundles ON items.bundleId = bundles.id WHERE bundles.bundleId = ?`, bundleId)

	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	for rows.Next() {
		var dataitemRow TelemetryDataItemRow
		if err := rows.Scan(
			&dataitemRow.Id,
			&dataitemRow.ItemId,
			&dataitemRow.ItemType,
			&dataitemRow.ItemTimestamp,
			&dataitemRow.ItemAnnotations,
			&dataitemRow.ItemData,
			&dataitemRow.ItemChecksum,
			&dataitemRow.BundleId); err != nil {
			log.Fatal(err)
		}

		itemRows = append(itemRows, &dataitemRow)

	}

	if err := rows.Err(); err != nil {
		log.Fatal(err)
	}

	return itemRows, err

}

func (d *DatabaseStore) GetBundleRowsInAReport(reportId string) (bundleRows []*TelemetryBundleRow, err error) {
	//perform a join between the bundles table and the report table to filter the bundles by the report ID.
	rows, err := d.Conn.Query(`SELECT bundles.id, bundles.bundleId, bundles.bundleTimestamp, bundles.bundleClientId, bundles.bundleCustomerId, bundles.bundleAnnotations, bundles.bundleChecksum, bundles.reportId FROM bundles JOIN reports ON bundles.reportId = reports.id WHERE reports.reportId = ?`, reportId)

	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	for rows.Next() {
		var bundleRow TelemetryBundleRow

		if err := rows.Scan(
			&bundleRow.Id,
			&bundleRow.BundleId,
			&bundleRow.BundleTimestamp,
			&bundleRow.BundleClientId,
			&bundleRow.BundleCustomerId,
			&bundleRow.BundleAnnotations,
			&bundleRow.BundleChecksum,
			&bundleRow.ReportId); err != nil {
			log.Fatal(err)
		}
		bundleRows = append(bundleRows, &bundleRow)
	}

	if err := rows.Err(); err != nil {
		log.Fatal(err)
	}

	return bundleRows, err

}

func (d *DatabaseStore) GetReports() (reportIDs []int64, reportRows []*TelemetryReportRow, err error) {

	rows, err := d.Conn.Query("SELECT * FROM reports")
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	for rows.Next() {
		var reportRow TelemetryReportRow

		if err := rows.Scan(
			&reportRow.Id,
			&reportRow.ReportId,
			&reportRow.ReportTimestamp,
			&reportRow.ReportClientId,
			&reportRow.ReportAnnotations,
			&reportRow.ReportChecksum); err != nil {
			log.Fatal(err)
		}
		reportRows = append(reportRows, &reportRow)
		reportIDs = append(reportIDs, reportRow.Id)
	}

	if err := rows.Err(); err != nil {
		log.Fatal(err)
	}

	return reportIDs, reportRows, err

}

func (d *DatabaseStore) GetDataItemCount() (int, error) {
	var count int
	err := d.Conn.QueryRow("SELECT COUNT(*) FROM items WHERE bundleId IS NULL").Scan(&count)
	if err != nil {
		log.Fatal(err)
	}
	return count, err
}

func (d *DatabaseStore) GetBundleCount() (int, error) {
	var count int
	err := d.Conn.QueryRow("SELECT COUNT(*) FROM bundles WHERE reportId IS NULL").Scan(&count)
	if err != nil {
		log.Fatal(err)
	}
	return count, err
}

func (d *DatabaseStore) GetReportCount() (int, error) {
	var count int
	err := d.Conn.QueryRow("SELECT COUNT(*) FROM reports").Scan(&count)
	if err != nil {
		log.Fatal(err)
	}
	return count, err
}

// only for testing
func (d *DatabaseStore) dropTables() (err error) {

	for name := range dbTables {
		dropCmd := fmt.Sprintf("DROP TABLE IF EXISTS %s", name)

		_, err = d.Conn.Exec(dropCmd)
		if err != nil {
			log.Printf("failed to drop table %q: %s", name, err.Error())
			return
		}
	}

	return
}
