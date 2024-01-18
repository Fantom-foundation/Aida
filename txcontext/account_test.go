package txcontext

import (
	"fmt"
	"math/big"
	"strings"
	"testing"

	substatecontext "github.com/Fantom-foundation/Aida/txcontext/substate"
	substate "github.com/Fantom-foundation/Substate"
	"github.com/ethereum/go-ethereum/common"
)

func TestAccount_EqualNonce(t *testing.T) {
	newAccount := substate.NewSubstateAccount(2, new(big.Int).SetUint64(1), []byte{1})
	oldAccount := substate.NewSubstateAccount(1, new(big.Int).SetUint64(1), []byte{1})

	if AccountEqual(substatecontext.NewAccount(newAccount), substatecontext.NewAccount(oldAccount)) {
		t.Fatal("accounts nonce are different but equal returned true")
	}

	newAccount.Nonce = oldAccount.Nonce
	if AccountEqual(substatecontext.NewAccount(newAccount), substatecontext.NewAccount(oldAccount)) {
		t.Fatal("accounts nonce are same but equal returned false")
	}
}

func TestAccount_EqualBalance(t *testing.T) {
	newAccount := substate.NewSubstateAccount(1, new(big.Int).SetUint64(2), []byte{1})
	oldAccount := substate.NewSubstateAccount(1, new(big.Int).SetUint64(1), []byte{1})
	if AccountEqual(substatecontext.NewAccount(newAccount), substatecontext.NewAccount(oldAccount)) {
		t.Fatal("accounts balances are different but equal returned true")
	}

	newAccount.Balance = oldAccount.Balance
	if AccountEqual(substatecontext.NewAccount(newAccount), substatecontext.NewAccount(oldAccount)) {
		t.Fatal("accounts balances are same but equal returned false")
	}
}

func TestAccount_EqualStorage(t *testing.T) {
	hashOne := common.BigToHash(new(big.Int).SetUint64(1))
	hashTwo := common.BigToHash(new(big.Int).SetUint64(2))
	hashThree := common.BigToHash(new(big.Int).SetUint64(3))

	newAccount := substate.NewSubstateAccount(1, new(big.Int).SetUint64(1), []byte{1})
	newAccount.Storage[hashOne] = hashTwo

	// first compare with no storage
	oldAccount := substate.NewSubstateAccount(1, new(big.Int).SetUint64(1), []byte{1})
	if AccountEqual(substatecontext.NewAccount(newAccount), substatecontext.NewAccount(oldAccount)) {
		t.Fatal("accounts storages are different but equal returned true")
	}

	// then compare different value for same key
	oldAccount.Storage[hashOne] = hashThree
	if AccountEqual(substatecontext.NewAccount(newAccount), substatecontext.NewAccount(oldAccount)) {
		t.Fatal("accounts storages are same but equal returned false")
	}

	// then compare same
	oldAccount.Storage[hashOne] = hashTwo
	if AccountEqual(substatecontext.NewAccount(newAccount), substatecontext.NewAccount(oldAccount)) {
		t.Fatal("accounts storages are different but equal returned true")
	}

	// then compare different keys
	oldAccount.Storage[hashTwo] = hashThree
	if AccountEqual(substatecontext.NewAccount(newAccount), substatecontext.NewAccount(oldAccount)) {
		t.Fatal("accounts storages are different but equal returned true")
	}

}

func TestAccount_EqualCode(t *testing.T) {
	newAccount := substate.NewSubstateAccount(1, new(big.Int).SetUint64(1), []byte{2})
	oldAccount := substate.NewSubstateAccount(1, new(big.Int).SetUint64(1), []byte{1})
	if AccountEqual(substatecontext.NewAccount(newAccount), substatecontext.NewAccount(oldAccount)) {
		t.Fatal("accounts codes are different but equal returned true")
	}

	newAccount.Code = oldAccount.Code
	if AccountEqual(substatecontext.NewAccount(newAccount), substatecontext.NewAccount(oldAccount)) {
		t.Fatal("accounts codes are same but equal returned false")
	}

}

func TestAccount_String(t *testing.T) {
	hashOne := common.BigToHash(new(big.Int).SetUint64(1))
	hashTwo := common.BigToHash(new(big.Int).SetUint64(2))

	acc := substate.NewSubstateAccount(1, new(big.Int).SetUint64(1), []byte{1})
	acc.Storage[hashOne] = hashTwo

	got := AccountString(substatecontext.NewAccount(acc))
	want := fmt.Sprintf("Account{\n\t\t\tnonce: %d\n\t\t\tbalance %v\n\t\t\tStorage{\n\t\t\t\t%v=%v\n\t\t\t}\n\t\t}", 1, 1, hashOne, hashTwo)
	if strings.Compare(got, want) != 0 {
		t.Fatalf("strings are different\ngot: %v\nwant: %v", got, want)
	}
}
