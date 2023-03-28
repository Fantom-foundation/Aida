package state

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"path/filepath"

	"github.com/Fantom-foundation/go-opera-fvm/erigon"
	"github.com/Fantom-foundation/go-opera-fvm/evmcore"
	"github.com/Fantom-foundation/go-opera-fvm/gossip/evmstore/erigonstate"

	substate "github.com/Fantom-foundation/Substate"
	"github.com/ethereum/go-ethereum/common"

	"github.com/ledgerwatch/erigon-lib/kv"
	"github.com/ledgerwatch/erigon-lib/kv/memdb"
	estate "github.com/ledgerwatch/erigon/core/state"
)

func MakeErigonStateDB(directory, variant string, rootHash common.Hash, chainKV kv.RwDB) (s StateDB, err error) {
	switch variant {
	case "go-memory":
		chainKV = memdb.New()
	case "go-erigon":
	default:
		err = fmt.Errorf("unkown variant: %v", variant)
		return
	}

	es := &erigonStateDB{
		chainKV:   chainKV,
		stateRoot: rootHash,
		directory: directory,
	}

	// initialize stateDB
	// or use NewPlainState
	es.BeginBlockApply()
	s = es
	return
}

type erigonStateDB struct {
	chainKV   kv.RwDB
	stateRoot common.Hash
	*evmcore.ErigonAdapter
	runVMStarted bool
	directory    string
	from, to     uint64
}

// BeginBlockApply creates a new statedb from an existing geth database
func (s *erigonStateDB) BeginBlockApply() error {
	var err error
	s.ErigonAdapter = evmcore.NewErigonAdapter(erigonstate.NewWithChainKV(s.chainKV))
	return err
}

func (s *erigonStateDB) BeginTransaction(number uint32) {
	// ignored
}

func (s *erigonStateDB) EndTransaction() {
	// ignored
}

func (s *erigonStateDB) BeginBlock(number uint64) {
	s.from = number
}

func (s *erigonStateDB) SetTxBlock(number uint64) {
	s.to = number
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

	tx, err := s.chainKV.BeginRw(context.Background())
	if err != nil {
		panic(err)
	}

	defer tx.Rollback()

	if err := s.processEndBlock(tx); err != nil {
		panic(err)
	}
}

func (s *erigonStateDB) processEndBlock(tx kv.RwTx) error {
	blockWriter := estate.NewPlainStateWriter(tx, tx, s.from)

	// flush pending changes into erigon plain state
	if err := s.ErigonAdapter.CommitBlock(blockWriter); err != nil {
		return err
	}

	if err := blockWriter.WriteChangeSets(); err != nil {
		return err
	}

	switch {
	case s.to == 0:
		//log.Println("EndBlock cleanly", "from", s.from, "to", s.to)
		// cleanly
		if err := erigon.PromoteHashedStateCleanly(tx, filepath.Join(s.directory, "hashedstate")); err != nil {
			return err
		}

		if _, err := erigon.RegenerateIntermediateHashes("IH", tx, filepath.Join(s.directory, "IH"), false); err != nil {
			return err
		}
	case s.to > 0 && s.to > s.from:
		// incrementally
		//log.Println("EndBlock incrementally", "from", s.from, "to", s.to)
		if err := erigon.PromoteHashedStateIncrementally("hashedstate", s.from, s.to, filepath.Join(s.directory, "hashedstate"), tx, nil); err != nil {
			return err
		}

		if _, err := erigon.IncrementIntermediateHashes("IH", tx, s.from, s.to, filepath.Join(s.directory, "IH"), false, nil); err != nil {
			return err
		}
	}

	return tx.Commit()
}

// TODO think about hashedstate and intermediatehashes
func (s *erigonStateDB) Commit(_ bool) (common.Hash, error) {
	tx, err := s.chainKV.BeginRw(context.Background())
	if err != nil {
		return common.Hash{}, err
	}

	defer tx.Rollback()

	blockWriter := estate.NewPlainStateWriter(tx, tx, s.from)
	//blockWriter := estate.NewPlainStateWriterNoHistory(tx)

	// flush pending changes into erigon plain state
	if err := s.ErigonAdapter.CommitBlock(blockWriter); err != nil {
		return common.Hash{}, err
	}

	if err := tx.Commit(); err != nil {
		return common.Hash{}, err
	}

	return common.Hash{}, nil
}

func (s *erigonStateDB) BeginEpoch(number uint64) {
	// ignored
}

func (s *erigonStateDB) EndEpoch() {
	tx, err := s.chainKV.BeginRw(context.Background())
	if err != nil {
		panic(err)
	}

	defer tx.Rollback()

	root, err := erigon.CalcRoot("", tx)
	if err != nil {
		panic(err)
	}
	s.stateRoot = common.Hash(root)
	log.Println("EndEpoch erigon stateRoot: ", s.stateRoot.Hex())
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

// TODO include s.EndBlock or not
// Close requests the StateDB to flush all its content to secondary storage and shut down.
// After this call no more operations will be allowed on the state.
func (s *erigonStateDB) Close() error {
	// flush changes to erigon db
	//s.EndBlock()

	// close underlying MDBX
	s.chainKV.Close()
	return nil
}

func (s *erigonStateDB) GetMemoryUsage() *MemoryUsage {
	// not supported yet
	return nil
}

// For priming initial state of stateDB
type erigonBulkLoad struct {
	db      *erigonStateDB
	num_ops int64
}

func (s *erigonStateDB) StartBulkLoad() BulkLoad {
	return &erigonBulkLoad{db: s}
}

func (l *erigonBulkLoad) CreateAccount(addr common.Address) {
	l.db.CreateAccount(addr)
	l.digest()
}

func (l *erigonBulkLoad) SetBalance(addr common.Address, value *big.Int) {
	old := l.db.GetBalance(addr)
	value = value.Sub(value, old)
	l.db.AddBalance(addr, value)
	l.digest()
}

func (l *erigonBulkLoad) SetNonce(addr common.Address, nonce uint64) {
	l.db.SetNonce(addr, nonce)
	l.digest()
}

func (l *erigonBulkLoad) SetState(addr common.Address, key common.Hash, value common.Hash) {
	l.db.SetState(addr, key, value)
	l.digest()
}

func (l *erigonBulkLoad) SetCode(addr common.Address, code []byte) {
	l.db.SetCode(addr, code)
	l.digest()
}

func (l *erigonBulkLoad) Close() error {
	l.db.EndBlock()
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
