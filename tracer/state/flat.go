package state

import (
	"fmt"

	"github.com/Fantom-foundation/go-opera-fvm/flat"
	"github.com/Fantom-foundation/go-opera-fvm/gossip/evmstore/state"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/substate"
)

func MakeFlatStateDB(directory, variant string) (s StateDB, err error) {
	var db ethdb.Database

	switch variant {
	case "go-memory":
		db = rawdb.NewMemoryDatabase()
	case "go-ldb":
		const cache_size = 512
		const file_handle = 128
		db, err = rawdb.NewLevelDBDatabase(directory, cache_size, file_handle, "", false)
		if err != nil {
			err = fmt.Errorf("Failed to create a new Level DB. %v", err)
			return
		}
	default:
		err = fmt.Errorf("unkown variant: %v", variant)
		return
	}

	fs := &flatStateDB{
		db: flat.NewDatabase(db),
	}
	if substate.RecordReplay {
		fs.substatePostAlloc = make(substate.SubstateAlloc)
	}

	s = fs
	return
}

type flatStateDB struct {
	db state.Database
	*state.StateDB
	substatePostAlloc substate.SubstateAlloc
}

// BeginBlockApply creates a new statedb from an existing geth database
func (s *flatStateDB) BeginBlockApply(root_hash common.Hash) error {
	state, err := state.New(root_hash, s.db)
	s.StateDB = state
	return err
}

// PrepareSubstate initiates the state DB for the next transaction.
func (s *flatStateDB) PrepareSubstate(*substate.SubstateAlloc) {
	return
}

func (s *flatStateDB) GetSubstatePostAlloc() substate.SubstateAlloc {
	// TODO: use or delete
	return s.substatePostAlloc
}

// Close requests the StateDB to flush all its content to secondary storage and shut down.
// After this call no more operations will be allowed on the state.
func (s *flatStateDB) Close() error {
	// Commit data to trie.
	hash, err := s.Commit(true)
	if err != nil {
		return err
	}
	// Close underlying trie caching intermediate results.
	db := s.Database().TrieDB()
	if err := db.Commit(hash, true, nil); err != nil {
		return err
	}
	// Close underlying LevelDB instance.
	err = db.DiskDB().Close()
	if err != nil {
		return err
	}

	return nil
}
