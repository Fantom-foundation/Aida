// Package ProfileDatas provides an SQLite based ProfileDatas database.
package parallelisation

import (
	"fmt"
	"math/rand"
	"os"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
)

func tempFile(require *require.Assertions) string {
	file, err := os.CreateTemp("", "*.db")
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
		curBlock:      5637800,
		tBlock:        5838,
		tSequential:   4439,
		tCritical:     2424,
		tCommit:       1398,
		speedup:       1.527263,
		ubNumProc:     2,
		numTx:         3,
		tTransactions: []int64{2382388, 11218838, 5939392888},
	}

	err = db.Add(ProfileData)
	require.NoError(err)

	require.Equal(len(db.buffer), 1)

	require.Equal(len(db.buffer[0].tTransactions), 3)
}

func TestFlush(t *testing.T) {
	// db has 0 records
	require := require.New(t)
	dbFile := tempFile(require)
	t.Logf("db file: %s", dbFile)
	db, err := NewProfileDB(dbFile)
	require.NoError(err)
	err = db.Add(ProfileData{})
	require.NoError(err)

	err = db.Flush()
	require.NoError(err)
	db.Close()

	// db has 2 records
	db, err = NewProfileDB(dbFile)
	require.NoError(err)

	pd := ProfileData{
		curBlock:      5637800,
		tBlock:        5838,
		tSequential:   4439,
		tCritical:     2424,
		tCommit:       1398,
		speedup:       1.527263,
		ubNumProc:     2,
		numTx:         3,
		tTransactions: []int64{2382388, 11218838, 5939392888},
	}

	err = db.Add(pd)
	require.NoError(err)

	pd = ProfileData{
		curBlock:      3239933,
		tBlock:        44939,
		tSequential:   3493848,
		tCritical:     434838,
		tCommit:       2332,
		speedup:       1.203983,
		ubNumProc:     2,
		numTx:         2,
		tTransactions: []int64{2382388, 11218838},
	}
	err = db.Add(pd)
	require.NoError(err)
	require.Len(db.buffer, 2)
	require.Len(db.buffer[0].tTransactions, 3)
	require.Len(db.buffer[1].tTransactions, 2)
	err = db.Flush()
	require.NoError(err)
	require.Len(db.buffer, 0)
	db.Close()

	// trigger Flush method inside Add
	db, err = NewProfileDB(dbFile)
	require.NoError(err)
	defer db.Close()

	for i := 1; i < bufferSize; i++ {
		profileData := ProfileData{
			curBlock:      uint64(i),
			tBlock:        5838,
			tSequential:   4439,
			tCritical:     2424,
			tCommit:       1398,
			speedup:       1.527263,
			ubNumProc:     2,
			numTx:         2,
			tTransactions: []int64{2382388, 11218838},
		}
		err = db.Add(profileData)
		require.NoError(err)
		require.Len(db.buffer, i)
	}

	pd = ProfileData{
		curBlock:      uint64(bufferSize),
		tBlock:        5838,
		tSequential:   4439,
		tCritical:     2424,
		tCommit:       1398,
		speedup:       1.527263,
		ubNumProc:     2,
		numTx:         3,
		tTransactions: []int64{2382388, 11218838, 232348228},
	}

	err = db.Add(pd)
	require.NoError(err)
	require.Len(db.buffer, 0)
}

// TestDeleteBlockRangeOverlap tests profileDB.DeleteByBlockRange function
func TestDeleteBlockRangeOverlapOneTx(t *testing.T) {
	require := require.New(t)

	dbFile := tempFile(require)
	t.Logf("db file: %s", dbFile)
	db, err := NewProfileDB(dbFile)
	require.NoError(err)

	startBlock, endBlock := uint64(500), uint64(2500)
	blockRange := endBlock - startBlock
	for i := startBlock; i <= endBlock; i++ {
		profileData := ProfileData{
			curBlock:      uint64(i),
			tBlock:        5838,
			tSequential:   4439,
			tCritical:     2424,
			tCommit:       1398,
			speedup:       1.527263,
			ubNumProc:     2,
			numTx:         1,
			tTransactions: []int64{232939829},
		}
		err = db.Add(profileData)
		require.NoError(err)
	}

	numDeletedRows, err := db.DeleteByBlockRange(startBlock, endBlock)
	require.NoError(err)
	if numDeletedRows != int64(2*blockRange) {
		t.Errorf("unexpected number of rows affected by deletion, expected: %d, got: %d", 2*blockRange, numDeletedRows)
	}
	db.Close()

	db, err = NewProfileDB(dbFile)
	require.NoError(err)
	defer db.Close()
	for i := startBlock; i <= endBlock; i++ {
		profileData := ProfileData{
			curBlock:      uint64(i),
			tBlock:        5838,
			tSequential:   4439,
			tCritical:     2424,
			tCommit:       1398,
			speedup:       1.527263,
			ubNumProc:     2,
			numTx:         1,
			tTransactions: []int64{232939829},
		}
		err = db.Add(profileData)
		require.NoError(err)
	}

	startDeleteBlock, endDeleteBlock := uint64(0), uint64(500)
	numDeletedRows, err = db.DeleteByBlockRange(startDeleteBlock, endDeleteBlock)
	require.NoError(err)
	if numDeletedRows != 2 {
		t.Errorf("unexpected number of rows affected by deletion")
	}
}

