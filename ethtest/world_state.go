package ethtest

import (
	"github.com/Fantom-foundation/Aida/txcontext"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
)

func NewWorldState(alloc core.GenesisAlloc) txcontext.WorldState {
	return worldStateAlloc{alloc}
}

type worldStateAlloc struct {
	alloc core.GenesisAlloc
}

func (w worldStateAlloc) Get(addr common.Address) txcontext.Account {
	acc, ok := w.alloc[addr]
	if !ok {
		return txcontext.NewNilAccount()
	}
	return txcontext.NewAccount(acc.Code, acc.Storage, acc.Balance, acc.Nonce)
}

func (w worldStateAlloc) Has(addr common.Address) bool {
	_, ok := w.alloc[addr]
	return ok
}

func (w worldStateAlloc) ForEachAccount(h txcontext.AccountHandler) {
	for addr, acc := range w.alloc {
		h(addr, txcontext.NewAccount(acc.Code, acc.Storage, acc.Balance, acc.Nonce))
	}
}

func (w worldStateAlloc) Len() int {
	return len(w.alloc)
}

func (w worldStateAlloc) Equal(y txcontext.WorldState) bool {
	return txcontext.WorldStateEqual(w, y)
}

func (w worldStateAlloc) Delete(addr common.Address) {
	delete(w.alloc, addr)
}

func (w worldStateAlloc) String() string {
	return txcontext.WorldStateString(w)
}
