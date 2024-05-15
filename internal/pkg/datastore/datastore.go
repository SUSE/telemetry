package datastore

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

// Storer defines the behavior for Add, Get, Delete data
type DataStorer interface {
	Add(key string, data []byte) error
	Get(key string) ([]byte, error)
	Delete(key string) error
	List() ([]string, error)
}

// DatabaseStorer is an implementation for storing data in a database.
type DatabaseStore struct {
	db *sql.DB
}

func NewDatabaseStore(dbParams string) (*DatabaseStore, error) {
	params := strings.SplitN(dbParams, ":", 2)
	var (
		db  *sql.DB
		err error
	)
	switch params[0] {
	case "sqlite3":
		if _, err := os.Stat(params[1]); os.IsNotExist(err) {
			dirPath := filepath.Dir(params[1])
			os.MkdirAll(dirPath, 0700)

			file, err := os.Create(params[1])
			if err != nil {
				log.Fatal(err.Error())
				return nil, err
			}
			file.Close()
			log.Println("created SQLite file", params[1])
		}

		db, err = sql.Open("sqlite3", params[1])
		if err != nil {
			log.Printf("failed to open the database: %s", err.Error())
			return nil, err
		}
	default:
		err = fmt.Errorf("unsupported database type %q", params[0])
		log.Print(err.Error())
		return nil, err
	}

	// Create a table named datastore
	_, err = db.Exec("CREATE TABLE IF NOT EXISTS datastore (key STRING PRIMARY KEY, value BLOB)")
	if err != nil {
		log.Print(err.Error())
		return nil, err
	}
	return &DatabaseStore{db: db}, nil
}

// Add stores the provided data in a database.
func (dbs *DatabaseStore) Add(key string, data []byte) error {
	_, err := dbs.db.Exec("INSERT OR REPLACE INTO datastore (key, value) VALUES (?, ?)", key, data)
	if err != nil {
		log.Printf("failed to add %q entry to DB: %s", key, err.Error())
	}
	return err
}

// Get data from the database based on a key.
func (dbs *DatabaseStore) Get(key string) ([]byte, error) {
	var data []byte
	err := dbs.db.QueryRow("SELECT value FROM datastore WHERE key = ?", key).Scan(&data)
	if err != nil {
		log.Printf("failed to retrieve value for %q: %s", key, err.Error())
	}

	return data, nil
}

func (dbs *DatabaseStore) List() ([]string, error) {
	var keys []string
	rows, err := dbs.db.Query("SELECT key FROM datastore")
	if err != nil {
		log.Fatal(err)
	}
	//defer rows.Close()

	// Parse and print JSON data
	for rows.Next() {
		var key string
		err := rows.Scan(&key)
		if err != nil {
			log.Fatal(err)
		}
		keys = append(keys, key)
	}
	return keys, nil
}

// Delete data from the database based on a key.
func (dbs *DatabaseStore) Delete(key string) error {
	_, err := dbs.db.Exec("DELETE FROM datastore WHERE key = ?", key)
	if err != nil {
		log.Printf("failed to delete %q entry from DB: %s", key, err.Error())
	}
	return err
}
