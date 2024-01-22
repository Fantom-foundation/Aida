package substate

import (
	"math/big"

	"github.com/Fantom-foundation/Aida/txcontext"
	oldSubstate "github.com/Fantom-foundation/Substate"
	"github.com/ethereum/go-ethereum/common"
)

// Deprecated: This is a workaround before oldSubstate repository is migrated to new structure.
// Use NewSubstateAccount instead.
func NewAccount(acc *oldSubstate.SubstateAccount) txcontext.Account {
	return &account{acc}
}

// Deprecated: This is a workaround before oldSubstate repository is migrated to new structure.
// Use substateAccount instead.
type account struct {
	*oldSubstate.SubstateAccount
}

func (a *account) GetNonce() uint64 {
	return a.Nonce
}

func (a *account) GetBalance() *big.Int {
	return a.Balance
}

func (a *account) HasStorageAt(key common.Hash) bool {
	_, ok := a.Storage[key]
	return ok
}

func (a *account) GetStorageAt(hash common.Hash) common.Hash {
	return a.Storage[hash]
}

func (a *account) GetCode() []byte {
	return a.Code
}

func (a *account) GetStorageSize() int {
	return len(a.Storage)
}

func (a *account) ForEachStorage(h txcontext.StorageHandler) {
	for keyHash, valueHash := range a.Storage {
		h(keyHash, valueHash)
	}
}

func (a *account) String() string {
	return txcontext.AccountString(a)
}
