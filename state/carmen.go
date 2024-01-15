package state

import (
	"fmt"
	"math/big"
	"strings"

	"github.com/Fantom-foundation/Aida/executor/transaction"
	cc "github.com/Fantom-foundation/Carmen/go/common"
	carmen "github.com/Fantom-foundation/Carmen/go/state"
	_ "github.com/Fantom-foundation/Carmen/go/state/cppstate"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

func MakeCarmenStateDB(directory, variant, archive string, schema int) (StateDB, error) {
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
	case "s4":
		archiveType = carmen.S4Archive
	case "s5":
		archiveType = carmen.S5Archive
	default:
		return nil, fmt.Errorf("unsupported archive type: %s", archive)
	}

	if variant == "" {
		variant = "go-file"
	}
	params := carmen.Parameters{
		Variant:   carmen.Variant(variant),
		Schema:    carmen.StateSchema(schema),
		Directory: directory,
		Archive:   archiveType,
	}

	state, err := carmen.NewState(params)
	if err != nil {
		return nil, err
	}
	db := carmen.CreateStateDBUsing(state)
	return &carmenStateDB{
		carmenVmStateDB: carmenVmStateDB{db},
		stateDb:         db,
	}, nil
}

type carmenVmStateDB struct {
	db carmen.VmStateDB
}

type carmenNonCommittableStateDB struct {
	carmenVmStateDB
	nonCommittableStateDB carmen.NonCommittableStateDB
}

type carmenStateDB struct {
	carmenVmStateDB
	stateDb          carmen.StateDB
	syncPeriodNumber uint64
	blockNumber      uint64
}

func (s *carmenVmStateDB) CreateAccount(addr common.Address) {
	s.db.CreateAccount(cc.Address(addr))
}

func (s *carmenVmStateDB) Exist(addr common.Address) bool {
	return s.db.Exist(cc.Address(addr))
}

func (s *carmenVmStateDB) Empty(addr common.Address) bool {
	return s.db.Empty(cc.Address(addr))
}

func (s *carmenVmStateDB) Suicide(addr common.Address) bool {
	return s.db.Suicide(cc.Address(addr))
}

func (s *carmenVmStateDB) HasSuicided(addr common.Address) bool {
	return s.db.HasSuicided(cc.Address(addr))
}

func (s *carmenVmStateDB) GetBalance(addr common.Address) *big.Int {
	return s.db.GetBalance(cc.Address(addr))
}

func (s *carmenVmStateDB) AddBalance(addr common.Address, value *big.Int) {
	s.db.AddBalance(cc.Address(addr), value)
}

func (s *carmenVmStateDB) SubBalance(addr common.Address, value *big.Int) {
	s.db.SubBalance(cc.Address(addr), value)
}

func (s *carmenVmStateDB) GetNonce(addr common.Address) uint64 {
	return s.db.GetNonce(cc.Address(addr))
}

func (s *carmenVmStateDB) SetNonce(addr common.Address, value uint64) {
	s.db.SetNonce(cc.Address(addr), value)
}

func (s *carmenVmStateDB) GetCommittedState(addr common.Address, key common.Hash) common.Hash {
	return common.Hash(s.db.GetCommittedState(cc.Address(addr), cc.Key(key)))
}

func (s *carmenVmStateDB) GetState(addr common.Address, key common.Hash) common.Hash {
	return common.Hash(s.db.GetState(cc.Address(addr), cc.Key(key)))
}

func (s *carmenVmStateDB) SetState(addr common.Address, key common.Hash, value common.Hash) {
	s.db.SetState(cc.Address(addr), cc.Key(key), cc.Value(value))
}

func (s *carmenVmStateDB) GetCode(addr common.Address) []byte {
	return s.db.GetCode(cc.Address(addr))
}

func (s *carmenVmStateDB) GetCodeSize(addr common.Address) int {
	return s.db.GetCodeSize(cc.Address(addr))
}

func (s *carmenVmStateDB) GetCodeHash(addr common.Address) common.Hash {
	return common.Hash(s.db.GetCodeHash(cc.Address(addr)))
}

func (s *carmenVmStateDB) SetCode(addr common.Address, code []byte) {
	s.db.SetCode(cc.Address(addr), code)
}

func (s *carmenVmStateDB) Snapshot() int {
	return s.db.Snapshot()
}

func (s *carmenVmStateDB) RevertToSnapshot(id int) {
	s.db.RevertToSnapshot(id)
}

func (s *carmenVmStateDB) BeginTransaction(uint32) {
	s.db.BeginTransaction()
}

func (s *carmenVmStateDB) EndTransaction() {
	s.db.EndTransaction()
}

func (s *carmenStateDB) BeginBlock(block uint64) {
	s.stateDb.BeginBlock()
	s.blockNumber = block
}

