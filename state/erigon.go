package state

import (
	"context"
	"fmt"
	"math/big"
	//"path"
	"errors"
	"log"
	"time"

	//"github.com/Fantom-foundation/go-opera-fvm/cmd/opera/launcher"
	"github.com/Fantom-foundation/go-opera-fvm/erigon"
	"github.com/Fantom-foundation/go-opera-fvm/evmcore"
	"github.com/Fantom-foundation/go-opera-fvm/gossip/evmstore/erigonstate"

	substate "github.com/Fantom-foundation/Substate"
	"github.com/ethereum/go-ethereum/common"

	"github.com/ledgerwatch/erigon-lib/kv"
	"github.com/ledgerwatch/erigon-lib/kv/memdb"
	estate "github.com/ledgerwatch/erigon/core/state"
)

func MakeErigonStateDB(directory, variant string, rootHash common.Hash, tx kv.RwTx) (s StateDB, err error) {
	log.Println("MakeErigonStateDB", "directory", directory, "variant", variant)
	switch variant {
	case "go-memory":
		db := memdb.New()
		tx, err = db.BeginRw(context.Background())
		if err != nil {
			return nil, err
		}
	case "go-erigon":
	default:
		err = fmt.Errorf("unkown variant: %v", variant)
		return
	}

	es := &erigonStateDB{
		rwTx:      tx,
		stateRoot: rootHash,
	}

	// initialize stateDB
	// or use NewPlainState
	es.ErigonAdapter = evmcore.NewErigonAdapter(erigonstate.NewWithStateReader(estate.NewPlainStateReader(tx)))
	s = es
	return
}

type erigonStateDB struct {
	rwTx      kv.RwTx
	stateRoot common.Hash
	*evmcore.ErigonAdapter
	block uint64
}

// BeginBlockApply creates a new statedb from an existing geth database
func (s *erigonStateDB) BeginBlockApply() error {
	return errors.New("ignored")
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

func (s *erigonStateDB) GetArchiveState(block uint64) (StateDB, error) {
	return nil, fmt.Errorf("archive states are not (yet) supported by this DB implementation")
}

func (s *erigonStateDB) AddPreimage(common.Hash, []byte) {
	// ignored
	panic("AddPreimage not implemented")
}

func (s *erigonStateDB) ForEachStorage(common.Address, func(common.Hash, common.Hash) bool) error {
	// ignored
	panic("ForEachStorage not implemented")
}

// TODO in erigon state root  is computed every epoch not every block
// decide whether to compute it every block or not
// TODO add a flag to enable erigon history writing, skip it for now, use NewPlainStateWriterNoHistory
// TODO add an option to use DbStateWriter instead of estate.NewPlainStateWriterNoHistory(rwTx) to speed up an executution
// TODO add caching
func (s *erigonStateDB) EndBlock() {

	
	blockWriter := estate.NewPlainStateWriterNoHistory(s.rwTx)

	// flush pending changes into erigon db
	if err := s.ErigonAdapter.CommitBlock(blockWriter); err != nil {
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
	log.Println("erigonStateDB.EndBlock", "\tErigon State root:  \t", s.stateRoot.Hex())

	log.Println("erigonStateDB.EndBlock, commmit erigon transaction")
	if err := s.rwTx.Commit(); err != nil {
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
	// flush changes to erigon db
	//s.EndBlock()
	// opem rwTx transaction
	blockWriter := estate.NewPlainStateWriterNoHistory(s.rwTx)

	// flush pending changes into erigon db
	return s.StateDB.CommitBlock(blockWriter)
}

func (s *erigonStateDB) GetMemoryUsage() *MemoryUsage {
	// not supported yet
	return nil
}

func (s *erigonStateDB) StartBulkLoad() BulkLoad {
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
	old := l.db.ErigonAdapter.GetBalance(addr)
	value = value.Sub(value, old)
	l.db.ErigonAdapter.AddBalance(addr, value)
	l.digest()
}

func (l *erigonBulkLoad) SetNonce(addr common.Address, nonce uint64) {
	l.db.ErigonAdapter.SetNonce(addr, nonce)
	l.digest()
}

func (l *erigonBulkLoad) SetState(addr common.Address, key common.Hash, value common.Hash) {
	l.db.ErigonAdapter.SetState(addr, key, value)
	l.digest()
}

func (l *erigonBulkLoad) SetCode(addr common.Address, code []byte) {
	l.db.ErigonAdapter.SetCode(addr, code)
	l.digest()
}

func (l *erigonBulkLoad) Close() error {
	log.Println("(l *erigonBulkLoad) Close()")
	start := time.Now()
	l.db.EndBlock()
	sec := time.Since(start).Seconds()
	log.Printf("\tElapsed time: %.2f s\n", sec)
	return nil
}

func (l *erigonBulkLoad) digest() {
	// Call EndBlock every 1M insert operation.
	l.num_ops++
	if l.num_ops%(1000*1000) != 0 {
		return
	}
	//l.db.EndBlock()
	//l.Close()
}