func TestDeleteBlockRangeOverlapMultipleTx(t *testing.T) {
	require := require.New(t)

	dbFile := tempFile(require)
	t.Logf("db file: %s", dbFile)
	db, err := NewProfileDB(dbFile)
	require.NoError(err)

	startBlock, endBlock := uint64(500), uint64(2500)
	blockRange := endBlock - startBlock
	numTx := 4
	for i := startBlock; i <= endBlock; i++ {
		profileData := ProfileData{
			curBlock:      uint64(i),
			tBlock:        5838,
			tSequential:   4439,
			tCritical:     2424,
			tCommit:       1398,
			speedup:       1.527263,
			ubNumProc:     2,
			numTx:         numTx,
			tTransactions: []int64{232939829, 938828288, 92388277, 9238828},
		}
		err = db.Add(profileData)
		require.NoError(err)
	}

	numDeletedRows, err := db.DeleteByBlockRange(startBlock, endBlock)
	require.NoError(err)
	expNumRows := blockRange + uint64(numTx)*blockRange
	if numDeletedRows != int64(expNumRows) {
		t.Errorf("unexpected number of rows affected by deletion, expected: %d, got: %d", expNumRows, numDeletedRows)
	}
	db.Close()

	db, err = NewProfileDB(dbFile)
	require.NoError(err)
	defer db.Close()
	for i := startBlock; i <= endBlock; i++ {
		profileData := ProfileData{
			curBlock:      uint64(i),
			tBlock:        5838,
			tSequential:   4439,
			tCritical:     2424,
			tCommit:       1398,
			speedup:       1.527263,
			ubNumProc:     2,
			numTx:         numTx,
			tTransactions: []int64{232939829, 938828288, 92388277, 9238828},
		}
		err = db.Add(profileData)
		require.NoError(err)
	}

	startDeleteBlock, endDeleteBlock := uint64(0), uint64(500)
	numDeletedRows, err = db.DeleteByBlockRange(startDeleteBlock, endDeleteBlock)
	require.NoError(err)
	if numDeletedRows != 1+int64(numTx) {
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
			curBlock:      uint64(i),
			tBlock:        5838,
			tSequential:   4439,
			tCritical:     2424,
			tCommit:       1398,
			speedup:       1.527263,
			ubNumProc:     2,
			numTx:         3,
			tTransactions: []int64{232444, 92398, 9282887},
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
			curBlock:      5637800,
			tBlock:        5838,
			tSequential:   4439,
			tCritical:     2424,
			tCommit:       1398,
			speedup:       rand.Float64() * 10,
			ubNumProc:     2,
			numTx:         3,
			tTransactions: []int64{2382388, 11218838, 5939392888},
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

func TestFlushProfileData(t *testing.T) {
	require := require.New(t)
	dbFile := tempFile(require)
	t.Logf("db file: %s", dbFile)

	db, err := NewProfileDB(dbFile)
	require.NoError(err)
	defer db.Close()

	ProfileData := ProfileData{
		curBlock:      5637800,
		tBlock:        5838,
		tSequential:   4439,
		tCritical:     2424,
		tCommit:       1398,
		speedup:       1.527263,
		ubNumProc:     2,
		numTx:         4,
		tTransactions: []int64{292988, 8387773, 923828772},
	}

	// start db transaction
	tx, err := db.sql.Begin()
	require.NoError(err)
	res, err := tx.Stmt(db.blockStmt).Exec(ProfileData.curBlock, ProfileData.tBlock, ProfileData.tSequential, ProfileData.tCritical,
		ProfileData.tCommit, ProfileData.speedup, ProfileData.ubNumProc, ProfileData.numTx)
	require.NoError(err)
	numRowsAffected, err := res.RowsAffected()
	require.NoError(err)
	if numRowsAffected != 1 {
		t.Errorf("invalid numRowsAffected value")
	}

	for i, tTransaction := range ProfileData.tTransactions {
		res, err = tx.Stmt(db.txStmt).Exec(ProfileData.curBlock, i, tTransaction)
		require.NoError(err)
		numRowsAffected, err := res.RowsAffected()
		require.NoError(err)
		if numRowsAffected != 1 {
			t.Errorf("invalid numRowsAffected value")
		}
	}
	require.NoError(tx.Commit())
}
