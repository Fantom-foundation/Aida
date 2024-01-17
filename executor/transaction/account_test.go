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

func TestAccount_EqualNonce(t *testing.T) {
	newAccount := substate_transaction.NewOldSubstateAccount(substate.NewSubstateAccount(2, new(big.Int).SetUint64(1), []byte{1}))
	oldAccount := substate_transaction.NewOldSubstateAccount(substate.NewSubstateAccount(1, new(big.Int).SetUint64(1), []byte{1}))

	if newAccount.Equal(oldAccount) {
		t.Fatal("accounts nonce are different but equal returned true")
	}

	newAccount.SetNonce(oldAccount.GetNonce())
	if !newAccount.Equal(oldAccount) {
		t.Fatal("accounts nonce are same but equal returned false")
	}
}

func TestAccount_EqualBalance(t *testing.T) {
	newAccount := substate_transaction.NewOldSubstateAccount(substate.NewSubstateAccount(1, new(big.Int).SetUint64(2), []byte{1}))
	oldAccount := substate_transaction.NewOldSubstateAccount(substate.NewSubstateAccount(1, new(big.Int).SetUint64(1), []byte{1}))

	if newAccount.Equal(oldAccount) {
		t.Fatal("accounts balances are different but equal returned true")
	}

	newAccount.SetBalance(oldAccount.GetBalance())
	if !newAccount.Equal(oldAccount) {
		t.Fatal("accounts balances are same but equal returned false")
	}
}

func TestAccount_EqualStorage(t *testing.T) {
	hashOne := common.BigToHash(new(big.Int).SetUint64(1))
	hashTwo := common.BigToHash(new(big.Int).SetUint64(2))
	hashThree := common.BigToHash(new(big.Int).SetUint64(3))

	newAccount := substate_transaction.NewOldSubstateAccount(substate.NewSubstateAccount(1, new(big.Int).SetUint64(1), []byte{1}))
	newAccount.SetStorageAt(hashOne, hashTwo)

	// first compare with no storage
	oldAccount := substate_transaction.NewOldSubstateAccount(substate.NewSubstateAccount(1, new(big.Int).SetUint64(1), []byte{1}))
	if newAccount.Equal(oldAccount) {
		t.Fatal("accounts storages are different but equal returned true")
	}

	// then compare different value for same key
	oldAccount.SetStorageAt(hashOne, hashThree)
	if !newAccount.Equal(oldAccount) {
		t.Fatal("accounts storages are same but equal returned false")
	}

	// then compare same
	oldAccount.SetStorageAt(hashOne, hashTwo)
	if !newAccount.Equal(oldAccount) {
		t.Fatal("accounts storages are different but equal returned true")
	}

	// then compare different keys
	oldAccount.SetStorageAt(hashTwo, hashThree)
	if newAccount.Equal(oldAccount) {
		t.Fatal("accounts storages are different but equal returned true")
	}

}

func TestAccount_EqualCode(t *testing.T) {
	newAccount := substate_transaction.NewOldSubstateAccount(substate.NewSubstateAccount(1, new(big.Int).SetUint64(1), []byte{2}))
	oldAccount := substate_transaction.NewOldSubstateAccount(substate.NewSubstateAccount(1, new(big.Int).SetUint64(1), []byte{1}))

	if newAccount.Equal(oldAccount) {
		t.Fatal("accounts codes are different but equal returned true")
	}

	newAccount.SetCode(oldAccount.GetCode())
	if !newAccount.Equal(oldAccount) {
		t.Fatal("accounts codes are same but equal returned false")
	}

}

func TestAccount_String(t *testing.T) {
	hashOne := common.BigToHash(new(big.Int).SetUint64(1))
	hashTwo := common.BigToHash(new(big.Int).SetUint64(2))

	acc := substate_transaction.NewOldSubstateAccount(substate.NewSubstateAccount(1, new(big.Int).SetUint64(1), []byte{1}))
	acc.SetStorageAt(hashOne, hashTwo)

	got := AccountString(acc)
	want := fmt.Sprintf("Account{\n\t\t\tnonce: %d\n\t\t\tbalance %v\n\t\t\tStorage{\n\t\t\t\t%v=%v\n\t\t\t}\n\t\t}", 1, 1, hashOne, hashTwo)
	if strings.Compare(got, want) != 0 {
		t.Fatalf("strings are different\ngot: %v\nwant: %v", got, want)
	}
}