func (s *carmenStateDB) EndBlock() {
	s.stateDb.EndBlock(s.blockNumber)
}

func (s *carmenStateDB) BeginSyncPeriod(number uint64) {
	s.stateDb.BeginEpoch()
	s.syncPeriodNumber = number
}

func (s *carmenStateDB) EndSyncPeriod() {
	s.stateDb.EndEpoch(s.syncPeriodNumber)
}

func (s *carmenVmStateDB) GetHash() common.Hash {
	return common.Hash(s.db.GetHash())
}

func (s *carmenStateDB) Close() error {
	return s.stateDb.Close()
}

func (s *carmenVmStateDB) AddRefund(amount uint64) {
	s.db.AddRefund(amount)
}

func (s *carmenVmStateDB) SubRefund(amount uint64) {
	s.db.SubRefund(amount)
}

func (s *carmenVmStateDB) GetRefund() uint64 {
	return s.db.GetRefund()
}

func (s *carmenVmStateDB) PrepareAccessList(sender common.Address, dest *common.Address, precompiles []common.Address, txAccesses types.AccessList) {
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

func (s *carmenVmStateDB) AddressInAccessList(addr common.Address) bool {
	return s.db.IsAddressInAccessList(cc.Address(addr))
}

func (s *carmenVmStateDB) SlotInAccessList(addr common.Address, slot common.Hash) (addressOk bool, slotOk bool) {
	return s.db.IsSlotInAccessList(cc.Address(addr), cc.Key(slot))
}

func (s *carmenVmStateDB) AddAddressToAccessList(addr common.Address) {
	s.db.AddAddressToAccessList(cc.Address(addr))
}

func (s *carmenVmStateDB) AddSlotToAccessList(addr common.Address, slot common.Hash) {
	s.db.AddSlotToAccessList(cc.Address(addr), cc.Key(slot))
}

func (s *carmenVmStateDB) AddLog(log *types.Log) {
	topics := make([]cc.Hash, 0, len(log.Topics))
	for _, topic := range log.Topics {
		topics = append(topics, cc.Hash(topic))
	}
	s.db.AddLog(&cc.Log{
		Address: cc.Address(log.Address),
		Topics:  topics,
		Data:    log.Data,
	})
}

func (s *carmenVmStateDB) GetLogs(common.Hash, common.Hash) []*types.Log {
	list := s.db.GetLogs()

	res := make([]*types.Log, 0, len(list))
	for _, log := range list {
		topics := make([]common.Hash, 0, len(log.Topics))
		for _, topic := range log.Topics {
			topics = append(topics, common.Hash(topic))
		}
		res = append(res, &types.Log{
			Address: common.Address(log.Address),
			Topics:  topics,
			Data:    log.Data,
		})

	}
	return res
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

func (s *carmenVmStateDB) Prepare(thash common.Hash, ti int) {
	// ignored
}

func (s *carmenStateDB) PrepareSubstate(substate transaction.Alloc, block uint64) {
	// ignored
}

func (s *carmenVmStateDB) GetSubstatePostAlloc() transaction.Alloc {
	// ignored
	return nil
}

func (s *carmenVmStateDB) AddPreimage(common.Hash, []byte) {
	// ignored
	panic("AddPreimage not implemented")
}

func (s *carmenVmStateDB) ForEachStorage(common.Address, func(common.Hash, common.Hash) bool) error {
	// ignored
	panic("ForEachStorage not implemented")
}

func (s *carmenStateDB) Error() error {
	// ignored
	return nil
}

func (s *carmenStateDB) StartBulkLoad(block uint64) BulkLoad {
	return &carmenBulkLoad{s.stateDb.StartBulkLoad(block)}
}

func (s *carmenStateDB) GetArchiveState(block uint64) (NonCommittableStateDB, error) {
	archive, err := s.stateDb.GetArchiveStateDB(block)
	if err != nil {
		return nil, err
	}
	return &carmenNonCommittableStateDB{
		carmenVmStateDB:       carmenVmStateDB{archive},
		nonCommittableStateDB: archive,
	}, nil
}

func (s *carmenStateDB) GetArchiveBlockHeight() (uint64, bool, error) {
	return s.stateDb.GetArchiveBlockHeight()
}

func (s *carmenStateDB) GetMemoryUsage() *MemoryUsage {
	usage := s.stateDb.GetMemoryFootprint()
	if usage == nil {
		return &MemoryUsage{uint64(0), nil}
	}
	return &MemoryUsage{uint64(usage.Total()), usage}
}

func (s *carmenStateDB) GetShadowDB() StateDB {
	return nil
}

func (s *carmenNonCommittableStateDB) Release() {
	s.nonCommittableStateDB.Release()
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
