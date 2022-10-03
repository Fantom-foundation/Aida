package accstate

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"math/big"
	"sort"
)

// TODO: comments
// Account is modification of GenesisAccount in core/genesis.go
type Account struct {
	//hash of account addr
	Hash    common.Hash
	Storage map[common.Hash]common.Hash
	Code    []byte
	state.Account
}

type StoredAccount struct {
	Nonce    uint64
	Balance  *big.Int
	Storage  [][2]common.Hash
	CodeHash []byte
}

func (a *Account) StoredAccount() *StoredAccount {
	var sa StoredAccount

	sa.Nonce = a.Nonce
	sa.Balance = new(big.Int).Set(a.Balance)
	sa.CodeHash = a.CodeHash

	sortedKeys := make([]common.Hash, 0, len(a.Storage))
	for key := range a.Storage {
		sortedKeys = append(sortedKeys, key)
	}
	sort.Slice(sortedKeys, func(i, j int) bool {
		return sortedKeys[i].Big().Cmp(sortedKeys[j].Big()) < 0
	})

	sa.Storage = make([][2]common.Hash, 0, len(a.Storage))
	for _, key := range sortedKeys {
		value := a.Storage[key]
		sa.Storage = append(sa.Storage, [2]common.Hash{key, value})
	}

	return &sa
}
