package datastore

import (
	"database/sql"
	"math/rand"
	"os"
	"strings"
	"time"
)

func CleanAll(dstype string, dsparams string) {
	switch dstype {
	case "dir":
		removeFiles(dsparams)
	case "memory":
		// nothing to do for memory type
	case "db":
		deleteAllRows(dsparams)
	}
}

func removeFiles(directory string) {
	// Open the directory and read all its files.
	dir, _ := os.Open(directory)
	files, _ := dir.Readdir(0)
	for index := range files {
		file := files[index]
		os.Remove(directory + "/" + file.Name())
	}
}

func deleteAllRows(dbparams string) error {
	params := strings.SplitN(dbparams, ":", 2)
	db, err := sql.Open(params[0], params[1])
	if err != nil {
		return err
	}
	_, err = db.Exec("DELETE FROM datastore")
	if err != nil {
		return err
	}
	return nil
}

func GenerateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz" +
		"ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

	var seededRand *rand.Rand = rand.New(
		rand.NewSource(time.Now().UnixNano()))

	b := make([]byte, length)
	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}
	return string(b)
}
