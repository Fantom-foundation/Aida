package state

/*
import (
	"fmt"
	"math/big"

	cc "github.com/Fantom-foundation/Carmen/go/common"
	carmen "github.com/Fantom-foundation/Carmen/go/state"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/substate"
)

func MakeCarmenStateDB(directory, variant string) (StateDB, error) {
	if variant == "" {
		variant = "go-memory"
	}

	var db carmen.State
	var err error
	switch variant {
	case "go-memory":
		db, err = carmen.NewGoMemoryState()
	case "go-file-nocache":
		db, err = carmen.NewGoFileState(directory)
	case "go-file":
		db, err = carmen.NewGoCachedFileState(directory)
	case "go-ldb-nocache":
		db, err = carmen.NewGoLeveLIndexAndStoreState(directory)
	case "go-ldb":
		db, err = carmen.NewGoCachedLeveLIndexAndStoreState(directory)
	case "cpp-memory":
		db, err = carmen.NewCppInMemoryState(directory)
	case "cpp-file":
		db, err = carmen.NewCppFileBasedState(directory)
	case "cpp-ldb":
		db, err = carmen.NewCppLevelDbBasedState(directory)
	default:
		return nil, fmt.Errorf("unkown variant: %v", variant)
	}
	if err != nil {
		return nil, err
	}
	return &carmenStateDB{carmen.CreateStateDBUsing(db), 0, 0}, nil
}

type carmenStateDB struct {
	db          carmen.StateDB
	epochNumber uint64
	blockNumber uint64
}

var getCodeCalled bool
var getCodeSizeCalled bool
var getCodeHashCalled bool
var setCodeCalled bool

func (s *carmenStateDB) BeginBlockApply() error {
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

func (s *carmenStateDB) BeginEpoch(number uint64) {
	s.db.BeginEpoch()
	s.epochNumber = number
}

func (s *carmenStateDB) EndEpoch() {
	s.db.EndEpoch(s.epochNumber)
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

func (s *carmenStateDB) PrepareSubstate(substate *substate.SubstateAlloc) {
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
	return nil
}

func (s *carmenStateDB) StartBulkLoad() BulkLoad {
	return &carmenBulkLoad{s.db.StartBulkLoad()}
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
*/
