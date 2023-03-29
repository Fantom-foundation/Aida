package state

import (
	"context"
	"fmt"
	"math/big"

	substate "github.com/Fantom-foundation/Substate"
	"github.com/Fantom-foundation/go-opera-erigon/erigon"
	"github.com/Fantom-foundation/go-opera-erigon/evmcore"
	state "github.com/Fantom-foundation/go-opera-erigon/gossip/evmstore/erigonstate"
	"github.com/Fantom-foundation/go-opera-erigon/gossip/evmstore/ethdb"
	"github.com/Fantom-foundation/go-opera-erigon/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ledgerwatch/erigon-lib/kv"
	"github.com/ledgerwatch/erigon-lib/kv/memdb"
	estate "github.com/ledgerwatch/erigon/core/state"
)

func MakeErigonStateDB(directory, variant string, rootHash common.Hash) (s StateDB, err error) {
	var db kv.RwDB

	switch variant {
	case "go-memory":
		db = memdb.New()
	case "go-mdbx":
		db = erigon.MakeChainDatabase(logger.New("chain-kv"), directory)
	default:
		err = fmt.Errorf("unkown variant: %v", variant)
		return
	}

	es := &erigonStateDB{
		db:        ethdb.NewObjectDatabase(db),
		stateRoot: rootHash,
	}

	// initialize stateDB
	es.BeginBlockApply()
	s = fs
	return
}

type erigonStateDB struct {
	db        erigonethdb.Database
	rwTx      kv.RwTx
	stateRoot common.Hash
	blockN    uint64
	*state.StateDB
}

// BeginBlockApply creates a new statedb from an existing geth database
func (s *erigonStateDB) BeginBlockApply() (err error) {
	s.rwTx, err = s.db.RwKV().BeginRw(context.Background())
	if err != nil {
		return
	}

	s.StateDB = evmcore.NewErigonAdapter(state.NewWithStateReader(estate.NewPlainStateReader(rwTx)))
	return
}

func (s *erigonStateDB) BeginTransaction(number uint32) {
	// ignored
}

func (s *erigonStateDB) EndTransaction() {
	// ignored
}

func (s *erigonStateDB) BeginBlock(number uint64) {
	s.blockN = number
}

func (s *erigonStateDB) EndBlock() {
	from, to := s.blockN-1, s.blockN
		
	blockWriter := estate.NewPlainStateWriter(rwTx, rwTx, to)
	
	err := s.StateDB.CommitBlock(blockWriter)
	if err != nil {
		panic(err)
	}

	err = blockWriter.WriteChangeSets()
	if  err != nil {
		panic(err)
	}

	err = blockWriter.WriteHistory()
	if  err != nil {
		panic(err)
	}

	err = erigon.PromoteHashedStateIncrementally("HashedState", from, to, s.rwTx, nil)
	if err != nil {
		panic(err)
	}
	
	s.stateRoot, err := erigon.IncrementIntermediateHashes("IH", rwTx, from, to, false, nil)
	if err != nil {
		panic(err)
	}

}

func (s *erigonStateDB) BeginEpoch(number uint64) {
	// ignored
}

func (s *erigonStateDB) EndEpoch() {
	// ignored
}

// PrepareSubstate initiates the state DB for the next transaction.
func (s *erigonStateDB) PrepareSubstate(*substate.SubstateAlloc, uint64) {
	// ignored
	return
}

func (s *erigonStateDB) GetSubstatePostAlloc() substate.SubstateAlloc {
	// ignored
	return substate.SubstateAlloc{}
}

// Close requests the StateDB to flush all its content to secondary storage and shut down.
// After this call no more operations will be allowed on the state.
func (s *erigonStateDB) Close() error {
	err := s.rwTx.Commit()
	if err != nil {
		return err
	}
	
	err = s.db.Close()
	if err != nil {
		return err
	}

	return nil
}

func (s *erigonStateDB) GetMemoryUsage() *MemoryUsage {
	// not supported yet
	return nil
}

func (s *erigonStateDB) StartBulkLoad() BulkLoad {
	return &flatBulkLoad{db: s}
}

func (s *erigonStateDB) GetArchiveState(block uint64) (StateDB, error) {
	return nil, fmt.Errorf("archive states are not (yet) supported by this DB implementation")
}

// For priming initial state of stateDB
type flatBulkLoad struct {
	db      *erigonStateDB
	num_ops int64
}

func (l *flatBulkLoad) CreateAccount(addr common.Address) {
	l.db.CreateAccount(addr)
	l.digest()
}

func (l *flatBulkLoad) SetBalance(addr common.Address, value *big.Int) {
	old := l.db.GetBalance(addr)
	value = value.Sub(value, old)
	l.db.AddBalance(addr, value)
	l.digest()
}

func (l *flatBulkLoad) SetNonce(addr common.Address, nonce uint64) {
	l.db.SetNonce(addr, nonce)
	l.digest()
}

func (l *flatBulkLoad) SetState(addr common.Address, key common.Hash, value common.Hash) {
	l.db.SetState(addr, key, value)
	l.digest()
}

func (l *flatBulkLoad) SetCode(addr common.Address, code []byte) {
	l.db.SetCode(addr, code)
	l.digest()
}

func (l *flatBulkLoad) Close() error {
	l.db.EndBlock()
	_, err := l.db.Commit(false)
	return err
}

func (l *flatBulkLoad) digest() {
	// Call EndBlock every 1M insert operation.
	l.num_ops++
	if l.num_ops%(1000*1000) != 0 {
		return
	}
	l.db.EndBlock()
}
