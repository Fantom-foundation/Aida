// Package db implements database interfaces for the world state manager.
package db

import (
	"fmt"
	"github.com/Fantom-foundation/Aida-Testing/world-state/db/kvdb2ethdb"
	"github.com/Fantom-foundation/lachesis-base/kvdb"
	"github.com/Fantom-foundation/lachesis-base/kvdb/leveldb"
	"github.com/Fantom-foundation/lachesis-base/kvdb/nokeyiserr"
	"github.com/Fantom-foundation/lachesis-base/kvdb/pebble"
	"github.com/Fantom-foundation/lachesis-base/kvdb/table"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"log"
)

// Connect opens the source database based on provided path and DB type.
func connect(dbType string, path string) (kvdb.IterableDBProducer, error) {
	// we support both LevelDB and Pebble DB as the state trie source
	switch dbType {
	case "ldb":
		return leveldb.NewProducer(path, func(string) (int, int) {
			return 100 * opt.MiB, 1000
		}), nil
	case "pbl":
		pebble.NewProducer(path, func(string) (int, int) {
			return 100 * opt.MiB, 1000
		})
	}

	// invalid DB type
	return nil, fmt.Errorf("invalid DB type; expected (ldb, pbl), %s given", dbType)
}

// Connect specified database.
func Connect(dbType string, dbPath string, dbName string) (kvdb.Store, error) {
	// connect the KV database
	kv, err := connect(dbType, dbPath)
	if err != nil {
		return nil, err
	}

	// try to open the Store
	store, err := kv.OpenDB(dbName)
	if err != nil {
		return nil, err
	}

	return store, nil
}

// OpenStateTrie opens the EVM state trie on the provided DB connection.
func OpenStateTrie(store kvdb.Store) state.Database {
	// evm data are stored under prefix M
	evmDB := table.New(store, []byte(("M")))
	wrappedEvmDB := rawdb.NewDatabase(kvdb2ethdb.Wrap(nokeyiserr.Wrap(evmDB)))

	return state.NewDatabaseWithConfig(wrappedEvmDB, &trie.Config{})
}

// OpenBlockEpochState opens the Opera block/epoch state database.
func OpenBlockEpochState(store kvdb.Store) kvdb.Store {
	return table.New(store, []byte(("D")))
}

// MustCloseStore closes opened store without raising any error.
func MustCloseStore(s kvdb.Store) {
	if s != nil {
		err := s.Close()
		if err != nil {
			log.Printf("could not close store; %s\n", err.Error())
		}
	}
}

// OpenBlocks opens the Opera blocks database.
func OpenBlocks(store kvdb.Store) kvdb.Store {
	return table.New(store, []byte(("b")))
}
