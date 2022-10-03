package accstate

import (
	"github.com/ethereum/go-ethereum/common"
	"math/big"
	"sort"
)

// Account is modification of GenesisAccount in core/genesis.go
type Account struct {
	//hash of account addr
	Hash    common.Hash
	Nonce   uint64
	Balance *big.Int
	Storage map[common.Hash]common.Hash
	//root index in mpt tree
	Root common.Hash
	//hash of code
	CodeHash []byte
	Code     []byte
}

type AccountStorageRLP struct {
	Nonce    uint64
	Balance  *big.Int
	Storage  [][2]common.Hash
	CodeHash []byte
}

func (sa *Account) NewAccountStorageRLP() *AccountStorageRLP {
	var saRLP AccountStorageRLP

	saRLP.Nonce = sa.Nonce
	saRLP.Balance = new(big.Int).Set(sa.Balance)
	saRLP.CodeHash = sa.CodeHash
	sortedKeys := []common.Hash{}
	for key := range sa.Storage {
		sortedKeys = append(sortedKeys, key)
	}
	sort.Slice(sortedKeys, func(i, j int) bool {
		return sortedKeys[i].Big().Cmp(sortedKeys[j].Big()) < 0
	})
	for _, key := range sortedKeys {
		value := sa.Storage[key]
		saRLP.Storage = append(saRLP.Storage, [2]common.Hash{key, value})
	}

	return &saRLP
}
