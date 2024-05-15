package datastore

import (
	"database/sql"
	"os"
	"strings"

	"github.com/xyproto/randomstring"
)

func CleanAll(dsparams string) {
	deleteAllRows(dsparams)
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
	return randomstring.HumanFriendlyString(length)
}