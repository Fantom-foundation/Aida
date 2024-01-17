package transaction

import (
	"fmt"
	"math/big"
	"strings"
	"testing"

	"github.com/Fantom-foundation/Aida/executor/transaction/substate_transaction"
	substate "github.com/Fantom-foundation/Substate"
	"github.com/ethereum/go-ethereum/common"
)

func TestWorldState_Equal(t *testing.T) {
	worldState := substate.SubstateAlloc{common.Address{1}: &substate.SubstateAccount{Nonce: 1, Balance: new(big.Int).SetUint64(1), Code: []byte{1}}}
	comparedWorldState := substate.SubstateAlloc{common.Address{1}: &substate.SubstateAccount{Nonce: 1, Balance: new(big.Int).SetUint64(1), Code: []byte{1}}}

	if WorldStateEqual(substate_transaction.NewOldSubstateAlloc(worldState), substate_transaction.NewOldSubstateAlloc(comparedWorldState)) {
		t.Fatal("allocs are same but equal returned false")
	}
}

func TestWorldState_NotEqual(t *testing.T) {
	worldState := substate.SubstateAlloc{common.Address{1}: &substate.SubstateAccount{Nonce: 1, Balance: new(big.Int).SetUint64(1), Code: []byte{1}}}
	comparedWorldState := substate.SubstateAlloc{common.Address{2}: &substate.SubstateAccount{Nonce: 1, Balance: new(big.Int).SetUint64(1), Code: []byte{1}}}

	if WorldStateEqual(substate_transaction.NewOldSubstateAlloc(worldState), substate_transaction.NewOldSubstateAlloc(comparedWorldState)) {
		t.Fatal("allocs are different but equal returned false")
	}
}

func TestWorldState_Equal_DifferentLen(t *testing.T) {
	worldState := substate.SubstateAlloc{common.Address{1}: &substate.SubstateAccount{Nonce: 1, Balance: new(big.Int).SetUint64(1), Code: []byte{1}}}
	comparedWorldState := substate.SubstateAlloc{common.Address{1}: &substate.SubstateAccount{Nonce: 1, Balance: new(big.Int).SetUint64(1), Code: []byte{1}}}

	// add one more acc to alloc
	comparedWorldState[common.Address{2}] = &substate.SubstateAccount{Nonce: 1, Balance: new(big.Int).SetUint64(1), Code: []byte{1}}

	if WorldStateEqual(substate_transaction.NewOldSubstateAlloc(worldState), substate_transaction.NewOldSubstateAlloc(comparedWorldState)) {
		t.Fatal("allocs are different but equal returned false")
	}
}

func TestWorldState_String(t *testing.T) {
	worldState := substate.SubstateAlloc{common.Address{1}: &substate.SubstateAccount{Nonce: 1, Balance: new(big.Int).SetUint64(1), Code: []byte{1}}}

	got := WorldStateString(substate_transaction.NewOldSubstateAlloc(worldState))
	want := fmt.Sprintf("\tAccounts:\n\t\t%x: %v\nAccount{\n\t\t\tnonce: %d\n\t\t\tbalance %v\n\t\t\tStorage{\n\t\t\t\t%v=%v\n\t\t\t}\n\t\t}", common.Address{1}, 1, 1, 1, nil, nil)
	if strings.Compare(got, want) != 0 {
		t.Fatalf("strings are different \ngot: %v\nwant: %v", got, want)
	}
}
