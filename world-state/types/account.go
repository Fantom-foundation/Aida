package types

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/rlp"
	"io"
	"math/big"
)

// Account is modification of SubstateAccount in substate/substate.go
type Account struct {
	Hash    common.Hash
	Storage map[common.Hash]common.Hash
	Code    []byte
	state.Account
}

// accountRLP contains data from Account in RLP supported formats
type accountRLP struct {
	Nonce       uint64
	Balance     *big.Int
	CodeHash    []byte
	StorageSize uint64
}

// accStorageItmRLP represents a pair of storage key and value in RLP stream
type accStorageItmRLP struct {
	Key   common.Hash
	Value common.Hash
}

// EncodeRLP encodes given Account into a RLP stream.
func (a *Account) EncodeRLP(w io.Writer) error {
	// write the base
	err := rlp.Encode(w, &accountRLP{
		Nonce:       a.Nonce,
		Balance:     a.Balance,
		CodeHash:    a.CodeHash,
		StorageSize: uint64(len(a.Storage)),
	})
	if err != nil {
		return err
	}

	// write the storage map
	for k, v := range a.Storage {
		err = rlp.Encode(w, &accStorageItmRLP{
			Key:   k,
			Value: v,
		})
		if err != nil {
			return err
		}
	}
	return nil
}

// DecodeRLP decodes Account from RLP stream.
func (a *Account) DecodeRLP(s *rlp.Stream) error {
	// decode the base data
	ac := accountRLP{}
	err := s.Decode(&ac)
	if err != nil {
		return err
	}

	a.Nonce = ac.Nonce
	a.Balance = ac.Balance
	a.CodeHash = ac.CodeHash
	a.Storage = make(map[common.Hash]common.Hash, ac.StorageSize)

	// load the storage
	for i := 0; i < int(ac.StorageSize); i++ {
		itm := accStorageItmRLP{}
		err = s.Decode(&itm)
		if err != nil {
			return err
		}

		a.Storage[itm.Key] = itm.Value
	}
	return nil
}
