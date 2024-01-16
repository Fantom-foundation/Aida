package transaction

import (
	"math/big"

	oldSubstate "github.com/Fantom-foundation/Substate"
	"github.com/ethereum/go-ethereum/common"
)

// Deprecated: This is a workaround before oldSubstate repository is migrated to new structure.
// Use NewSubstateAlloc instead.
func NewOldSubstateAlloc(alloc oldSubstate.SubstateAlloc) WorldState {
	return oldSubstateAlloc{alloc: alloc}
}

// Deprecated: This is a workaround before oldSubstate repository is migrated to new structure.
// Use substateAlloc instead.
type oldSubstateAlloc struct {
	alloc oldSubstate.SubstateAlloc
}

func (a oldSubstateAlloc) Has(addr common.Address) bool {
	_, ok := a.alloc[addr]
	return ok
}

func (a oldSubstateAlloc) Equal(y WorldState) bool {
	return allocEqual(a, y)
}

func (a oldSubstateAlloc) Get(addr common.Address) Account {
	acc, ok := a.alloc[addr]
	if !ok {
		return nil
	}

	return NewOldSubstateAccount(acc)

}

func (a oldSubstateAlloc) Add(addr common.Address, acc Account) {
	a.alloc[addr] = oldSubstate.NewSubstateAccount(acc.GetNonce(), new(big.Int).Set(acc.GetBalance()), acc.GetCode())
}

func (a oldSubstateAlloc) ForEach(h accountHandler) {
	for addr, acc := range a.alloc {
		h(addr, NewOldSubstateAccount(acc))
	}
}

func (a oldSubstateAlloc) Len() int {
	return len(a.alloc)
}

func (a oldSubstateAlloc) Delete(addr common.Address) {
	delete(a.alloc, addr)
}

func (a oldSubstateAlloc) String() string {
	return allocString(a)
}
