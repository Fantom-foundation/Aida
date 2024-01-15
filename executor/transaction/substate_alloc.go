package transaction

import (
	"math/big"

	substateCommon "github.com/Fantom-foundation/Substate/geth/common"
	"github.com/Fantom-foundation/Substate/substate"
	"github.com/ethereum/go-ethereum/common"
)

func NewSubstateAlloc(alloc substate.Alloc) Alloc {
	return substateAlloc{alloc: alloc}
}

type substateAlloc struct {
	alloc substate.Alloc
}

func (a substateAlloc) Has(addr common.Address) bool {
	_, ok := a.alloc[substateCommon.Address(addr)]
	return ok
}

func (a substateAlloc) Equal(y Alloc) bool {
	return allocEqual(a, y)
}

func (a substateAlloc) Get(addr common.Address) Account {
	acc, ok := a.alloc[substateCommon.Address(addr)]
	if !ok {
		return nil
	}

	return NewSubstateAccount(acc)
}

func (a substateAlloc) Add(addr common.Address, acc Account) {
	a.alloc[substateCommon.Address(addr)] = substate.NewAccount(acc.GetNonce(), new(big.Int).Set(acc.GetBalance()), acc.GetCode())
}

func (a substateAlloc) ForEach(h accountHandler) {
	for addr, acc := range a.alloc {
		h(common.Address(addr), NewSubstateAccount(acc))
	}
}

func (a substateAlloc) Len() int {
	return len(a.alloc)
}

func (a substateAlloc) Delete(addr common.Address) {
	delete(a.alloc, substateCommon.Address(addr))
}

func (a substateAlloc) String() string {
	return allocString(a)
}
