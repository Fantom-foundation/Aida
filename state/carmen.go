package state

import (
	"fmt"
	"math/big"

	"github.com/Fantom-foundation/Aida/txcontext"
	"github.com/Fantom-foundation/Carmen/go/carmen"
	oldCommon "github.com/Fantom-foundation/Carmen/go/common"
	_ "github.com/Fantom-foundation/Carmen/go/state/cppstate"
	_ "github.com/Fantom-foundation/Carmen/go/state/gostate"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

func MakeCarmenStateDB(dir string, variant carmen.Variant, schema carmen.Schema, archive carmen.Archive) (StateDB, error) {
	return MakeCarmenStateDBWithCacheSize(dir, variant, schema, archive, 0, 0)
}

func MakeCarmenStateDBWithCacheSize(dir string, variant carmen.Variant, schema carmen.Schema, archive carmen.Archive, liveDbCacheSize, archiveCacheSize int) (StateDB, error) {
	cfg := carmen.Configuration{
		Variant: variant,
		Schema:  schema,
		Archive: archive,
	}

	properties := make(carmen.Properties)
	if liveDbCacheSize > 0 {
		properties.SetInteger(carmen.LiveDBCache, liveDbCacheSize)
	}

	if archiveCacheSize > 0 {
		properties.SetInteger(carmen.ArchiveCache, archiveCacheSize)
	}

	db, err := carmen.OpenDatabase(dir, cfg, properties)
	if err != nil {
		return nil, fmt.Errorf("cannot open carmen database; %w", err)
	}

	return &carmenHeadState{
		carmenStateDB: carmenStateDB{
			db: db,
		},
	}, nil
}

type carmenStateDB struct {
	db          carmen.Database
	txCtx       carmen.TransactionContext
	blockNumber uint64
}

type carmenHeadState struct {
	carmenStateDB
	blkCtx carmen.HeadBlockContext
}

type carmenHistoricState struct {
	carmenStateDB
	blkCtx    carmen.HistoricBlockContext
	blkNumber uint64
}

func (s *carmenStateDB) CreateAccount(addr common.Address) {
	s.txCtx.CreateAccount(carmen.Address(addr))
}

func (s *carmenStateDB) Exist(addr common.Address) bool {
	return s.txCtx.Exist(carmen.Address(addr))
}

func (s *carmenStateDB) Empty(addr common.Address) bool {
	return s.txCtx.Empty(carmen.Address(addr))
}

func (s *carmenStateDB) Suicide(addr common.Address) bool {
	return s.txCtx.SelfDestruct(carmen.Address(addr))
}

func (s *carmenStateDB) HasSuicided(addr common.Address) bool {
	return s.txCtx.HasSelfDestructed(carmen.Address(addr))
}

func (s *carmenStateDB) GetBalance(addr common.Address) *big.Int {
	return s.txCtx.GetBalance(carmen.Address(addr))
}

func (s *carmenStateDB) AddBalance(addr common.Address, value *big.Int) {
	s.txCtx.AddBalance(carmen.Address(addr), value)
}

func (s *carmenStateDB) SubBalance(addr common.Address, value *big.Int) {
	s.txCtx.SubBalance(carmen.Address(addr), value)
}

func (s *carmenStateDB) GetNonce(addr common.Address) uint64 {
	return s.txCtx.GetNonce(carmen.Address(addr))
}

func (s *carmenStateDB) SetNonce(addr common.Address, value uint64) {
	s.txCtx.SetNonce(carmen.Address(addr), value)
}

func (s *carmenStateDB) GetCommittedState(addr common.Address, key common.Hash) common.Hash {
	return common.Hash(s.txCtx.GetCommittedState(carmen.Address(addr), carmen.Key(key)))
}

func (s *carmenStateDB) GetState(addr common.Address, key common.Hash) common.Hash {
	return common.Hash(s.txCtx.GetState(carmen.Address(addr), carmen.Key(key)))
}

func (s *carmenStateDB) SetState(addr common.Address, key common.Hash, value common.Hash) {
	s.txCtx.SetState(carmen.Address(addr), carmen.Key(key), carmen.Value(value))
}

func (s *carmenStateDB) GetCode(addr common.Address) []byte {
	return s.txCtx.GetCode(carmen.Address(addr))
}

func (s *carmenStateDB) GetCodeSize(addr common.Address) int {
	return s.txCtx.GetCodeSize(carmen.Address(addr))
}

func (s *carmenStateDB) GetCodeHash(addr common.Address) common.Hash {
	return common.Hash(s.txCtx.GetCodeHash(carmen.Address(addr)))
}

func (s *carmenStateDB) SetCode(addr common.Address, code []byte) {
	s.txCtx.SetCode(carmen.Address(addr), code)
}

func (s *carmenStateDB) Snapshot() int {
	return s.txCtx.Snapshot()
}

func (s *carmenStateDB) RevertToSnapshot(id int) {
	s.txCtx.RevertToSnapshot(id)
}

func (s *carmenHeadState) BeginTransaction(tx uint32) error {
	var err error
	s.txCtx, err = s.blkCtx.BeginTransaction(int(tx))
	return err
}

func (s *carmenStateDB) EndTransaction() error {
	return s.txCtx.Commit()
}

func (s *carmenHeadState) BeginBlock(block uint64) error {
	var err error
	s.blkCtx, err = s.db.BeginBlock(block)
	return err
}

func (s *carmenHeadState) EndBlock() error {
	return s.blkCtx.Commit()
}

func (s *carmenHeadState) BeginSyncPeriod(number uint64) {
	// ignored for Carmen
}

func (s *carmenHeadState) EndSyncPeriod() {
	// ignored for Carmen
}

func (s *carmenStateDB) GetHash() (common.Hash, error) {
	hash, err := s.db.GetHeadStateHash()
	return common.Hash(hash), err
}

func (s *carmenStateDB) Close() error {
	return s.db.Close()
}

func (s *carmenStateDB) AddRefund(amount uint64) {
	s.txCtx.AddRefund(amount)
}

func (s *carmenStateDB) SubRefund(amount uint64) {
	s.txCtx.SubRefund(amount)
}

func (s *carmenStateDB) GetRefund() uint64 {
	return s.txCtx.GetRefund()
}

func (s *carmenStateDB) PrepareAccessList(sender common.Address, dest *common.Address, precompiles []common.Address, txAccesses types.AccessList) {
	s.txCtx.ClearAccessList()
	s.txCtx.AddAddressToAccessList(carmen.Address(sender))
	if dest != nil {
		s.txCtx.AddAddressToAccessList(carmen.Address(*dest))
	}
	for _, addr := range precompiles {
		s.txCtx.AddAddressToAccessList(carmen.Address(addr))
	}
	for _, el := range txAccesses {
		s.txCtx.AddAddressToAccessList(carmen.Address(el.Address))
		for _, key := range el.StorageKeys {
			s.txCtx.AddSlotToAccessList(carmen.Address(el.Address), carmen.Key(key))
		}
	}
}

func (s *carmenStateDB) AddressInAccessList(addr common.Address) bool {
	return s.txCtx.IsAddressInAccessList(carmen.Address(addr))
}

func (s *carmenStateDB) SlotInAccessList(addr common.Address, slot common.Hash) (addressOk bool, slotOk bool) {
	return s.txCtx.IsSlotInAccessList(carmen.Address(addr), carmen.Key(slot))
}

func (s *carmenStateDB) AddAddressToAccessList(addr common.Address) {
	s.txCtx.AddAddressToAccessList(carmen.Address(addr))
}

func (s *carmenStateDB) AddSlotToAccessList(addr common.Address, slot common.Hash) {
	s.txCtx.AddSlotToAccessList(carmen.Address(addr), carmen.Key(slot))
}

func (s *carmenStateDB) AddLog(log *types.Log) {
	topics := make([]oldCommon.Hash, 0, len(log.Topics))
	for _, topic := range log.Topics {
		topics = append(topics, oldCommon.Hash(topic))
	}
	s.txCtx.AddLog(&carmen.Log{
		Address: oldCommon.Address(log.Address),
		Topics:  topics,
		Data:    log.Data,
	})
}

func (s *carmenStateDB) GetLogs(common.Hash, common.Hash) []*types.Log {
	list := s.txCtx.GetLogs()

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

func (s *carmenStateDB) Finalise(bool) {
	// ignored
}

func (s *carmenStateDB) IntermediateRoot(bool) common.Hash {
	// ignored
	return common.Hash{}
}

func (s *carmenStateDB) Commit(bool) (common.Hash, error) {
	// ignored
	return common.Hash{}, nil
}

func (s *carmenStateDB) Prepare(common.Hash, int) {
	// ignored
}

func (s *carmenStateDB) PrepareSubstate(txcontext.WorldState, uint64) {
	// ignored
}

func (s *carmenStateDB) GetSubstatePostAlloc() txcontext.WorldState {
	// ignored
	return nil
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

func (s *carmenHistoricState) BeginTransaction(tx uint32) error {
	var err error
	s.txCtx, err = s.blkCtx.BeginTransaction(int(tx))
	return err
}

func (s *carmenHistoricState) GetHash() (common.Hash, error) {
	h, err := s.db.GetHistoricStateHash(s.blkNumber)
	return common.Hash(h), err
}

func (s *carmenStateDB) StartBulkLoad(block uint64) BulkLoad {
	bl, err := s.db.StartBulkLoad(block)
	if err != nil {
		return nil
	}
	return &carmenBulkLoad{bl}
}

func (s *carmenHeadState) GetArchiveState(block uint64) (NonCommittableStateDB, error) {
	historicBlkCtx, err := s.db.GetHistoricContext(block)
	if err != nil {
		return nil, err
	}

	return &carmenHistoricState{
		carmenStateDB: s.carmenStateDB,
		blkCtx:        historicBlkCtx,
	}, nil
}

func (s *carmenHeadState) GetArchiveBlockHeight() (uint64, bool, error) {
	blk, err := s.db.GetBlockHeight()
	if err != nil {
		return 0, false, err
	}
	if blk == -1 {
		return 0, false, nil
	}
	return uint64(blk), true, nil
}

func (s *carmenStateDB) GetMemoryUsage() *MemoryUsage {
	// todo waiting for implementation from carmen side
	//usage := s.db.GetMemoryFootprint()
	//if usage == nil {
	//	return &MemoryUsage{uint64(0), nil}
	//}
	return &MemoryUsage{uint64(0), nil}
}

func (s *carmenStateDB) GetShadowDB() StateDB {
	return nil
}

func (s *carmenHistoricState) Release() error {
	return s.blkCtx.Close()
}

// ----------------------------------------------------------------------------
//                                  BulkLoad
// ----------------------------------------------------------------------------

type carmenBulkLoad struct {
	load carmen.BulkLoad
}

func (l *carmenBulkLoad) CreateAccount(addr common.Address) {
	l.load.CreateAccount(carmen.Address(addr))
}

func (l *carmenBulkLoad) SetBalance(addr common.Address, value *big.Int) {
	l.load.SetBalance(carmen.Address(addr), value)
}

func (l *carmenBulkLoad) SetNonce(addr common.Address, nonce uint64) {
	l.load.SetNonce(carmen.Address(addr), nonce)
}

func (l *carmenBulkLoad) SetState(addr common.Address, key common.Hash, value common.Hash) {
	l.load.SetState(carmen.Address(addr), carmen.Key(key), carmen.Value(value))
}

func (l *carmenBulkLoad) SetCode(addr common.Address, code []byte) {
	l.load.SetCode(carmen.Address(addr), code)
}

func (l *carmenBulkLoad) Close() error {
	return l.load.Finalize()
}
