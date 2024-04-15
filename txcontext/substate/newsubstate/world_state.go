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

package newsubstate

import (
	"github.com/Fantom-foundation/Aida/txcontext"
	substateCommon "github.com/Fantom-foundation/Substate/geth/common"
	"github.com/Fantom-foundation/Substate/substate"
	"github.com/ethereum/go-ethereum/common"
)

func NewWorldState(alloc substate.Alloc) txcontext.WorldState {
	return worldState{alloc: alloc}
}

type worldState struct {
	alloc substate.Alloc
}

func (a worldState) Has(addr common.Address) bool {
	_, ok := a.alloc[substateCommon.Address(addr)]
	return ok
}

func (a worldState) Equal(y txcontext.WorldState) bool {
	return txcontext.WorldStateEqual(a, y)
}

func (a worldState) Get(addr common.Address) txcontext.Account {
	acc, ok := a.alloc[substateCommon.Address(addr)]
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
	delete(a.alloc, substateCommon.Address(addr))
}

func (a worldState) String() string {
	return txcontext.WorldStateString(a)
}
