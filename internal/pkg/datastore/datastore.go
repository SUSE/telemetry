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

func NewDataStore(storeType string, storeParams string) (dataStore DataStorer, err error) {

	switch storeType {
	case "dir":
		dataStore, err = NewFileStore(storeParams)
	case "memory":
		dataStore, err = NewMemStore(storeParams)
	case "db":
		dataStore, err = NewDatabaseStore(storeParams)
	default:
		dataStore, err = nil, fmt.Errorf("unsupported data store type %q", storeType)
	}

	return
}

// FileStorer is an implementation for storing data in a file.
const FILE_STORE_SFX = ".data"

type FileStore struct {
	RootPath string
}

func (fs *FileStore) keyToPath(key string) string {
	return filepath.Join(fs.RootPath, key+FILE_STORE_SFX)
}

func (fs *FileStore) keyFromPath(keyPath string) string {
	return strings.TrimSuffix(filepath.Base(keyPath), FILE_STORE_SFX)
}

func (fs *FileStore) listFiles() (files []string, err error) {
	return filepath.Glob(fs.keyToPath("*"))
}

func NewFileStore(rootpath string) (*FileStore, error) {
	err := os.MkdirAll(rootpath, 0700)
	if err != nil {
		return nil, err
	}
	return &FileStore{RootPath: rootpath}, nil
}

// Add a file with name key to a file system with data being the contents of the file
func (fs *FileStore) Add(key string, data []byte) (err error) {
	filename := fs.keyToPath(key)
	err = os.WriteFile(filename, data, 0644)
	if err != nil {
		log.Printf("error writing output file %q: %s\n", filename, err)
		return
	}

	log.Printf("successfully stored data in %q\n", filename)
	return
}

func (fs *FileStore) getData(keyPath string) (contents []byte, err error) {
	contents, err = os.ReadFile(keyPath)
	if err != nil {
		log.Println("Error reading file:", err)
		return
	}
	return
}

// Get data from the file based on a key (filename)
func (fs *FileStore) Get(key string) ([]byte, error) {
	return fs.getData(fs.keyToPath(key))
}

// List the keys in the data store
func (fs *FileStore) List() ([]string, error) {
	var lst []string
	keyFiles, err := fs.listFiles()
	if err != nil {
		log.Printf("error listing files in %q: %s", fs.RootPath, err.Error())
		return nil, err
	}

	for _, keyPath := range keyFiles {
		key := fs.keyFromPath(keyPath)
		lst = append(lst, key)
	}
	return lst, nil
}

// Delete the file given the filename
func (fs *FileStore) Delete(key string) (err error) {
	filename := fs.keyToPath(key)
	err = os.Remove(filename)
	if err != nil {
		log.Printf("failed to delete %q: %s", filename, err.Error())
		return
	}

	log.Printf("successfully deleted %q", filename)
	return

}

// MemStore is an implementation for storing data in memory
type MemStore struct {
	elements map[string][]byte
}

func NewMemStore(_ string) (*MemStore, error) {
	return &MemStore{elements: make(map[string][]byte)}, nil
}

// add stores the value with the specified key in memory.
func (ms *MemStore) Add(key string, data []byte) error {
	ms.elements[key] = data
	log.Printf("successfully added key %q", key)
	return nil
}

// Get data from the map based on a key.
func (ms *MemStore) Get(key string) ([]byte, error) {
	value, ok := ms.elements[key]
	if !ok {
		log.Printf("key %q not found", key)
	}
	return value, nil
}

func (ms *MemStore) List() ([]string, error) {
	var lst []string
	for key := range ms.elements {
		lst = append(lst, key)
	}
	return lst, nil
}

// Delete removes the value associated with the specified key from memory.
func (ms *MemStore) Delete(key string) error {
	delete(ms.elements, key)
	log.Printf("successfully delete %q entry \n", key)
	return nil
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
