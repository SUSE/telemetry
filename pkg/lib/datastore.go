package telemetrylib

import (
	"database/sql"
	"fmt"
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
	persistent bool
}

func NewDatabaseStore(dbConfig config.DBConfig) (ds *DatabaseStore, err error) {
	ds = &DatabaseStore{}

	switch dbConfig.Driver {
	case "sqlite3":
		dbPath, opts, optsFound := strings.Cut(dbConfig.Params, "?")

		if !strings.Contains(dbPath, `:memory:`) {
			if _, err := os.Stat(dbPath); os.IsNotExist(err) {
				// ensure that the path to the DB file exists and that we can access it
				dirPath := filepath.Dir(dbPath)
				os.MkdirAll(dirPath, 0700)

				// create file if it doesn't already exist, without truncating it if it does.
				file, err := os.OpenFile(dbPath, os.O_RDONLY|os.O_CREATE, 0600)
				if err != nil {
					slog.Error(
						"Failed to open(O_CREATE) SQLite file",
						slog.String("dbPath", dbPath),
						slog.String("error", err.Error()),
					)
					return nil, err
				}
				file.Close()
				slog.Debug(
					"created SQLite file",
					slog.String("dbPath", dbPath),
				)
			}

			// file backed so can be persistent
			ds.persistent = true
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
	var persistent string

	if d.persistent {
		persistent = ",persistent"
	}

	return fmt.Sprintf("%p<%s,%s,%s>", d.Conn, d.Driver, d.DataSource, persistent)
}

func (d DatabaseStore) Persistent() bool {
	return d.persistent
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
		[]string{"id", "itemId", "itemType", "itemTimestamp", "itemAnnotations", "itemData", "itemChecksum", "compression", "bundleId"},
		"bundleId",
		bundleIds,
	)

	// NOTE: Query() extra args must be of type any hence queryIds is type []any
	rows, err := d.Conn.Query(query, queryBundleIds...)
	if err != nil {
		slog.Error(
			"Failed to retrieve items with specified bundleIds",
			slog.Any("bundleIds", bundleIds),
			slog.String("error", err.Error()),
		)
		return
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
			&itemRow.Compression,
			&itemRow.BundleId); err != nil {
			slog.Error(
				"Failed to scan item row",
				slog.String("error", err.Error()),
			)
			return nil, nil, err
		}

		// ItemData can be stored as compressed data
		itemRow.ItemData, err = utils.DecompressWhenNeeded(itemRow.ItemData, itemRow.Compression)
		if err != nil {
			slog.Error(
				"Failed to decompress item data",
				slog.String("itemId", itemRow.ItemId),
				slog.String("error", err.Error()),
			)
			return nil, nil, err
		}

		itemRows = append(itemRows, &itemRow)
		itemRowIds = append(itemRowIds, itemRow.Id)
	}

	if err = rows.Err(); err != nil {
		slog.Error(
			"Failed to process retrieved item rows",
			slog.String("error", err.Error()),
		)
		return
	}

	return
}

func (d *DatabaseStore) GetBundles(reportIds ...any) (bundleRowIds []int64, bundleRows []*TelemetryBundleRow, err error) {
	// generate the SQL populate query statement for the bundles table
	query, queryBundleIds := genSqlPopulateQuery(
		"bundles",
		[]string{"id", "bundleId", "bundleTimestamp", "bundleClientId", "bundleCustomerId", "bundleAnnotations", "reportId"},
		"reportId",
		reportIds,
	)

	// NOTE: Query() extra args must be of type any hence queryIds is type []any
	rows, err := d.Conn.Query(query, queryBundleIds...)
	if err != nil {
		slog.Error(
			"Failed to retrieve bundles with specified reportIds",
			slog.Any("reportIds", reportIds),
			slog.String("error", err.Error()),
		)
		return
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
			&bundleRow.ReportId); err != nil {
			slog.Error(
				"Failed to scan bundle row",
				slog.String("error", err.Error()),
			)
			return nil, nil, err
		}
		bundleRows = append(bundleRows, &bundleRow)
		bundleRowIds = append(bundleRowIds, bundleRow.Id)
	}

	if err = rows.Err(); err != nil {
		slog.Error(
			"Failed to process retrieved bundle rows",
			slog.String("error", err.Error()),
		)
		return
	}

	return
}

func (d *DatabaseStore) GetReports(ids ...any) (reportRowIds []int64, reportRows []*TelemetryReportRow, err error) {
	// generate the SQL populate query statement for the reports table
	query, queryIds := genSqlPopulateQuery(
		"reports",
		[]string{"id", "reportId", "reportTimestamp", "reportClientId", "reportAnnotations"},
		"id",
		ids,
	)

	// NOTE: Query() extra args must be of type any hence queryIds is type []any
	rows, err := d.Conn.Query(query, queryIds...)
	if err != nil {
		slog.Error(
			"Failed to retrieve reports with specified ids",
			slog.Any("ids", ids),
			slog.String("error", err.Error()),
		)
		return
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
		); err != nil {
			slog.Error(
				"Failed to scan report row",
				slog.String("error", err.Error()),
			)
			return nil, nil, err
		}
		reportRows = append(reportRows, &reportRow)
		reportRowIds = append(reportRowIds, reportRow.Id)
	}

	if err = rows.Err(); err != nil {
		slog.Error(
			"Failed to process retrieved report rows",
			slog.String("error", err.Error()),
		)
		return
	}

	return
}

