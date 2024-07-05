package telemetrylib

import (
	"database/sql"
	"fmt"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/SUSE/telemetry/pkg/config"
	"github.com/SUSE/telemetry/pkg/utils"
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
		dbPath, opts, optsFound := strings.Cut(dbConfig.Params, "?")

		// ensure that the path to the DB file exists and that we can access it
		if _, err := os.Stat(dbPath); os.IsNotExist(err) {
			dirPath := filepath.Dir(dbPath)
			os.MkdirAll(dirPath, 0700)

			// create file if it doesn't already exist, without truncating it if it does.
			file, err := os.OpenFile(dbPath, os.O_RDONLY|os.O_CREATE, 0600)
			if err != nil {
				log.Fatal(err.Error())
				return nil, err
			}
			file.Close()
			slog.Info("created SQLite file", slog.String("filePath", dbPath))
		}

		// exsure foreign_keys and journal_mode options are specified
		if !optsFound {
			opts = ""
		}
		extraOpts := []string{}
		if !strings.Contains(opts, "_foreign_keys=") {
			extraOpts = append(extraOpts, "_foreign_keys=on")
		}
		if !strings.Contains(opts, "_journal_mode=") {
			extraOpts = append(extraOpts, "_journal_mode=WAL")
		}
		if len(extraOpts) > 0 {
			if len(opts) > 0 {
				opts += "&"
			}
			opts += strings.Join(extraOpts, "&")
		}

		dbConfig.Params = dbPath + "?" + opts

		ds.Setup(dbConfig)
		ds.Connect()

	default:
		slog.Error("unsupported database type", slog.String("dbDriver", dbConfig.Driver))
		return nil, err
	}

	err = ds.EnsureTablesExist()
	if err != nil {
		slog.Error("databaseStora error", slog.String("err", err.Error()))
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
		slog.Error(
			"failed to connect to DB",
			slog.String("db", d.Driver),
			slog.String("dataSource", d.DataSource),
			slog.String("err", err.Error()),
		)
	}

	return
}

