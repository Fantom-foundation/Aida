package state

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
		db, err = carmen.NewMemory(directory)
	case "go-file":
		db, err = carmen.NewCachedLeveLIndexFileStore(directory)
	case "go-ldb":
		db, err = carmen.NewLeveLIndexAndStore(directory)
	case "cpp-memory":
		db, err = carmen.NewCppInMemoryState(directory)
	case "cpp-file":
		db, err = carmen.NewCppFileBasedState(directory)
	default:
		return nil, fmt.Errorf("unkown variant: %v", variant)
	}
	if err != nil {
		return nil, err
	}
	return &carmenStateDB{carmen.CreateStateDBUsing(db)}, nil
}

type carmenStateDB struct {
	db carmen.StateDB
}

var getCodeCalled bool
var getCodeSizeCalled bool
var getCodeHashCalled bool
var setCodeCalled bool

func (s *carmenStateDB) BeginBlockApply(root_hash common.Hash) error {
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
	s.db.Suicide(cc.Address(addr))
	return false // TODO: support an actual return value
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

func (s *carmenStateDB) Finalise(deleteEmptyObjects bool) {
	// In Geth 'Finalise' is called to end a transaction and seal its effects.
	// In Carmen, this event is called 'EndTransaction'.
	s.db.EndTransaction()
}

func (s *carmenStateDB) IntermediateRoot(deleteEmptyObjects bool) common.Hash {
	s.db.EndTransaction()
	return common.Hash(s.db.GetHash())
}

func (s *carmenStateDB) Commit(deleteEmptyObjects bool) (common.Hash, error) {
	return common.Hash(s.db.GetHash()), nil
}

func (s *carmenStateDB) Prepare(thash common.Hash, ti int) {
	//ignored
}

func (s *carmenStateDB) PrepareSubstate(substate *substate.SubstateAlloc) {
	// ignored
}

func (s *carmenStateDB) GetSubstatePostAlloc() substate.SubstateAlloc {
	return substate.SubstateAlloc{}
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

func (s *carmenStateDB) AddPreimage(common.Hash, []byte) {
	// ignored
	panic("AddPreimage not implemented")
}

func (s *carmenStateDB) ForEachStorage(common.Address, func(common.Hash, common.Hash) bool) error {
	// ignored
	panic("ForEachStorage not implemented")
	return nil
}
