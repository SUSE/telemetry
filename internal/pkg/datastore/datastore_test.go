package datastore

import (
	"os"
	"testing"

	"github.com/stretchr/testify/suite"
)

type DataStorerTestSuite struct {
	suite.Suite
	datastorer DataStorer
	db         string
}

func (t *DataStorerTestSuite) SetupSuite() {
	t.db = "sqlite3:/tmp/datastore.db"
}

func (t *DataStorerTestSuite) TearDownSuite() {
	CleanAll(t.db)
}

/*
This test will test all the DataStorer method definitions
for file based, db based with sqlite3 backend and memory based
storage mechanisms.
*/

func (t *DataStorerTestSuite) TestAddGetDeleteList() {

	CleanAll(t.db)
	key := GenerateRandomString(10)
	value := "test"

	dataStorer, err := NewDatabaseStore(t.db)
	t.NoError(err)

	t.datastorer = dataStorer

	// test add method
	err = t.datastorer.Add(key, []byte(value))
	t.NoError(err)

	// test get method
	v, err := t.datastorer.Get(key)
	t.NoError(err)
	t.Equal(value, string(v))

	//test delete method
	err = t.datastorer.Delete(key)
	t.NoError(err)

	//test get again after deletion
	v, _ = t.datastorer.Get(key)
	t.Nil(v)

	// test list
	key1 := GenerateRandomString(10)
	key2 := GenerateRandomString(10)

	err = t.datastorer.Add(key1, []byte(value))
	t.NoError(err)
	err = t.datastorer.Add(key2, []byte(value))
	t.NoError(err)

	arr, err := t.datastorer.List()
	t.NoError(err)
	t.Equal(2, len(arr))
	t.Contains(arr, key1)
	t.Contains(arr, key2)

	CleanAll(t.db)

}

/*
func (t *DataStorerTestSuite) TestUnsupported() {
	//Test unsupported datastore
	ds, err := NewDataStore("unsupported", "/tmp/datastore")
	t.Error(err)
	t.Nil(ds)
}

func (t *DataStorerTestSuite) TestPermissionDenied() {
	//Test Access Denied for mkdir
	ds, err := NewDataStore("dir", "/etc/datastore")
	t.Error(err)
	t.Nil(ds)
}

func (t *DataStorerTestSuite) TestUnsupportedDatabase() {
	//Test Unsupport database backend
	ds, err := NewDataStore("db", "mongodb://localhost:27017/db")
	t.Error(err)
	t.Nil(ds)
}
*/

func (t *DataStorerTestSuite) TestDatabaseWriteError() {
	//Test database write error
	db := "/tmp/" + GenerateRandomString(10) + ".db"
	f, _ := os.Create(db)
	os.Chmod(db, 0444)
	defer f.Close()
	ds, err := NewDatabaseStore("sqlite3:" + db)
	t.Error(err)
	t.Nil(ds)

	os.Remove(db)
}

func TestDataStorerSuite(t *testing.T) {
	suite.Run(t, new(DataStorerTestSuite))
}