func (d *DatabaseStore) EnsureTablesExist() (err error) {
	for name, columns := range dbTables {
		createCmd := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s %s", name, columns)
		_, err = d.Conn.Exec(createCmd)
		if err != nil {
			slog.Error(
				"failed to create table",
				slog.String("table", name),
				slog.String("err", err.Error()),
			)
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

func genSqlPopulateQuery(table string, fields []string, matchField string, inputValues []any) (query string, outputValues []any) {
	query = `SELECT ` + strings.Join(fields, ", ") + ` FROM ` + table
	outputValues = inputValues

	numValues := len(inputValues)
	if numValues > 0 {
		query += ` WHERE ` + matchField
		switch {
		case inputValues[0] == "NULL":
			query += " IS NULL"
			// clear the
			outputValues = []any{}
		default:
			query += " IN (?" + strings.Repeat(", ?", numValues-1) + ")"
		}
	}

	return
}

func genSqlCountQuery(table, countField, matchField string, inputValues []any) (query string, outputValues []any) {
	query = `SELECT COUNT(` + countField + `) FROM ` + table
	outputValues = inputValues

	numValues := len(inputValues)
	if numValues > 0 {
		query += ` WHERE ` + matchField
		switch {
		case inputValues[0] == "NULL":
			query += " IS NULL"
			// clear the
			outputValues = []any{}
		default:
			query += " IN (?" + strings.Repeat(", ?", numValues-1) + ")"
		}
	}

	return
}

func (d *DatabaseStore) GetItems(bundleIds ...any) (itemRowIds []int64, itemRows []*TelemetryDataItemRow, err error) {
	// generate the SQL populate query statement for the items table
	query, queryBundleIds := genSqlPopulateQuery(
		"items",
		[]string{"id", "itemId", "itemType", "itemTimestamp", "itemAnnotations", "itemData", "itemChecksum", "bundleId"},
		"bundleId",
		bundleIds,
	)

	// NOTE: Query() extra args must be of type any hence queryIds is type []any
	rows, err := d.Conn.Query(query, queryBundleIds...)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	for rows.Next() {
		var itemRow TelemetryDataItemRow

		if err := rows.Scan(
			&itemRow.Id,
			&itemRow.ItemId,
			&itemRow.ItemType,
			&itemRow.ItemTimestamp,
			&itemRow.ItemAnnotations,
			&itemRow.ItemData,
			&itemRow.ItemChecksum,
			&itemRow.BundleId); err != nil {
			log.Fatal(err)
		}

		// ItemData is stored as compressed data
		decompressedItemData, err := utils.DecompressGZIP(itemRow.ItemData)
		if err != nil {
			log.Fatal(err)
		}

		itemRow.ItemData = decompressedItemData
		itemRows = append(itemRows, &itemRow)
		itemRowIds = append(itemRowIds, itemRow.Id)
	}

	if err := rows.Err(); err != nil {
		log.Fatal(err)
	}

	return itemRowIds, itemRows, err
}

func (d *DatabaseStore) GetBundles(reportIds ...any) (bundleRowIds []int64, bundleRows []*TelemetryBundleRow, err error) {
	// generate the SQL populate query statement for the bundles table
	query, queryBundleIds := genSqlPopulateQuery(
		"bundles",
		[]string{"id", "bundleId", "bundleTimestamp", "bundleClientId", "bundleCustomerId", "bundleAnnotations", "bundleChecksum", "reportId"},
		"reportId",
		reportIds,
	)

	// NOTE: Query() extra args must be of type any hence queryIds is type []any
	rows, err := d.Conn.Query(query, queryBundleIds...)
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
		bundleRowIds = append(bundleRowIds, bundleRow.Id)
	}

	if err := rows.Err(); err != nil {
		log.Fatal(err)
	}

	return bundleRowIds, bundleRows, err

}

func (d *DatabaseStore) GetReports(ids ...any) (reportRowIds []int64, reportRows []*TelemetryReportRow, err error) {
	// generate the SQL populate query statement for the reports table
	query, queryIds := genSqlPopulateQuery(
		"reports",
		[]string{"id", "reportId", "reportTimestamp", "reportClientId", "reportAnnotations", "reportChecksum"},
		"id",
		ids,
	)

	// NOTE: Query() extra args must be of type any hence queryIds is type []any
	rows, err := d.Conn.Query(query, queryIds...)
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
		reportRowIds = append(reportRowIds, reportRow.Id)
	}

	if err := rows.Err(); err != nil {
		log.Fatal(err)
	}

	return reportRowIds, reportRows, err
}

func (d *DatabaseStore) GetItemCount(bundleIds ...any) (int, error) {
	var count int
	// generate the SQL count query statement for the items table
	query, queryIds := genSqlCountQuery(
		"items",
		"id",
		"bundleId",
		bundleIds,
	)
	// NOTE: Query() extra args must be of type any hence queryIds is type []any
	err := d.Conn.QueryRow(query, queryIds...).Scan(&count)
	if err != nil {
		log.Fatal(err)
	}
	return count, err
}

func (d *DatabaseStore) GetBundleCount(reportIds ...any) (int, error) {
	var count int
	// generate the SQL count query statement for the bundles table
	query, queryIds := genSqlCountQuery(
		"bundles",
		"id",
		"reportId",
		reportIds,
	)
	// NOTE: Query() extra args must be of type any hence queryIds is type []any
	err := d.Conn.QueryRow(query, queryIds...).Scan(&count)
	if err != nil {
		log.Fatal(err)
	}
	return count, err
}

func (d *DatabaseStore) GetReportCount(ids ...any) (int, error) {
	var count int
	// generate the SQL count query statement for the reports table
	query, queryIds := genSqlCountQuery(
		"reports",
		"id",
		"id",
		ids,
	)
	// NOTE: Query() extra args must be of type any hence queryIds is type []any
	err := d.Conn.QueryRow(query, queryIds...).Scan(&count)
	if err != nil {
		log.Fatal(err)
	}
	return count, err
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

// only for testing
func (d *DatabaseStore) dropTables() (err error) {

	for name := range dbTables {
		dropCmd := fmt.Sprintf("DROP TABLE IF EXISTS %s", name)

		_, err = d.Conn.Exec(dropCmd)
		if err != nil {
			slog.Error(
				"failed to drop table",
				slog.String("table", name),
				slog.String("err", err.Error()),
			)
			return
		}
	}

	return
}
