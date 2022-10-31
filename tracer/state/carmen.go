package state

import (
	"fmt"
	"math/big"

	cc "github.com/Fantom-foundation/Carmen/go/common"
	carmen "github.com/Fantom-foundation/Carmen/go/state"
	"github.com/ethereum/go-ethereum/common"
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
		db, err = carmen.NewGoInMemoryState()
	case "go-ldb":
		db, err = carmen.NewGoLevelDbState(directory)
	case "cpp-memory":
		db, err = carmen.NewCppInMemoryState()
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
var getCodeHashCalled bool

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

func (s *carmenStateDB) GetCodeHash(addr common.Address) common.Hash {
	if !getCodeHashCalled {
		fmt.Printf("WARNING: GetCodeHash not implemented\n")
		getCodeHashCalled = true
	}
	return common.Hash{}
}

func (s *carmenStateDB) GetCode(addr common.Address) []byte {
	if !getCodeCalled {
		fmt.Printf("WARNING: GetCode not implemented\n")
		getCodeCalled = true
	}
	return []byte{}
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
	// To be fair to the geth implementation, we comput the state hash after each transaction.
	s.db.GetHash()
}

func (s *carmenStateDB) PrepareSubstate(substate *substate.SubstateAlloc) {
	// ignored
}

func (s *carmenStateDB) GetSubstatePostAlloc() substate.SubstateAlloc {
	return substate.SubstateAlloc{}
}

func (s *carmenStateDB) Close() error {
	// TODO: implement
	return nil
}
