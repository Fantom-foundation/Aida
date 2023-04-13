package state

import (
	"fmt"
	"math/big"
	"strings"

	cc "github.com/Fantom-foundation/Carmen/go/common"
	carmen "github.com/Fantom-foundation/Carmen/go/state"
	substate "github.com/Fantom-foundation/Substate"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"

	estate "github.com/ledgerwatch/erigon/core/state"
	erigonethdb "github.com/ledgerwatch/erigon/ethdb"

	"github.com/ledgerwatch/erigon-lib/kv"
)

func MakeCarmenStateDB(directory, variant, archive string, schema int) (StateDB, error) {
	if variant == "" {
		variant = "go-memory"
	}

	var archiveType carmen.ArchiveType
	switch strings.ToLower(archive) {
	case "none":
		archiveType = carmen.NoArchive
	case "": // = default option
		fallthrough
	case "ldb":
		fallthrough
	case "leveldb":
		archiveType = carmen.LevelDbArchive
	case "sql":
		fallthrough
	case "sqlite":
		archiveType = carmen.SqliteArchive
	default:
		return nil, fmt.Errorf("unsupported archive type: %s", archive)
	}

	params := carmen.Parameters{
		Schema:    carmen.StateSchema(schema),
		Directory: directory,
		Archive:   archiveType,
	}

	var db carmen.State
	var err error
	switch variant {
	case "go-memory":
		db, err = carmen.NewGoMemoryState(params)
	case "go-file-nocache":
		db, err = carmen.NewGoFileState(params)
	case "go-file":
		db, err = carmen.NewGoCachedFileState(params)
	case "go-ldb-nocache":
		db, err = carmen.NewGoLeveLIndexAndStoreState(params)
	case "go-ldb":
		db, err = carmen.NewGoCachedLeveLIndexAndStoreState(params)
	case "cpp-memory":
		db, err = carmen.NewCppInMemoryState(params)
	case "cpp-file":
		db, err = carmen.NewCppFileBasedState(params)
	case "cpp-ldb":
		db, err = carmen.NewCppLevelDbBasedState(params)
	default:
		return nil, fmt.Errorf("unkown variant: %v", variant)
	}
	if err != nil {
		return nil, err
	}
	return &carmenStateDB{carmen.CreateStateDBUsing(db), 0, 0}, nil
}

type carmenStateDB struct {
	db               carmen.StateDB
	syncPeriodNumber uint64
	blockNumber      uint64
}

func (s *carmenStateDB) DB() erigonethdb.Database { return nil }

func (s *carmenStateDB) CommitBlock(stateWriter estate.StateWriter) error { return nil }

func (s *carmenStateDB) CommitBlockWithStateWriter() error { return nil }

func (s *carmenStateDB) NewBatch(kv.RwTx, chan struct{}) erigonethdb.DbWithPendingMutations {
	return nil
}

func (s *carmenStateDB) BeginBlockApply() error { return nil }

func (s *carmenStateDB) BeginBlockApplyBatch(batch erigonethdb.DbWithPendingMutations, noHistory bool, rwTx kv.RwTx) error {
	return nil
}

func (s *carmenStateDB) CreateAccount(addr common.Address) {
	s.db.CreateAccount(cc.Address(addr))
}

func (s *carmenStateDB) Exist(addr common.Address) bool {
	return s.db.Exist(cc.Address(addr))
}

func (s *carmenStateDB) Empty(addr common.Address) bool {
	return s.db.Empty(cc.Address(addr))
}

func (s *carmenStateDB) Suicide(addr common.Address) bool {
	return s.db.Suicide(cc.Address(addr))
}

func (s *carmenStateDB) HasSuicided(addr common.Address) bool {
	return s.db.HasSuicided(cc.Address(addr))
}

func (s *carmenStateDB) GetBalance(addr common.Address) *big.Int {
	return s.db.GetBalance(cc.Address(addr))
}

func (s *carmenStateDB) AddBalance(addr common.Address, value *big.Int) {
	s.db.AddBalance(cc.Address(addr), value)
}

func (s *carmenStateDB) SubBalance(addr common.Address, value *big.Int) {
	s.db.SubBalance(cc.Address(addr), value)
}

func (s *carmenStateDB) GetNonce(addr common.Address) uint64 {
	return s.db.GetNonce(cc.Address(addr))
}

func (s *carmenStateDB) SetNonce(addr common.Address, value uint64) {
	s.db.SetNonce(cc.Address(addr), value)
}

func (s *carmenStateDB) GetCommittedState(addr common.Address, key common.Hash) common.Hash {
	return common.Hash(s.db.GetCommittedState(cc.Address(addr), cc.Key(key)))
}

func (s *carmenStateDB) GetState(addr common.Address, key common.Hash) common.Hash {
	return common.Hash(s.db.GetState(cc.Address(addr), cc.Key(key)))
}

func (s *carmenStateDB) SetState(addr common.Address, key common.Hash, value common.Hash) {
	s.db.SetState(cc.Address(addr), cc.Key(key), cc.Value(value))
}

func (s *carmenStateDB) GetCode(addr common.Address) []byte {
	return s.db.GetCode(cc.Address(addr))
}

func (s *carmenStateDB) GetCodeSize(addr common.Address) int {
	return s.db.GetCodeSize(cc.Address(addr))
}

func (s *carmenStateDB) GetCodeHash(addr common.Address) common.Hash {
	return common.Hash(s.db.GetCodeHash(cc.Address(addr)))
}

func (s *carmenStateDB) SetCode(addr common.Address, code []byte) {
	s.db.SetCode(cc.Address(addr), code)
}

