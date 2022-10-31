package state

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	rawdb "github.com/ethereum/go-ethereum/core/rawdb"
	geth "github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/substate"
)

func MakeGethStateDB(directory, variant string) (StateDB, error) {
	if variant != "" {
		return nil, fmt.Errorf("unkown variant: %v", variant)
	}
	return OpenGethStateDB(directory, common.Hash{})
}

func OpenGethStateDB(directory string, root_hash common.Hash) (StateDB, error) {
	const cache_size = 512
	const file_handle = 128
	ldb, err := rawdb.NewLevelDBDatabase(directory, cache_size, file_handle, "", false)
	if err != nil {
		return nil, err
	}
	db, err := geth.New(root_hash, geth.NewDatabase(ldb), nil)
	if err != nil {
		return nil, err
	}
	return &gethStateDb{db}, nil
}

type gethStateDb struct {
	db BasicStateDB
}

func (s *gethStateDb) CreateAccount(addr common.Address) {
	s.db.CreateAccount(addr)
}

func (s *gethStateDb) Exist(addr common.Address) bool {
	return s.db.Exist(addr)
}

func (s *gethStateDb) Empty(addr common.Address) bool {
	return s.db.Empty(addr)
}

func (s *gethStateDb) Suicide(addr common.Address) bool {
	return s.db.Suicide(addr)
}

func (s *gethStateDb) HasSuicided(addr common.Address) bool {
	return s.db.HasSuicided(addr)
}

func (s *gethStateDb) GetBalance(addr common.Address) *big.Int {
	return s.db.GetBalance(addr)
}

func (s *gethStateDb) AddBalance(addr common.Address, value *big.Int) {
	s.db.AddBalance(addr, value)
}

func (s *gethStateDb) SubBalance(addr common.Address, value *big.Int) {
	s.db.SubBalance(addr, value)
}

func (s *gethStateDb) GetNonce(addr common.Address) uint64 {
	return s.db.GetNonce(addr)
}

func (s *gethStateDb) SetNonce(addr common.Address, value uint64) {
	s.db.SetNonce(addr, value)
}

func (s *gethStateDb) GetCommittedState(addr common.Address, key common.Hash) common.Hash {
	return s.db.GetCommittedState(addr, key)
}

func (s *gethStateDb) GetState(addr common.Address, key common.Hash) common.Hash {
	return s.db.GetState(addr, key)
}

func (s *gethStateDb) SetState(addr common.Address, key common.Hash, value common.Hash) {
	s.db.SetState(addr, key, value)
}

func (s *gethStateDb) GetCodeHash(addr common.Address) common.Hash {
	return s.db.GetCodeHash(addr)
}

func (s *gethStateDb) GetCode(addr common.Address) []byte {
	return s.db.GetCode(addr)
}

func (s *gethStateDb) Snapshot() int {
	return s.db.Snapshot()
}

func (s *gethStateDb) RevertToSnapshot(id int) {
	s.db.RevertToSnapshot(id)
}

func (s *gethStateDb) Finalise(deleteEmptyObjects bool) {
	// IntermediateRoot implicitly calls Finalise but also commits changes.
	// Without calling this, no changes are ever committed.
	state, ok := s.db.(*geth.StateDB)
	if ok {
		// Until we have an initial world state, we do not delete empty objects.
		// This would remove changes to unknown accounts, and thus not commit
		// anything. TODO: re-evaluate once world state is available.
		//state.IntermediateRoot(deleteEmptyObjects)
		state.IntermediateRoot(false)
	} else {
		s.db.Finalise(deleteEmptyObjects)
	}
}

func (s *gethStateDb) PrepareSubstate(substate *substate.SubstateAlloc) {
	// ignored
}

func (s *gethStateDb) GetSubstatePostAlloc() substate.SubstateAlloc {
	return s.db.GetSubstatePostAlloc()
}

func (s *gethStateDb) Close() error {
	// Skip closing if implementation is not Geth based.
	state, ok := s.db.(*geth.StateDB)
	if !ok {
		return nil
	}
	// Commit data to trie.
	hash, err := state.Commit(true)
	if err != nil {
		return err
	}

	// Close underlying trie caching intermediate results.
	db := state.Database().TrieDB()
	if err := db.Commit(hash, true, nil); err != nil {
		return err
	}

	// Close underlying LevelDB instance.
	return db.DiskDB().Close()
}
