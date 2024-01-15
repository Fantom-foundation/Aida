package transaction

import (
	"math/big"

	oldSubstate "github.com/Fantom-foundation/Substate"
	"github.com/ethereum/go-ethereum/common"
)

func NewOldSubstateAccount(acc *oldSubstate.SubstateAccount) Account {
	return &oldSubstateAccount{acc}
}

type oldSubstateAccount struct {
	*oldSubstate.SubstateAccount
}

func (a *oldSubstateAccount) GetNonce() uint64 {
	return a.Nonce
}

func (a *oldSubstateAccount) GetBalance() *big.Int {
	return a.Balance
}

func (a *oldSubstateAccount) HasStorageAt(key common.Hash) bool {
	_, ok := a.Storage[key]
	return ok
}

func (a *oldSubstateAccount) GetStorageAt(hash common.Hash) common.Hash {
	return a.Storage[hash]
}

func (a *oldSubstateAccount) GetCode() []byte {
	return a.Code
}

func (a *oldSubstateAccount) SetNonce(nonce uint64) {
	a.Nonce = nonce
}

func (a *oldSubstateAccount) SetBalance(balance *big.Int) {
	a.Balance.Set(balance)
}

func (a *oldSubstateAccount) SetStorageAt(h1, h2 common.Hash) {
	a.Storage[h1] = h2
}

func (a *oldSubstateAccount) SetCode(code []byte) {
	a.Code = code
}
func (a *oldSubstateAccount) GetStorageSize() int {
	return len(a.Storage)
}

func (a *oldSubstateAccount) ForEachStorage(h storageHandler) {
	for keyHash, valueHash := range a.Storage {
		h(keyHash, valueHash)
	}
}

func (a *oldSubstateAccount) Equal(y Account) bool {
	return accountEqual(a, y)
}

func (a *oldSubstateAccount) String() string {
	return accountString(a)
}
