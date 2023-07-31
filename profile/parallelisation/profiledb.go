package parallelisation

import (
	"database/sql"
	"fmt"

	// Your main or test packages require this import so the sql package is properly initialized.
	_ "github.com/mattn/go-sqlite3"
)

const (
	// bufferSize is the buffer size of the in-memory buffer for storing profile data
	bufferSize = 1000

	// SQL for inserting new data records
	insertSQL = `
INSERT INTO parallelprofile (
	block, tBlock, tSequential, tCritical, tCommit, speedup, ubNumProc, numTx
) VALUES (
	?, ?, ?, ?, ?, ?, ?, ?
)
`

	// SQL for creating a new profiling table
	createSQL = `
	PRAGMA journal_mode = MEMORY;
	CREATE TABLE IF NOT EXISTS parallelprofile (
    block INTEGER,
	tBlock INTEGER,
	tSequential INTEGER,
	tCritical INTEGER,
	tCommit INTEGER,
	speedup FLOAT,
	ubNumProc INTEGER,
	numTx INTEGER);
	CREATE TABLE IF NOT EXISTS txProfile (
    block INTEGER,
	tx    INTEGER, 
	duration INTEGER
);
`

	// SQL for deleting data records for a given block range
	deleteSql = `
	DELETE FROM parallelprofile 
	WHERE block >= $1 AND block <= $2
`
)

// ProfileDB is a database of ProfileData
type ProfileDB struct {
	sql    *sql.DB       // Sqlite3 database
	stmt   *sql.Stmt     // Prepared insert statement
	buffer []ProfileData // record buffer
}

// NewProfileDB constructs a ProfileDatas value for managing stock ProfileDatas in a
// SQLite database. This API is not thread safe.
func NewProfileDB(dbFile string) (*ProfileDB, error) {
	// open SQLITE3 DB
	sqlDB, err := sql.Open("sqlite3", dbFile)
	if err != nil {
		return nil, err
	}
	// create profile schema if not exists
	if _, err = sqlDB.Exec(createSQL); err != nil {
		return nil, fmt.Errorf("sqlDB.Exec, err: %q", err)
	}
	// prepare the INSERT statement for subsequent use
	stmt, err := sqlDB.Prepare(insertSQL)
	if err != nil {
		return nil, err
	}
	db := ProfileDB{
		sql:    sqlDB,
		stmt:   stmt,
		buffer: make([]ProfileData, 0, bufferSize),
	}
	return &db, nil
}

// Close flushes all ProfileDatas to the database and prevents any future trading.
func (db *ProfileDB) Close() error {
	defer func() {
		db.stmt.Close()
		db.sql.Close()
	}()
	if err := db.Flush(); err != nil {
		return err
	}
	return nil
}

// Add stores a profile data record into a buffer. Once the buffer is full, the
// records are flushed into the database.
func (db *ProfileDB) Add(ProfileData ProfileData) error {
	db.buffer = append(db.buffer, ProfileData)
	if len(db.buffer) == cap(db.buffer) {
		if err := db.Flush(); err != nil {
			return fmt.Errorf("unable to flush ProfileDatas: %w", err)
		}
	}
	return nil
}

// Flush inserts pending ProfileDatas into the database inside DB transaction.
func (db *ProfileDB) Flush() error {
	tx, err := db.sql.Begin()
	if err != nil {
		return err
	}
	for _, ProfileData := range db.buffer {
		_, err := tx.Stmt(db.stmt).Exec(ProfileData.curBlock, ProfileData.tBlock, ProfileData.tSequential, ProfileData.tCritical,
			ProfileData.tCommit, ProfileData.speedup, ProfileData.ubNumProc, ProfileData.numTx)
		// write into new txProfile table here the transaction durations
		if err != nil {
			tx.Rollback()
			return err
		}
	}
	db.buffer = db.buffer[:0]
	return tx.Commit()
}

// DeleteByBlockRange deletes rows in a given block range
func (db *ProfileDB) DeleteByBlockRange(firstBlock, lastBlock uint64) (int64, error) {
	tx, err := db.sql.Begin()
	if err != nil {
		return 0, err
	}
	stmt, err := db.sql.Prepare(deleteSql)
	if err != nil {
		return 0, err
	}
	res, err := tx.Stmt(stmt).Exec(firstBlock, lastBlock)
	if err != nil {
		tx.Rollback()
		return 0, err
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return 0, err
	}

	if err := tx.Commit(); err != nil {
		return 0, err
	}

	return rows, nil
}
