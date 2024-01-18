package substate

import (
	"github.com/Fantom-foundation/Aida/txcontext"
	oldSubstate "github.com/Fantom-foundation/Substate"
	"github.com/ethereum/go-ethereum/common"
)

// Deprecated: This is a workaround before oldSubstate repository is migrated to new structure.
// Use NewSubstateAlloc instead.
func NewWorldState(alloc oldSubstate.SubstateAlloc) txcontext.WorldState {
	return worldState{alloc: alloc}
}

// Deprecated: This is a workaround before oldSubstate repository is migrated to new structure.
// Use substateAlloc instead.
type worldState struct {
	alloc oldSubstate.SubstateAlloc
}

func (a worldState) Has(addr common.Address) bool {
	_, ok := a.alloc[addr]
	return ok
}

func (a worldState) Equal(y txcontext.WorldState) bool {
	return txcontext.WorldStateEqual(a, y)
}

func (a worldState) Get(addr common.Address) txcontext.Account {
	acc, ok := a.alloc[addr]
	if !ok {
		return nil
	}

	return NewAccount(acc)

}

func (a worldState) ForEachAccount(h txcontext.AccountHandler) {
	for addr, acc := range a.alloc {
		h(addr, NewAccount(acc))
	}
}

func (a worldState) Len() int {
	return len(a.alloc)
}

func (a worldState) Delete(addr common.Address) {
	delete(a.alloc, addr)
}

func (a worldState) String() string {
	return txcontext.WorldStateString(a)
}
