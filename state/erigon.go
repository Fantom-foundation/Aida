package state

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math/big"
	"path/filepath"

	"github.com/Fantom-foundation/go-opera-erigon/cmd/opera/launcher"
	"github.com/Fantom-foundation/go-opera-erigon/evmcore"
	"github.com/Fantom-foundation/go-opera-erigon/gossip/evmstore/erigonstate"
	"github.com/Fantom-foundation/go-opera-erigon/gossip/evmstore/ethdb"

	substate "github.com/Fantom-foundation/Substate"
	"github.com/ethereum/go-ethereum/common"

	"github.com/ledgerwatch/erigon-lib/kv"
	estate "github.com/ledgerwatch/erigon/core/state"
	erigonethdb "github.com/ledgerwatch/erigon/ethdb"
	"github.com/ledgerwatch/erigon/ethdb/olddb"

	"github.com/c2h5oh/datasize"
	lru "github.com/hashicorp/golang-lru"
)

func MakeErigonStateDB(directory, variant string, rootHash common.Hash, batchLimit datasize.ByteSize, firstBlock, lastBlock uint64, appName string) (s StateDB, err error) {
	var kv kv.RwDB
	erigonDirectory := filepath.Join(directory, "erigon")
	// erigon go-memory variant is not compatible with erigon batch mode
	switch variant {
	case "": // = default option
		kv = launcher.InitChainKV(erigonDirectory)
	case "go-mdbx":
		kv = launcher.InitChainKV(erigonDirectory)
	default:
		err = fmt.Errorf("unknown variant: %v", variant)
		return
	}

	es := &erigonStateDB{
		db:         ethdb.NewObjectDatabase(kv),
		stateRoot:  rootHash,
		directory:  erigonDirectory,
		batchLimit: batchLimit,
		firstBlock: firstBlock,
		lastBlock:  lastBlock,
		appName:    appName,
	}

	// initialize stateDB
	// or use NewPlainState
	es.openStateDB()
	s = es
	return
}

type erigonStateDB struct {
	db        erigonethdb.Database
	stateRoot common.Hash
	*evmcore.ErigonAdapter
	stateWriter estate.WriterWithChangeSets
	directory   string
	block       uint64
	firstBlock  uint64
	lastBlock   uint64
	appName     string
	batchMode   bool
	batchLimit  datasize.ByteSize
	tx          kv.RwTx
	batch       erigonethdb.DbWithPendingMutations
}

// BeginBlockApply creates a new statedb from an existing geth database
func (s *erigonStateDB) openStateDB() error {
	s.ErigonAdapter = evmcore.NewErigonAdapter(erigonstate.NewWithChainKV(s.db.RwKV()))
	s.batchMode = false
	return nil
}

// BeginBlockApplyBatch set up stateReader and stateWriter for erigon impl. It also creates new ErigonAdapter with new stateReader.
func (s *erigonStateDB) beginBlockApplyBatch() error {
	s.stateWriter = estate.NewPlainStateWriterNoHistory(s.batch)
	s.batchMode = true
	s.ErigonAdapter = evmcore.NewErigonAdapter(erigonstate.NewWithStateReader(estate.NewPlainStateReader(s.batch)))
	return nil
}

func (s *erigonStateDB) BeginTransaction(number uint32) {
	// ignored
}

func (s *erigonStateDB) EndTransaction() {
	if err := s.endTransaction(); err != nil {
		panic(err)
	}
}

func (s *erigonStateDB) endTransaction() error {
	if !s.batchMode {
		return nil
	}

	if err := s.commitBlockWithStateWriter(); err != nil {
		return err
	}

	return s.commitIfNeeded()
}

