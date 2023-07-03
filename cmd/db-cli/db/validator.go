package db

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"hash"
	"time"

	"github.com/Fantom-foundation/Aida/logger"
	substate "github.com/Fantom-foundation/Substate"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/op/go-logging"
)

// validator is used to iterate over all key/value pairs inside AidaDb and creating md5 hash
type validator struct {
	db     ethdb.Database
	start  time.Time
	hasher hash.Hash
	log    *logging.Logger
}

// newDbValidator returns new insta of validator
func newDbValidator(pathToDb, logLevel string) *validator {
	l := logger.NewLogger(logLevel, "Db-Validator")

	db, err := rawdb.NewLevelDBDatabase(pathToDb, 1024, 100, "profiling", true)
	if err != nil {
		l.Fatalf("cannot create new db; %v", err)
	}

	return &validator{
		db:     db,
		start:  time.Now(),
		hasher: md5.New(),
		log:    l,
	}
}

// validate AidaDb on given path pathToDb
func validate(pathToDb, logLevel string) ([]byte, error) {
	v := newDbValidator(pathToDb, logLevel)

	if err := v.iterate(); err != nil {
		return nil, fmt.Errorf("cannot iterate over aida-db; %v", err)
	}

	sum := v.hasher.Sum(nil)

	v.log.Noticef("AidaDb MD5 sum: %v", hex.EncodeToString(sum))

	return sum, nil
}

// iterate calls doIterate func for each prefix inside metadata
func (v *validator) iterate() error {
	var (
		err error
		now time.Time
	)

	now = time.Now()

	v.log.Notice("Iterating over Stage 1 Substate...")
	if err = v.doIterate(substate.Stage1SubstatePrefix); err != nil {
		return err
	}

	v.log.Infof("Stage 1 Substate took %v.", time.Since(now).Round(1*time.Second))

	now = time.Now()

	v.log.Notice("Iterating over Substate Alloc...")
	if err = v.doIterate(substate.SubstateAllocPrefix); err != nil {
		return err
	}

	v.log.Infof("Substate Alloc took %v.", time.Since(now).Round(1*time.Second))

	now = time.Now()

	v.log.Notice("Iterating over Destroyed Accounts...")
	if err = v.doIterate(substate.DestroyedAccountPrefix); err != nil {
		return err
	}

	v.log.Infof("Destroyed Accounts took %v.", time.Since(now).Round(1*time.Second))

	now = time.Now()

	v.log.Notice("Iterating over Stage 1 Code...")
	if err = v.doIterate(substate.Stage1CodePrefix); err != nil {
		return err
	}

	v.log.Infof("Stage 1 Code took %v.", time.Since(now).Round(1*time.Second))

	v.log.Noticef("Total time elapsed: %v", time.Since(v.start).Round(1*time.Second))

	return nil

}

// doIterate over all key/value inside AidaDb and create md5 hash for each par for given prefix
func (v *validator) doIterate(prefix string) error {
	iter := v.db.NewIterator([]byte(prefix), nil)

	var (
		n, written int
		err        error
	)

	for iter.Next() {
		// we must make sure we wrote all data
		for written < len(iter.Key()) {
			n, err = v.hasher.Write(iter.Key())
			written += n
		}

		// reset check
		written = 0

		// we must make sure we wrote all data
		for written < len(iter.Value()) {
			n, err = v.hasher.Write(iter.Value())
			written += n
		}

		// reset check
		written = 0

		// checking error for both key and value just slows down the program. Check the error at the end only once
		if err != nil {
			return fmt.Errorf("cannot write; %v", err)
		}
	}

	if iter.Error() != nil {
		return iter.Error()
	}

	return nil
}