func (s *carmenStateDB) Snapshot() int {
	return s.db.Snapshot()
}

func (s *carmenStateDB) RevertToSnapshot(id int) {
	s.db.RevertToSnapshot(id)
}

func (s *carmenStateDB) BeginTransaction(uint32) {
	s.db.BeginTransaction()
}

func (s *carmenStateDB) EndTransaction() {
	s.db.EndTransaction()
}

func (s *carmenStateDB) BeginBlock(block uint64) {
	s.db.BeginBlock()
	s.blockNumber = block
}

func (s *carmenStateDB) EndBlock() {
	s.db.EndBlock(s.blockNumber)
}

func (s *carmenStateDB) BeginSyncPeriod(number uint64) {
	s.db.BeginEpoch()
	s.syncPeriodNumber = number
}

func (s *carmenStateDB) EndSyncPeriod() {
	s.db.EndEpoch(s.syncPeriodNumber)
}

func (s *carmenStateDB) Close() error {
	return s.db.Close()
}

func (s *carmenStateDB) AddRefund(amount uint64) {
	s.db.AddRefund(amount)
}

func (s *carmenStateDB) SubRefund(amount uint64) {
	s.db.SubRefund(amount)
}

func (s *carmenStateDB) GetRefund() uint64 {
	return s.db.GetRefund()
}

func (s *carmenStateDB) PrepareAccessList(sender common.Address, dest *common.Address, precompiles []common.Address, txAccesses types.AccessList) {
	s.db.ClearAccessList()
	s.db.AddAddressToAccessList(cc.Address(sender))
	if dest != nil {
		s.db.AddAddressToAccessList(cc.Address(*dest))
	}
	for _, addr := range precompiles {
		s.db.AddAddressToAccessList(cc.Address(addr))
	}
	for _, el := range txAccesses {
		s.db.AddAddressToAccessList(cc.Address(el.Address))
		for _, key := range el.StorageKeys {
			s.db.AddSlotToAccessList(cc.Address(el.Address), cc.Key(key))
		}
	}
}

func (s *carmenStateDB) AddressInAccessList(addr common.Address) bool {
	return s.db.IsAddressInAccessList(cc.Address(addr))
}

func (s *carmenStateDB) SlotInAccessList(addr common.Address, slot common.Hash) (addressOk bool, slotOk bool) {
	return s.db.IsSlotInAccessList(cc.Address(addr), cc.Key(slot))
}

func (s *carmenStateDB) AddAddressToAccessList(addr common.Address) {
	s.db.AddAddressToAccessList(cc.Address(addr))
}

func (s *carmenStateDB) AddSlotToAccessList(addr common.Address, slot common.Hash) {
	s.db.AddSlotToAccessList(cc.Address(addr), cc.Key(slot))
}

func (s *carmenStateDB) AddLog(*types.Log) {
	// ignored
}

func (s *carmenStateDB) GetLogs(hash common.Hash, blockHash common.Hash) []*types.Log {
	// ignored
	return nil
}

func (s *carmenStateDB) Finalise(deleteEmptyObjects bool) {
	// ignored
}

func (s *carmenStateDB) IntermediateRoot(deleteEmptyObjects bool) common.Hash {
	// ignored
	return common.Hash{}
}

func (s *carmenStateDB) Commit(deleteEmptyObjects bool) (common.Hash, error) {
	// ignored
	return common.Hash{}, nil
}

func (s *carmenStateDB) Prepare(thash common.Hash, ti int) {
	//ignored
}

func (s *carmenStateDB) PrepareSubstate(substate *substate.SubstateAlloc, block uint64) {
	// ignored
}

func (s *carmenStateDB) GetSubstatePostAlloc() substate.SubstateAlloc {
	// ignored
	return substate.SubstateAlloc{}
}

func (s *carmenStateDB) AddPreimage(common.Hash, []byte) {
	// ignored
	panic("AddPreimage not implemented")
}

func (s *carmenStateDB) ForEachStorage(common.Address, func(common.Hash, common.Hash) bool) error {
	// ignored
	panic("ForEachStorage not implemented")
}

func (s *carmenStateDB) Error() error {
	// ignored
	return nil
}

func (s *carmenStateDB) StartBulkLoad() BulkLoad {
	return &carmenBulkLoad{s.db.StartBulkLoad()}
}

func (s *carmenStateDB) GetArchiveState(block uint64) (StateDB, error) {
	state, err := s.db.GetArchiveStateDB(block)
	if err != nil {
		return nil, err
	}
	return &carmenStateDB{state, 0, 0}, nil
}

func (s *carmenStateDB) GetMemoryUsage() *MemoryUsage {
	usage := s.db.GetMemoryFootprint()
	return &MemoryUsage{uint64(usage.Total()), usage}
}

type carmenBulkLoad struct {
	load carmen.BulkLoad
}

func (l *carmenBulkLoad) CreateAccount(addr common.Address) {
	l.load.CreateAccount(cc.Address(addr))
}

func (l *carmenBulkLoad) SetBalance(addr common.Address, value *big.Int) {
	l.load.SetBalance(cc.Address(addr), value)
}

func (l *carmenBulkLoad) SetNonce(addr common.Address, nonce uint64) {
	l.load.SetNonce(cc.Address(addr), nonce)
}

func (l *carmenBulkLoad) SetState(addr common.Address, key common.Hash, value common.Hash) {
	l.load.SetState(cc.Address(addr), cc.Key(key), cc.Value(value))
}

func (l *carmenBulkLoad) SetCode(addr common.Address, code []byte) {
	l.load.SetCode(cc.Address(addr), code)
}

func (l *carmenBulkLoad) Close() error {
	return l.load.Close()
}
