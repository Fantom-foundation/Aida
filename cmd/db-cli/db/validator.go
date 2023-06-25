package db

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"github.com/Fantom-foundation/Aida/logger"
	substate "github.com/Fantom-foundation/Substate"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/op/go-logging"
	"hash"
)

type validator struct {
	db     ethdb.Database
	hasher hash.Hash
	closed chan any
	log    *logging.Logger
}

func newDbValidator(pathToDb, logLevel string) *validator {
	l := logger.NewLogger(logLevel, "Db-Validator")

	db, err := rawdb.NewLevelDBDatabase(pathToDb, 1024, 100, "profiling", true)
	if err != nil {
		l.Fatalf("cannot create new db; %v", err)
	}

	return &validator{
		db:     db,
		hasher: md5.New(),
		closed: make(chan any, 1),
		log:    l,
	}
}

func validate(pathToDb, logLevel string) ([]byte, error) {
	v := newDbValidator(pathToDb, logLevel)

	if err := v.iterate(); err != nil {
		return nil, fmt.Errorf("cannot iterate over aida-db; %v", err)
	}

	sum := v.hasher.Sum(nil)

	v.log.Noticef("AidaDb MD5 sum: %v", hex.EncodeToString(sum))

	return sum, nil
}

func (v *validator) iterate() error {
	var err error

	v.log.Notice("Iterating over Stage 1 Substate...")
	if err = v.doIterate(substate.Stage1SubstatePrefix); err != nil {
		return err
	}

	v.log.Notice("Iterating over Substate Alloc...")
	if err = v.doIterate(substate.SubstateAllocPrefix); err != nil {
		return err
	}

	v.log.Notice("Iterating over Destroyed Accounts...")
	if err = v.doIterate(substate.DestroyedAccountPrefix); err != nil {
		return err
	}

	v.log.Notice("Iterating over Stage 1 Code...")
	if err = v.doIterate(substate.Stage1CodePrefix); err != nil {
		return err
	}

	return nil

}

func (v *validator) doIterate(prefix string) error {
	iter := v.db.NewIterator([]byte(prefix), substate.BlockToBytes(0))

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
