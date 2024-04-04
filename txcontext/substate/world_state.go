package substate

import (
	"github.com/Fantom-foundation/Aida/txcontext"
	"github.com/Fantom-foundation/Substate/substate"
	"github.com/Fantom-foundation/Substate/types"
	"github.com/ethereum/go-ethereum/common"
)

func NewWorldState(alloc substate.WorldState) txcontext.WorldState {
	return worldState{alloc: alloc}
}

type worldState struct {
	alloc substate.WorldState
}

func (a worldState) Has(addr common.Address) bool {
	_, ok := a.alloc[types.Address(addr)]
	return ok
}

func (a worldState) Equal(y txcontext.WorldState) bool {
	return txcontext.WorldStateEqual(a, y)
}

func (a worldState) Get(addr common.Address) txcontext.Account {
	acc, ok := a.alloc[types.Address(addr)]
	if !ok {
		return nil
	}

	return NewAccount(acc)
}

func (a worldState) ForEachAccount(h txcontext.AccountHandler) {
	for addr, acc := range a.alloc {
		h(common.Address(addr), NewAccount(acc))
	}
}

func (a worldState) Len() int {
	return len(a.alloc)
}

func (a worldState) Delete(addr common.Address) {
	delete(a.alloc, types.Address(addr))
}

func (a worldState) String() string {
	return txcontext.WorldStateString(a)
}
