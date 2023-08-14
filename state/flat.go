package state

import (
	"fmt"
	"math/big"

	substate "github.com/Fantom-foundation/Substate"
	"github.com/Fantom-foundation/go-opera-flat/flat"
	"github.com/Fantom-foundation/go-opera-flat/gossip/evmstore/state"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/ethdb"
)

func MakeFlatStateDB(directory, variant string, rootHash common.Hash) (s StateDB, err error) {
	var db ethdb.Database

	switch variant {
	case "": // = default option
		fallthrough
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
		err = fmt.Errorf("unknown variant: %v", variant)
		return
	}

	fs := &flatStateDB{
		db:        flat.NewDatabase(db),
		stateRoot: rootHash,
	}

	// initialize stateDB
	fs.openStateDB()
	s = fs
	return
}

type flatStateDB struct {
	db        state.Database
	stateRoot common.Hash
	*state.StateDB
}

// openStateDB creates a new statedb from an existing geth database
func (s *flatStateDB) openStateDB() error {
	var err error
	s.StateDB, err = state.New(s.stateRoot, s.db)
	return err
}

func (s *flatStateDB) BeginTransaction(number uint32) {
	// ignored
}

func (s *flatStateDB) EndTransaction() {
	s.Finalise(true)
}

func (s *flatStateDB) BeginBlock(number uint64) {
	s.openStateDB()
}

func (s *flatStateDB) EndBlock() {
	var err error
	//commit at the end of a block
	s.stateRoot, err = s.Commit(true)
	if err != nil {
		panic(fmt.Errorf("StateDB commit failed\n"))
	}
}

func (s *flatStateDB) BeginSyncPeriod(number uint64) {
	// ignored
}

func (s *flatStateDB) EndSyncPeriod() {
	// ignored
}

// PrepareSubstate initiates the state DB for the next transaction.
func (s *flatStateDB) PrepareSubstate(*substate.SubstateAlloc, uint64) {
	// ignored
	return
}

func (s *flatStateDB) GetSubstatePostAlloc() substate.SubstateAlloc {
	// ignored
	return substate.SubstateAlloc{}
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

func (s *flatStateDB) GetMemoryUsage() *MemoryUsage {
	// not supported yet
	return &MemoryUsage{uint64(0), nil}
}

func (s *flatStateDB) StartBulkLoad(block uint64) BulkLoad {
	s.BeginBlock(block)
	return &flatBulkLoad{db: s}
}

func (s *flatStateDB) GetArchiveState(block uint64) (StateDB, error) {
	return nil, fmt.Errorf("archive states are not (yet) supported by this DB implementation")
}

func (s *flatStateDB) GetShadowDB() StateDB {
	return nil
}

// For priming initial state of stateDB
type flatBulkLoad struct {
	db *flatStateDB
}

func (l *flatBulkLoad) CreateAccount(addr common.Address) {
	l.db.CreateAccount(addr)
}

func (l *flatBulkLoad) SetBalance(addr common.Address, value *big.Int) {
	old := l.db.GetBalance(addr)
	value = value.Sub(value, old)
	l.db.AddBalance(addr, value)
}

func (l *flatBulkLoad) SetNonce(addr common.Address, nonce uint64) {
	l.db.SetNonce(addr, nonce)
}

func (l *flatBulkLoad) SetState(addr common.Address, key common.Hash, value common.Hash) {
	l.db.SetState(addr, key, value)
}

func (l *flatBulkLoad) SetCode(addr common.Address, code []byte) {
	l.db.SetCode(addr, code)
}

func (l *flatBulkLoad) Close() error {
	l.db.EndBlock()
	_, err := l.db.Commit(false)
	return err
}