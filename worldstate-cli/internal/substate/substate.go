package substate

import (
	"github.com/ethereum/go-ethereum/common"
	"math/big"
	"sort"
)

// SubstateAccount is modification of GenesisAccount in core/genesis.go
type SubstateAccount struct {
	Nonce   uint64
	Balance *big.Int
	Storage map[common.Hash]common.Hash
	Code    []byte
}

type SubstateAccountRLP struct {
	Nonce   uint64
	Balance *big.Int
	Storage [][2]common.Hash
	Code    []byte
}

func (sa *SubstateAccount) NewSubstateAccountRLP() *SubstateAccountRLP {
	var saRLP SubstateAccountRLP

	saRLP.Nonce = sa.Nonce
	saRLP.Balance = new(big.Int).Set(sa.Balance)
	//TODO NOT SAME AS ORIGINAL SUBSTATE - put code in separate record?
	saRLP.Code = sa.Code
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
