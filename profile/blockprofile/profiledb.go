package blockprofile

import (
	"database/sql"
	"fmt"
	"os"

	// Your main or test packages require this import so the sql package is properly initialized.
	_ "github.com/mattn/go-sqlite3"
)

const (
	// bufferSize of the in-memory buffer for storing profile data
	bufferSize = 1000

	// SQL statement for inserting a profile record of a new block
	insertBlockSQL = `
INSERT INTO blockProfile (
	block, tBlock, tSequential, tCritical, tCommit, speedup, ubNumProc, numTx, gasBlock
) VALUES (
	?, ?, ?, ?, ?, ?, ?, ?, ?
)
`
	// SQL statement for inserting a profile record of a new transaction
	insertTxSQL = `
INSERT INTO txProfile (
block, tx, txType, duration, gas
) VALUES (
?, ?, ?, ?, ?
)
`

	// SQL statement for creating profiling tables
	createSQL = `
PRAGMA journal_mode = MEMORY;
CREATE TABLE IF NOT EXISTS blockProfile (
	block INTEGER,
	tBlock INTEGER,
	tSequential INTEGER,
	tCritical INTEGER,
	tCommit INTEGER,
	speedup FLOAT,
	ubNumProc INTEGER,
	numTx INTEGER,
	gasBlock INTEGER
);
CREATE TABLE IF NOT EXISTS txProfile (
	block INTEGER,
	tx    INTEGER, 
	txType INTEGER,
	duration INTEGER,
	gas INTEGER
);
`
)

// ProfileDB is a profiling database for block processing.
type ProfileDB struct {
	sql       *sql.DB       // Sqlite3 database
	blockStmt *sql.Stmt     // Prepared insert statement for a block
	txStmt    *sql.Stmt     // Prepared insert statement for a transaction
	buffer    []ProfileData // record buffer
}

// NewProfileDB constructs a new profiling database.
func NewProfileDB(dbFile string) (*ProfileDB, error) {
	if _, err := os.Stat(dbFile); err != nil {
		file, err := os.Create(dbFile)
		if err != nil {
			return nil, fmt.Errorf("cannot create file for database %v; %v", dbFile, err)
		}
		err = file.Close()
		if err != nil {
			return nil, fmt.Errorf("cannot close db file; %v", err)
		}
	}
	// open SQLITE3 DB
	sqlDB, err := sql.Open("sqlite3", dbFile)
	if err != nil {
		return nil, fmt.Errorf("failed to open database %v; %v", dbFile, err)
	}
	// create profile schema if not exists
	if _, err = sqlDB.Exec(createSQL); err != nil {
		return nil, fmt.Errorf("sqlDB.Exec, err: %q", err)
	}
	// prepare INSERT statements for subsequent use
	blockStmt, err := sqlDB.Prepare(insertBlockSQL)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare a SQL statement for block profile; %v", err)
	}
	txStmt, err := sqlDB.Prepare(insertTxSQL)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare a SQL statement for tx profile; %v", err)
	}

	return &ProfileDB{
		sql:       sqlDB,
		blockStmt: blockStmt,
		txStmt:    txStmt,
		buffer:    make([]ProfileData, 0, bufferSize),
	}, nil
}

// Close flushes buffers of profiling database and closes the profiling database.
func (db *ProfileDB) Close() error {
	defer func() {
		db.txStmt.Close()
		db.blockStmt.Close()
		db.sql.Close()
	}()
	if err := db.Flush(); err != nil {
		return err
	}
	return nil
}

// Add a profile data record to the profiling database.
func (db *ProfileDB) Add(ProfileData ProfileData) error {
	db.buffer = append(db.buffer, ProfileData)
	if len(db.buffer) == cap(db.buffer) {
		if err := db.Flush(); err != nil {
			return fmt.Errorf("unable to flush ProfileDatas: %w", err)
		}
	}
	return nil
}

// Flush the profiling records in the database.
func (db *ProfileDB) Flush() error {
	// open new transaction
	tx, err := db.sql.Begin()
	if err != nil {
		return err
	}
	// write profiling records into sqlite3 database
	for _, ProfileData := range db.buffer {
		// write block data
		_, err := tx.Stmt(db.blockStmt).Exec(ProfileData.curBlock, ProfileData.tBlock, ProfileData.tSequential, ProfileData.tCritical,
			ProfileData.tCommit, ProfileData.speedup, ProfileData.ubNumProc, ProfileData.numTx, ProfileData.gasBlock)
		if err != nil {
			_ = tx.Rollback()
			return err
		}
		// write transactions
		for i, tTransaction := range ProfileData.tTransactions {
			_, err = tx.Stmt(db.txStmt).Exec(ProfileData.curBlock, i, ProfileData.tTypes[i], tTransaction, ProfileData.gasTransactions[i])
			if err != nil {
				_ = tx.Rollback()
				return err
			}
		}
	}
	// clear buffer
	db.buffer = db.buffer[:0]
	// commit transaction
	return tx.Commit()
}

// DeleteByBlockRange deletes information for a block range; used prior insertion
func (db *ProfileDB) DeleteByBlockRange(firstBlock, lastBlock uint64) (int64, error) {
	const (
		blockProfile = "blockProfile"
		txProfile    = "txProfile"
	)
	var totalNumRows int64

	tx, err := db.sql.Begin()
	if err != nil {
		return 0, err
	}

	for _, table := range []string{blockProfile, txProfile} {
		deleteSql := fmt.Sprintf("DELETE FROM %s WHERE block >= %d AND block <= %d;", table, firstBlock, lastBlock)
		res, err := db.sql.Exec(deleteSql)
		if err != nil {
			return 0, err
		}

		numRowsAffected, err := res.RowsAffected()
		if err != nil {
			return 0, err
		}

		totalNumRows += numRowsAffected
	}

	if err := tx.Commit(); err != nil {
		return 0, err
	}

	return totalNumRows, nil
}
