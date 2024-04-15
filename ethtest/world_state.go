// Copyright 2024 Fantom Foundation
// This file is part of Aida Testing Infrastructure for Sonic
//
// Aida is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Aida is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with Aida. If not, see <http://www.gnu.org/licenses/>.

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
