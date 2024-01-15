package transaction

import (
	"math/big"

	substateCommon "github.com/Fantom-foundation/Substate/geth/common"
	"github.com/Fantom-foundation/Substate/substate"
	"github.com/ethereum/go-ethereum/common"
)

func NewSubstateAccount(acc *substate.Account) Account {
	return &substateAccount{acc}
}

type substateAccount struct {
	*substate.Account
}

func (a *substateAccount) GetNonce() uint64 {
	return a.Nonce
}

func (a *substateAccount) GetBalance() *big.Int {
	return a.Balance
}

func (a *substateAccount) HasStorageAt(key common.Hash) bool {
	_, ok := a.Storage[substateCommon.Hash(key)]
	return ok
}

func (a *substateAccount) GetStorageAt(hash common.Hash) common.Hash {
	return common.Hash(a.Storage[substateCommon.Hash(hash)])
}

func (a *substateAccount) GetCode() []byte {
	return a.Code
}

func (a *substateAccount) SetNonce(nonce uint64) {
	a.Nonce = nonce
}

func (a *substateAccount) SetBalance(balance *big.Int) {
	a.Balance.Set(balance)
}

func (a *substateAccount) SetStorageAt(h1, h2 common.Hash) {
	a.Storage[substateCommon.Hash(h1)] = substateCommon.Hash(h2)
}

func (a *substateAccount) SetCode(code []byte) {
	a.Code = code
}
func (a *substateAccount) GetStorageSize() int {
	return len(a.Storage)
}

func (a *substateAccount) ForEachStorage(h storageHandler) {
	for keyHash, valueHash := range a.Storage {
		h(common.Hash(keyHash), common.Hash(valueHash))
	}
}

func (a *substateAccount) Equal(y Account) bool {
	return accountEqual(a, y)
}
