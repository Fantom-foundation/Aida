package types

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"math/big"
	"sort"
)

// Account is modification of SubstateAccount in substate/substate.go
type Account struct {
	Hash    common.Hash
	Storage map[common.Hash]common.Hash
	Code    []byte
	state.Account
}

// StoredAccount contains data from Account in RLP supported formats
type StoredAccount struct {
	Nonce       uint64
	Balance     *big.Int
	StorageRoot common.Hash
	CodeHash    []byte
	Storage     [][2]common.Hash
}

// ToStoredAccount converts Account into StoredAccount
func (a *Account) ToStoredAccount() *StoredAccount {
	var sa StoredAccount

	sa.Nonce = a.Nonce
	sa.Balance = new(big.Int).Set(a.Balance)
	sa.CodeHash = a.CodeHash
	sa.StorageRoot = a.Root

	// sorting storage by keys
	sortedKeys := make([]common.Hash, 0, len(a.Storage))
	for key := range a.Storage {
		sortedKeys = append(sortedKeys, key)
	}
	sort.Slice(sortedKeys, func(i, j int) bool {
		return sortedKeys[i].Big().Cmp(sortedKeys[j].Big()) < 0
	})

	// inserting sorted database into storage
	sa.Storage = make([][2]common.Hash, 0, len(a.Storage))
	for _, key := range sortedKeys {
		value := a.Storage[key]
		sa.Storage = append(sa.Storage, [2]common.Hash{key, value})
	}

	return &sa
}

// ToAccount converts stored account to Account structure.
func (sa *StoredAccount) ToAccount() *Account {
	var ac Account

	ac.Nonce = sa.Nonce
	ac.Balance = sa.Balance
	ac.CodeHash = sa.CodeHash
	ac.Root = sa.StorageRoot

	// convert the storage representation to the hash map
	ac.Storage = make(map[common.Hash]common.Hash, len(sa.Storage))
	for _, si := range sa.Storage {
		ac.Storage[si[0]] = si[1]
	}

	return &ac
}
