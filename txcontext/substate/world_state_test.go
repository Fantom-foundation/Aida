// Copyright 2024 Fantom Foundation
// This file is part of Aida Testing Infrastructure for Sonic
//
// Aida is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Aida is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with Aida. If not, see <http://www.gnu.org/licenses/>.

package substate

import (
	"fmt"
	"math/big"
	"strings"
	"testing"

	"github.com/Fantom-foundation/Aida/txcontext"
	substate "github.com/Fantom-foundation/Substate"
	"github.com/ethereum/go-ethereum/common"
)

// TestWorldState_Equal tests whether Equal if Worlds States are same.
func TestWorldState_Equal(t *testing.T) {
	ws := substate.SubstateAlloc{common.Address{1}: &substate.SubstateAccount{Nonce: 1, Balance: new(big.Int).SetUint64(1), Code: []byte{1}}}
	comparedWorldState := substate.SubstateAlloc{common.Address{1}: &substate.SubstateAccount{Nonce: 1, Balance: new(big.Int).SetUint64(1), Code: []byte{1}}}

	if !txcontext.WorldStateEqual(NewWorldState(ws), NewWorldState(comparedWorldState)) {
		t.Fatal("world states are same but equal returned false")
	}
}

// TestWorldState_NotEqual tests whether Equal if Worlds States are not same.
func TestWorldState_NotEqual(t *testing.T) {
	ws := substate.SubstateAlloc{common.Address{1}: &substate.SubstateAccount{Nonce: 1, Balance: new(big.Int).SetUint64(1), Code: []byte{1}}}
	comparedWorldState := substate.SubstateAlloc{common.Address{2}: &substate.SubstateAccount{Nonce: 1, Balance: new(big.Int).SetUint64(1), Code: []byte{1}}}

	if !txcontext.WorldStateEqual(NewWorldState(ws), NewWorldState(comparedWorldState)) {
		t.Fatal("world states are different but equal returned false")
	}
}

// TestWorldState_Equal_DifferentLen tests whether Equal if Worlds States have different len.
func TestWorldState_Equal_DifferentLen(t *testing.T) {
	ws := substate.SubstateAlloc{common.Address{1}: &substate.SubstateAccount{Nonce: 1, Balance: new(big.Int).SetUint64(1), Code: []byte{1}}}
	comparedWorldState := substate.SubstateAlloc{common.Address{1}: &substate.SubstateAccount{Nonce: 1, Balance: new(big.Int).SetUint64(1), Code: []byte{1}}}

	// add one more acc to alloc
	comparedWorldState[common.Address{2}] = &substate.SubstateAccount{Nonce: 1, Balance: new(big.Int).SetUint64(1), Code: []byte{1}}

	if txcontext.WorldStateEqual(NewWorldState(ws), NewWorldState(comparedWorldState)) {
		t.Fatal("world states are different but equal returned true")
	}
}

// TestWorldState_String tests whether Stringify method has correct format.
func TestWorldState_String(t *testing.T) {
	ws := substate.SubstateAlloc{common.Address{1}: &substate.SubstateAccount{Nonce: 1, Balance: new(big.Int).SetUint64(1), Code: []byte{1}}}

	w := NewWorldState(ws)
	got := txcontext.WorldStateString(w)
	want := fmt.Sprintf("World State {\n\tsize: %d\n\tAccounts:\n\t\t%x: %v\n}", 1, common.Address{1}, w.Get(common.Address{1}).String())
	if strings.Compare(got, want) != 0 {
		t.Fatalf("strings are different \ngot: %v\nwant: %v", got, want)
	}
}
