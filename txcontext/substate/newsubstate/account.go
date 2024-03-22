package newsubstate

import (
	"math/big"

	"github.com/Fantom-foundation/Aida/txcontext"
	"github.com/Fantom-foundation/Substate/substate"
	"github.com/Fantom-foundation/Substate/types"
	"github.com/ethereum/go-ethereum/common"
)

func NewAccount(acc *substate.Account) txcontext.Account {
	return &account{acc}
}

type account struct {
	*substate.Account
}

func (a *account) GetNonce() uint64 {
	return a.Nonce
}

func (a *account) GetBalance() *big.Int {
	return a.Balance
}

func (a *account) HasStorageAt(key common.Hash) bool {
	_, ok := a.Storage[types.Hash(key)]
	return ok
}

func (a *account) GetStorageAt(hash common.Hash) common.Hash {
	return common.Hash(a.Storage[types.Hash(hash)])
}

func (a *account) GetCode() []byte {
	return a.Code
}

func (a *account) GetStorageSize() int {
	return len(a.Storage)
}

func (a *account) ForEachStorage(h txcontext.StorageHandler) {
	for keyHash, valueHash := range a.Storage {
		h(common.Hash(keyHash), common.Hash(valueHash))
	}
}

func (a *account) String() string {
	return txcontext.AccountString(a)
}
