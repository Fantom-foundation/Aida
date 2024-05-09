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

// TestAccount_EqualNonce tests whether Equal works with nonce.
func TestAccount_EqualNonce(t *testing.T) {
	newAccount := substate.NewSubstateAccount(2, new(big.Int).SetUint64(1), []byte{1})
	oldAccount := substate.NewSubstateAccount(1, new(big.Int).SetUint64(1), []byte{1})

	if txcontext.AccountEqual(NewAccount(newAccount), NewAccount(oldAccount)) {
		t.Fatal("accounts nonce are different but equal returned true")
	}

	newAccount.Nonce = oldAccount.Nonce
	if !txcontext.AccountEqual(NewAccount(newAccount), NewAccount(oldAccount)) {
		t.Fatal("accounts nonce are same but equal returned false")
	}
}

// TestAccount_EqualBalance tests whether Equal works with balance.
func TestAccount_EqualBalance(t *testing.T) {
	newAccount := substate.NewSubstateAccount(1, new(big.Int).SetUint64(2), []byte{1})
	oldAccount := substate.NewSubstateAccount(1, new(big.Int).SetUint64(1), []byte{1})
	if txcontext.AccountEqual(NewAccount(newAccount), NewAccount(oldAccount)) {
		t.Fatal("accounts balances are different but equal returned true")
	}

	newAccount.Balance = oldAccount.Balance
	if !txcontext.AccountEqual(NewAccount(newAccount), NewAccount(oldAccount)) {
		t.Fatal("accounts balances are same but equal returned false")
	}
}

// TestAccount_EqualStorage tests whether Equal works with storage.
func TestAccount_EqualStorage(t *testing.T) {
	hashOne := common.BigToHash(new(big.Int).SetUint64(1))
	hashTwo := common.BigToHash(new(big.Int).SetUint64(2))
	hashThree := common.BigToHash(new(big.Int).SetUint64(3))

	newAccount := substate.NewSubstateAccount(1, new(big.Int).SetUint64(1), []byte{1})
	newAccount.Storage[hashOne] = hashTwo

	// first compare with no storage
	oldAccount := substate.NewSubstateAccount(1, new(big.Int).SetUint64(1), []byte{1})
	if txcontext.AccountEqual(NewAccount(newAccount), NewAccount(oldAccount)) {
		t.Fatal("accounts storages are different but equal returned true")
	}

	// then compare different value for same key
	oldAccount.Storage[hashOne] = hashThree
	if !txcontext.AccountEqual(NewAccount(newAccount), NewAccount(oldAccount)) {
		t.Fatal("accounts storages are same but equal returned false")
	}

	// then compare same
	oldAccount.Storage[hashOne] = hashTwo
	if !txcontext.AccountEqual(NewAccount(newAccount), NewAccount(oldAccount)) {
		t.Fatal("accounts storages are different but equal returned true")
	}

	// then compare different keys
	oldAccount.Storage[hashTwo] = hashThree
	if txcontext.AccountEqual(NewAccount(newAccount), NewAccount(oldAccount)) {
		t.Fatal("accounts storages are different but equal returned true")
	}

}

// TestAccount_EqualCode tests whether Equal works with code.
func TestAccount_EqualCode(t *testing.T) {
	newAccount := substate.NewSubstateAccount(1, new(big.Int).SetUint64(1), []byte{2})
	oldAccount := substate.NewSubstateAccount(1, new(big.Int).SetUint64(1), []byte{1})
	if txcontext.AccountEqual(NewAccount(newAccount), NewAccount(oldAccount)) {
		t.Fatal("accounts codes are different but equal returned true")
	}

	newAccount.Code = oldAccount.Code
	if !txcontext.AccountEqual(NewAccount(newAccount), NewAccount(oldAccount)) {
		t.Fatal("accounts codes are same but equal returned false")
	}

}

// TestAccount_String tests whether Stringify method works correctly.
func TestAccount_String(t *testing.T) {
	hashOne := common.BigToHash(new(big.Int).SetUint64(1))
	hashTwo := common.BigToHash(new(big.Int).SetUint64(2))

	acc := substate.NewSubstateAccount(1, new(big.Int).SetUint64(1), []byte{1})
	acc.Storage[hashOne] = hashTwo

	got := txcontext.AccountString(NewAccount(acc))
	want := fmt.Sprintf("Account{\n\t\t\tnonce: %d\n\t\t\tbalance %v\n\t\t\tStorage{\n\t\t\t\t%v=%v\n\t\t\t}\n\t\t}", 1, 1, hashOne, hashTwo)
	if strings.Compare(got, want) != 0 {
		t.Fatalf("strings are different\ngot: %v\nwant: %v", got, want)
	}
}
