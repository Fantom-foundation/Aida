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

func TestWorldState_Equal(t *testing.T) {
	ws := substate.SubstateAlloc{common.Address{1}: &substate.SubstateAccount{Nonce: 1, Balance: new(big.Int).SetUint64(1), Code: []byte{1}}}
	comparedWorldState := substate.SubstateAlloc{common.Address{1}: &substate.SubstateAccount{Nonce: 1, Balance: new(big.Int).SetUint64(1), Code: []byte{1}}}

	if !txcontext.WorldStateEqual(NewWorldState(ws), NewWorldState(comparedWorldState)) {
		t.Fatal("world states are same but equal returned false")
	}
}

func TestWorldState_NotEqual(t *testing.T) {
	ws := substate.SubstateAlloc{common.Address{1}: &substate.SubstateAccount{Nonce: 1, Balance: new(big.Int).SetUint64(1), Code: []byte{1}}}
	comparedWorldState := substate.SubstateAlloc{common.Address{2}: &substate.SubstateAccount{Nonce: 1, Balance: new(big.Int).SetUint64(1), Code: []byte{1}}}

	if !txcontext.WorldStateEqual(NewWorldState(ws), NewWorldState(comparedWorldState)) {
		t.Fatal("world states are different but equal returned false")
	}
}

func TestWorldState_Equal_DifferentLen(t *testing.T) {
	ws := substate.SubstateAlloc{common.Address{1}: &substate.SubstateAccount{Nonce: 1, Balance: new(big.Int).SetUint64(1), Code: []byte{1}}}
	comparedWorldState := substate.SubstateAlloc{common.Address{1}: &substate.SubstateAccount{Nonce: 1, Balance: new(big.Int).SetUint64(1), Code: []byte{1}}}

	// add one more acc to alloc
	comparedWorldState[common.Address{2}] = &substate.SubstateAccount{Nonce: 1, Balance: new(big.Int).SetUint64(1), Code: []byte{1}}

	if txcontext.WorldStateEqual(NewWorldState(ws), NewWorldState(comparedWorldState)) {
		t.Fatal("world states are different but equal returned true")
	}
}

func TestWorldState_String(t *testing.T) {
	ws := substate.SubstateAlloc{common.Address{1}: &substate.SubstateAccount{Nonce: 1, Balance: new(big.Int).SetUint64(1), Code: []byte{1}}}

	w := NewWorldState(ws)
	got := txcontext.WorldStateString(w)
	want := fmt.Sprintf("World State {\n\tsize: %d\n\tAccounts:\n\t\t%x: %v\n}", 1, common.Address{1}, w.Get(common.Address{1}).String())
	if strings.Compare(got, want) != 0 {
		t.Fatalf("strings are different \ngot: %v\nwant: %v", got, want)
	}
}
