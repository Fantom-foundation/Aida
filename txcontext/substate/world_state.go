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
