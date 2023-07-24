// Package ProfileDatas provides an SQLite based ProfileDatas database.
package parallelisation

import (
	"fmt"
	"io/ioutil"
	"math/rand"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
)

func tempFile(require *require.Assertions) string {
	file, err := ioutil.TempFile("", "*.db")
	require.NoError(err)
	file.Close()
	return file.Name()
}

func TestAdd(t *testing.T) {
	require := require.New(t)

	dbFile := tempFile(require)
	t.Logf("db file: %s", dbFile)
	db, err := NewProfileDB(dbFile)
	require.NoError(err)
	defer db.Close()

	ProfileData := ProfileData{
		curBlock:    5637800,
		tBlock:      5838,
		tSequential: 4439,
		tCritical:   2424,
		tCommit:     1398,
		speedup:     1.527263,
		ubNumProc:   2,
		numTx:       3,
	}

	err = db.Add(ProfileData)
	require.NoError(err)

	require.Equal(len(db.buffer), 1)
}

// TestDeleteBlockRangeOverlap tests profileDB.DeleteByBlockRange function
func TestDeleteBlockRangeOverlap(t *testing.T) {
	require := require.New(t)

	dbFile := tempFile(require)
	t.Logf("db file: %s", dbFile)
	db, err := NewProfileDB(dbFile)
	require.NoError(err)

	startBlock, endBlock := uint64(500), uint64(2500)
	blockRange := endBlock - startBlock
	for i := startBlock; i <= endBlock; i++ {
		profileData := ProfileData{
			curBlock:    uint64(i),
			tBlock:      5838,
			tSequential: 4439,
			tCritical:   2424,
			tCommit:     1398,
			speedup:     1.527263,
			ubNumProc:   2,
			numTx:       3,
		}
		err = db.Add(profileData)
		require.NoError(err)
	}

	numDeletedRows, err := db.DeleteByBlockRange(startBlock, endBlock)
	require.NoError(err)
	if numDeletedRows != int64(blockRange) {
		t.Errorf("unexpected number of rows affected by deletion")
	}
	db.Close()

	db, err = NewProfileDB(dbFile)
	require.NoError(err)
	defer db.Close()
	for i := startBlock; i <= endBlock; i++ {
		profileData := ProfileData{
			curBlock:    uint64(i),
			tBlock:      5838,
			tSequential: 4439,
			tCritical:   2424,
			tCommit:     1398,
			speedup:     1.527263,
			ubNumProc:   2,
			numTx:       3,
		}
		err = db.Add(profileData)
		require.NoError(err)
	}

	startDeleteBlock, endDeleteBlock := uint64(0), uint64(500)
	numDeletedRows, err = db.DeleteByBlockRange(startDeleteBlock, endDeleteBlock)
	require.NoError(err)
	if numDeletedRows != 1 {
		t.Errorf("unexpected number of rows affected by deletion")
	}
}

// TestDeleteBlockRangeNoOverlap tests profileDB.DeleteByBlockRange function
func TestDeleteBlockRangeNoOverlap(t *testing.T) {
	require := require.New(t)

	dbFile := tempFile(require)
	t.Logf("db file: %s", dbFile)
	db, err := NewProfileDB(dbFile)
	require.NoError(err)
	defer db.Close()

	startBlock, endBlock := uint64(500), uint64(2500)
	for i := startBlock; i <= endBlock; i++ {
		profileData := ProfileData{
			curBlock:    uint64(i),
			tBlock:      5838,
			tSequential: 4439,
			tCritical:   2424,
			tCommit:     1398,
			speedup:     1.527263,
			ubNumProc:   2,
			numTx:       3,
		}
		err = db.Add(profileData)
		require.NoError(err)
	}

	startDeleteBlock, endDeleteBlock := uint64(0), uint64(499)
	numDeletedRows, err := db.DeleteByBlockRange(startDeleteBlock, endDeleteBlock)
	require.NoError(err)
	if numDeletedRows != 0 {
		t.Errorf("unexpected number of rows affected by deletion")
	}
}

func BenchmarkAdd(b *testing.B) {
	require := require.New(b)
	dbFile := tempFile(require)
	b.Logf("db file: %s", dbFile)

	db, err := NewProfileDB(dbFile)
	require.NoError(err)
	ProfileData := ProfileData{
		curBlock:    5637800,
		tBlock:      5838,
		tSequential: 4439,
		tCritical:   2424,
		tCommit:     1398,
		speedup:     1.527263,
		ubNumProc:   2,
		numTx:       3,
	}
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		err := db.Add(ProfileData)
		require.NoError(err)
	}
}

func ExampleDB() {
	dbFile := "/tmp/db-test" + time.Now().Format(time.RFC3339)
	db, err := NewProfileDB(dbFile)
	if err != nil {
		fmt.Println("ERROR: create -", err)
		return
	}
	defer db.Close()

	const count = 10_000
	for i := 0; i < count; i++ {
		ProfileData := ProfileData{
			curBlock:    5637800,
			tBlock:      5838,
			tSequential: 4439,
			tCritical:   2424,
			tCommit:     1398,
			speedup:     rand.Float64() * 10,
			ubNumProc:   2,
			numTx:       3,
		}
		if err := db.Add(ProfileData); err != nil {
			fmt.Println("ERROR: insert - ", err)
			return
		}
	}

	fmt.Printf("inserted %d records\n", count)
	// Output:
	// inserted 10000 records
}