func (d *DatabaseStore) GetItemCount(bundleIds ...any) (count int, err error) {
	// generate the SQL count query statement for the items table
	query, queryIds := genSqlCountQuery(
		"items",
		"id",
		"bundleId",
		bundleIds,
	)
	// NOTE: Query() extra args must be of type any hence queryIds is type []any
	err = d.Conn.QueryRow(query, queryIds...).Scan(&count)
	if err != nil {
		slog.Error(
			"Failed to count items associated with specified bundles",
			slog.Any("bundleIds", bundleIds),
			slog.String("error", err.Error()),
		)
		return
	}
	return
}

func (d *DatabaseStore) GetBundleCount(reportIds ...any) (count int, err error) {
	// generate the SQL count query statement for the bundles table
	query, queryIds := genSqlCountQuery(
		"bundles",
		"id",
		"reportId",
		reportIds,
	)
	// NOTE: Query() extra args must be of type any hence queryIds is type []any
	err = d.Conn.QueryRow(query, queryIds...).Scan(&count)
	if err != nil {
		slog.Error(
			"Failed to count bundles associated with specified reports",
			slog.Any("reportIds", reportIds),
			slog.String("error", err.Error()),
		)
		return
	}
	return
}

func (d *DatabaseStore) GetReportCount(ids ...any) (count int, err error) {
	// generate the SQL count query statement for the reports table
	query, queryIds := genSqlCountQuery(
		"reports",
		"id",
		"id",
		ids,
	)
	// NOTE: Query() extra args must be of type any hence queryIds is type []any
	err = d.Conn.QueryRow(query, queryIds...).Scan(&count)
	if err != nil {
		slog.Error(
			"Failed to count reports with specified ids",
			slog.Any("ids", ids),
			slog.String("error", err.Error()),
		)
		return
	}
	return
}

func (d *DatabaseStore) GetDataItemRowsInABundle(bundleId string) (itemRows []*TelemetryDataItemRow, err error) {
	//perform a join between the items table and the bundle table to filter the items by the bundle ID.
	rows, err := d.Conn.Query(
		`SELECT items.id,
		        items.itemId,
						items.itemType,
						items.itemTimestamp,
						items.itemAnnotations,
						items.itemData,
						items.itemChecksum,
						items.compression,
						items.bundleId
		 FROM items JOIN bundles ON items.bundleId = bundles.id
		 WHERE bundles.bundleId = ?`,
		bundleId,
	)

	if err != nil {
		slog.Error(
			"Failed to retrieve items with specified bundleId",
			slog.String("bundleId", bundleId),
			slog.String("error", err.Error()),
		)
		return
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
			&itemRow.Compression,
			&itemRow.BundleId); err != nil {
			slog.Error(
				"Failed to scan item row",
				slog.String("error", err.Error()),
			)
			return nil, err
		}

		// ItemData can be stored as compressed data
		itemRow.ItemData, err = utils.DecompressWhenNeeded(itemRow.ItemData, itemRow.Compression)
		if err != nil {
			slog.Error(
				"Failed to decompress item data",
				slog.String("itemId", itemRow.ItemId),
				slog.String("error", err.Error()),
			)
			return nil, err
		}

		itemRows = append(itemRows, &itemRow)

	}

	if err = rows.Err(); err != nil {
		slog.Error(
			"Failed to process retrieved item rows",
			slog.String("error", err.Error()),
		)
		return
	}

	return
}

func (d *DatabaseStore) GetBundleRowsInAReport(reportId string) (bundleRows []*TelemetryBundleRow, err error) {
	//perform a join between the bundles table and the report table to filter the bundles by the report ID.
	rows, err := d.Conn.Query(
		`SELECT bundles.id,
		        bundles.bundleId,
						bundles.bundleTimestamp,
						bundles.bundleClientId,
						bundles.bundleCustomerId,
						bundles.bundleAnnotations,
						bundles.reportId
		 FROM bundles JOIN reports ON bundles.reportId = reports.id
		 WHERE reports.reportId = ?`,
		reportId,
	)

	if err != nil {
		slog.Error(
			"Failed to retrieve bundles with specified reportId",
			slog.String("reportId", reportId),
			slog.String("error", err.Error()),
		)
		return
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
			&bundleRow.ReportId,
		); err != nil {
			slog.Error(
				"Failed to scan bundle row",
				slog.String("error", err.Error()),
			)
			return nil, err
		}
		bundleRows = append(bundleRows, &bundleRow)
	}

	if err = rows.Err(); err != nil {
		slog.Error(
			"Failed to process retrieved bundle rows",
			slog.String("error", err.Error()),
		)
		return
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