func (s *erigonStateDB) BeginBlock(number uint64) {
	s.block = number
	if s.appName != "trace" && s.block == s.firstBlock {
		log.Printf("run-vm: begin batch execution at block %d\n", s.block)
		err := s.beginRwTxBatch()
		if err != nil {
			panic(err)
		}
	}

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

// Endblock flushes changes either into batch or erigon database
func (s *erigonStateDB) EndBlock() {
	if err := s.endBlock(); err != nil {
		panic(err)
	}
}

func (s *erigonStateDB) endBlock() error {
	if !s.batchMode {
		return s.processEndBlock()
	}
	if err := s.commitBlockWithStateWriter(); err != nil {
		return err
	}

	if s.appName != "trace" && s.block == s.lastBlock {
		log.Printf("run-vm: finalize batch execution at block %d\n", s.block)
		return s.finalizeExecution()
	}

	return nil
}

func (s *erigonStateDB) processEndBlock() error {
	tx, err := s.db.RwKV().BeginRw(context.Background())
	if err != nil {
		return err
	}

	defer tx.Rollback()

	// flush pending changes into erigon plain state
	if err := s.ErigonAdapter.CommitBlock(estate.NewPlainStateWriterNoHistory(tx)); err != nil {
		return err
	}

	return tx.Commit()
}

func (s *erigonStateDB) commitBlockWithStateWriter() error {
	if s.stateWriter == nil {
		return errors.New("stateWriter is nil")
	}
	return s.ErigonAdapter.CommitBlock(s.stateWriter)
}

// TODO think about hashedstate and intermediatehashes
func (s *erigonStateDB) Commit(_ bool) (common.Hash, error) {
	return common.Hash{}, nil
}

func (s *erigonStateDB) BeginSyncPeriod(number uint64) {
	// ignored
}

func (s *erigonStateDB) EndSyncPeriod() {
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

	// close underlying MDBX
	s.db.RwKV().Close()
	return nil
}

func (s *erigonStateDB) GetMemoryUsage() *MemoryUsage {
	// not supported yet
	return nil
}

// BeginRwTxBatch begins erigon read/write transaction and batch
func (s *erigonStateDB) beginRwTxBatch() (err error) {
	s.tx, err = s.db.RwKV().BeginRw(context.Background())
	if err != nil {
		return
	}

	// start erigon batch execution
	s.newBatch()
	s.beginBlockApplyBatch()

	return
}

// newBatch creates new erigon batch
func (s *erigonStateDB) newBatch() {
	const lruDefaultSize = 1_000_000 // 56 MB

	whitelistedTables := []string{kv.Code, kv.ContractCode}

	contractCodeCache, err := lru.New(lruDefaultSize)
	if err != nil {
		panic(err)
	}

	// Contract code is unlikely to change too much, so let's keep it cached
	s.batch = olddb.NewHashBatch(s.tx, nil, s.directory, whitelistedTables, contractCodeCache)
}

// commitAndBegin commits current erigon batch and database transaction. It also begins new database transaction and batch
func (s *erigonStateDB) commitAndBegin() error {
	if err := s.commitBatchRwTx(); err != nil {
		return err
	}

	return s.beginRwTxBatch()
}

// commitBatchRwTx commits erigon batch and transaction
func (s *erigonStateDB) commitBatchRwTx() (err error) {
	err = s.batch.Commit()
	if err != nil {
		return
	}

	return s.tx.Commit()
}

// commitNeeded verifies whether batch size exceeds batch limit or not
func (s *erigonStateDB) commitNeeded() bool {
	return s.batch.BatchSize() >= int(s.batchLimit)
}

// commitIfNeeded commits batch and database transaction if batch size exceeds batch limit. It also begins new dabatase transaction and batch
func (s *erigonStateDB) commitIfNeeded() error {
	if s.appName == "trace" {
		log.Printf("traceReplaySubstateTask: batch.Commit")
		return s.commitBatchRwTx()
	}
	if s.commitNeeded() {
		log.Printf("run-vm: batch.Commit")
		err := s.commitAndBegin()
		if err != nil {
			return err
		}
	}
	return nil
}

// FinalizeExecution completes batch execution by commiting erigon batch and database transaction. It also unsets batchmode for db
func (s *erigonStateDB) finalizeExecution() error {
	if err := s.commitBatchRwTx(); err != nil {
		return err
	}
	// unset batchMode for db
	s.openStateDB()
	return nil
}

func (s *erigonStateDB) GetShadowDB() StateDB {
	return nil
}

// For priming initial state of stateDB
type erigonBulkLoad struct {
	db      *erigonStateDB
	num_ops int64
}

func (s *erigonStateDB) StartBulkLoad(_ uint64) BulkLoad {
	esDB := &erigonStateDB{
		db:         s.db,
		stateRoot:  s.stateRoot,
		directory:  s.directory,
		batchLimit: s.batchLimit,
		block:      s.block,
		firstBlock: s.firstBlock,
		lastBlock:  s.lastBlock,
		appName:    s.appName,
	}

	if err := esDB.beginRwTxBatch(); err != nil {
		panic(err)
	}

	return &erigonBulkLoad{db: esDB}
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

// Close flushes pending changes into batch. It also commits erigon batch and database transaction
func (l *erigonBulkLoad) Close() error {
	l.db.EndBlock()
	err := l.db.commitBatchRwTx()
	if err != nil {
		return err
	}
	return nil
}

func (l *erigonBulkLoad) digest() {
	// Call EndBlock every 1M insert operation.
	l.num_ops++
	if l.num_ops%(1000*1000) != 0 {
		return
	}
	l.db.EndBlock()
}
