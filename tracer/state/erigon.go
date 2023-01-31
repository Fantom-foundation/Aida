package state

import (
	"fmt"
	"math/big"
	"context"

	"github.com/Fantom-foundation/go-opera-fvm-erigon/erigon"
	"github.com/Fantom-foundation/go-opera-fvm-erigon/evmcore"
	"github.com/Fantom-foundation/go-opera-fvm-erigon/gossip/evmstore/erigonstate"
	"github.com/Fantom-foundation/go-opera-fvm-erigon/logger"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/substate"

	"github.com/ledgerwatch/erigon-lib/kv"
	"github.com/ledgerwatch/erigon-lib/kv/memdb"
	estate "github.com/ledgerwatch/erigon/core/state"
)

func MakeErigonStateDB(directory, variant string) (s StateDB, err error) {
	var chainKV kv.RwDB

	switch variant {
	case "go-memory":
		chainKV = memdb.New()
	case "go-erigon":
		chainKV = erigon.MakeChainDatabase(logger.New("chain-kv"), kv.ChainDB, 0)
	default:
		err = fmt.Errorf("unkown variant: %v", variant)
		return
	}

	es := &erigonStateDB{
		//db:        flat.NewDatabase(db),
		chainKV:   chainKV,
		stateRoot: common.Hash{},
	}

	// initialize stateDB
	es.BeginBlockApply()
	s = es
	return
}

type erigonStateDB struct {
	chainKV   kv.RwDB
	rwTx      kv.RwTx
	stateRoot common.Hash
	evmcore.StateDB
	block uint64
}

// BeginBlockApply creates a new statedb from an existing geth database
func (s *erigonStateDB) BeginBlockApply() error {
	s.StateDB = evmcore.NewErigonAdapter(erigonstate.New())
	return nil
}

func (s *erigonStateDB) BeginTransaction(number uint32) {
	// ignored
}

func (s *erigonStateDB) EndTransaction() {
	// ignored
}

func (s *erigonStateDB) BeginBlock(number uint64) {
	// ignored
}
// TODO in erigon state root  is computed every epoch not every block
// decide whether to compute it every block or not 
func (s *erigonStateDB) EndBlock() {
	// TODO check that s.rwTx not nil
	blockWriter := estate.NewPlainStateWriter(s.rwTx, s.rwTx, s.block)

	// flush pending changes into erigon db
	if err := s.StateDB.CommitBlock(blockWriter); err != nil {
		panic(err)
	}

	// write changesets
	if err := blockWriter.WriteChangeSets(); err != nil {
		panic(err)
	}

	// write history index
	if err := blockWriter.WriteHistory(); err != nil {
		panic(err)
	}

	// convert kv.Plainstate into Hashedstate. Required for later stateroot computation
	if err := erigon.PromoteHashedStateIncrementally("HashedState", s.block-1, s.block, s.rwTx, nil); err != nil {
		panic(err)
	}

	// setting sealing argument to true enables state root computation
	root, err := erigon.IncrementIntermediateHashes("IH", s.rwTx, s.block-1, s.block, true, nil)
	if err != nil {
		panic(err)
	}

	s.stateRoot = common.Hash(root)
}

func (s *erigonStateDB) BeginEpoch(number uint64) {
	// ignored
}

func (s *erigonStateDB) EndEpoch() {
	// ignored
}

// PrepareSubstate initiates the state DB for the next transaction.
func (s *erigonStateDB) PrepareSubstate(*substate.SubstateAlloc) {
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
	rwTx, err := s.chainKV.BeginRw(context.Background())
	if err != nil {
		return err
	}

	defer func() {
		rwTx.Rollback()
		s.chainKV.Close()
	}()

	s.rwTx = rwTx

	// flush changes to erigon db
	s.StateDB.EndBlock()

	// commit rwTx
	if err := s.rwTx.Commit(); err != nil {
		return err
	}

	return nil
}

func (s *erigonStateDB) GetMemoryUsage() *MemoryUsage {
	// not supported yet
	return nil
}

func (s *erigonStateDB) StartBulkLoad() BulkLoad {
	// open erigon rwTx transaction
	rwTx, err := s.chainKV.BeginRw(context.Background())
	if err != nil {
		panic(err)
	}

	s.rwTx = rwTx
	return &erigonBulkLoad{db: s}
}

// For priming initial state of stateDB
type erigonBulkLoad struct {
	db      *erigonStateDB
	num_ops int64
}

func (l *erigonBulkLoad) CreateAccount(addr common.Address) {
	l.db.StateDB.CreateAccount(addr)
	l.digest()
}

func (l *erigonBulkLoad) SetBalance(addr common.Address, value *big.Int) {
	old := l.db.StateDB.GetBalance(addr)
	value = value.Sub(value, old)
	l.db.StateDB.AddBalance(addr, value)
	l.digest()
}

func (l *erigonBulkLoad) SetNonce(addr common.Address, nonce uint64) {
	l.db.StateDB.SetNonce(addr, nonce)
	l.digest()
}

func (l *erigonBulkLoad) SetState(addr common.Address, key common.Hash, value common.Hash) {
	l.db.StateDB.SetState(addr, key, value)
	l.digest()
}

func (l *erigonBulkLoad) SetCode(addr common.Address, code []byte) {
	l.db.StateDB.SetCode(addr, code)
	l.digest()
}

func (l *erigonBulkLoad) Close() error {
	defer l.db.rwTx.Rollback()

	l.db.StateDB.EndBlock()

	// commit rwTx
	return l.db.rwTx.Commit()
}

func (l *erigonBulkLoad) digest() {
	// Call EndBlock every 1M insert operation.
	l.num_ops++
	if l.num_ops%(1000*1000) != 0 {
		return
	}
	l.db.StateDB.EndBlock()
}
