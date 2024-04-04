package types

import (
	"bytes"
	"fmt"
	"io"
	"math/big"

	"github.com/Fantom-foundation/Substate/substate"
	"github.com/Fantom-foundation/lachesis-base/hash"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
)

// Account is modification of Account in substate/substate.go
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

var (
	// EmptyCode represents hash bytes of an empty code account.
	EmptyCode = crypto.Keccak256(nil)

	// EmptyCodeHash is used by create to ensure deployment is disallowed to already
	// deployed contract addresses (relevant after the account abstraction).
	EmptyCodeHash = common.BytesToHash(EmptyCode)

	// validateHasher is used for hashing storage slots when validation evolve
	validateHasher = crypto.NewKeccakState()
)

var (
	ErrAccountHash         = fmt.Errorf("different account hash")
	ErrAccountNonce        = fmt.Errorf("different account nonce")
	ErrAccountBalance      = fmt.Errorf("different account balance")
	ErrAccountStorage      = fmt.Errorf("uninitialized storage")
	ErrAccountCode         = fmt.Errorf("different account code")
	ErrAccountStorageSize  = fmt.Errorf("different storage size")
	ErrAccountStorageItem  = fmt.Errorf("missing storage item")
	ErrAccountStorageValue = fmt.Errorf("different storage item value")
)

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

// IsEmpty checks if the account is empty.
func (a *Account) IsEmpty() bool {
	return a.Nonce == 0 && a.Balance.Sign() == 0 && bytes.Equal(a.CodeHash, EmptyCode)
}

// IsIdentical compares the account to another instance
// and returns TRUE if and only if the accounts are identical.
func (a *Account) IsIdentical(b *Account) bool {
	err := a.IsDifferent(b)
	return err == nil
}

// IsDifferent compares the account to another instance
// and returns an error if and only if the accounts are different.
func (a *Account) IsDifferent(b *Account) error {
	// address hash must be the same (and the address with it)
	if a.Hash != b.Hash {
		return ErrAccountHash
	}

	// nonce must be the same
	if a.Nonce != b.Nonce {
		return ErrAccountNonce
	}

	// balance must be the same
	if a.Balance.Cmp(b.Balance) != 0 {
		return ErrAccountBalance
	}

	// storage must be either both nil, or both non-nil
	if (a.Storage == nil && b.Storage != nil) || (a.Storage != nil && b.Storage == nil) {
		return ErrAccountStorage
	}

	// if there is no storage, we are done
	if a.Storage == nil {
		return nil
	}

	// the storage size must be the same; if there is a storage
	if len(a.Storage) != len(b.Storage) {
		return ErrAccountStorageSize
	}

	// -----------------------
	// expensive checks below
	// -----------------------

	// code must be the same
	if bytes.Compare(a.Code, b.Code) != 0 {
		return ErrAccountCode
	}

	// compare storage content; we already know both have the same number of items
	for k, va := range a.Storage {
		vb, ok := b.Storage[k]
		if !ok {
			return ErrAccountStorageItem
		}

		if bytes.Compare(va.Bytes(), vb.Bytes()) != 0 {
			return ErrAccountStorageValue
		}
	}

	return nil
}

// IsDifferentToSubstate compares the substate account
// and returns an error if and only if the accounts are different.
func (a *Account) IsDifferentToSubstate(b *substate.Account, block uint64, address string, v func(error)) {
	// nonce must be the same
	if a.Nonce != b.Nonce {
		v(fmt.Errorf("%d - %s %v - expected: %v, world-state %v", block, address, ErrAccountNonce, b.Nonce, a.Nonce))
	}

	// balance must be the same
	if a.Balance.Cmp(b.Balance) != 0 {
		v(fmt.Errorf("%d - %s %v - expected: %v, world-state %v", block, address, ErrAccountBalance, b.Balance, a.Balance))
	}

	// storage must be initialized if Account storage is initialized
	if (a.Storage == nil && b.Storage != nil) || (a.Storage != nil && b.Storage == nil) {
		v(ErrAccountStorage)
	}

	// if there is no storage, we are done
	if b.Storage == nil {
		return
	}

	// -----------------------
	// expensive checks below
	// -----------------------

	// code must be the same
	if bytes.Compare(a.Code, b.Code) != 0 {
		v(fmt.Errorf("%d - %s %v - expected: %v, world-state %v", block, address, ErrAccountCode, b.Code, a.Code))
	}

	// compare storage content; we already know both have the same number of items
	for k, vb := range b.Storage {
		if bytes.Compare(vb.Bytes(), hash.Zero.Bytes()) == 0 {
			continue
		}

		kk := slotHash(k.Bytes())
		va, ok := a.Storage[kk]
		if !ok {
			v(fmt.Errorf("%d - %s %v - key: %v, expected: %v", block, address, ErrAccountStorageItem, k, vb.String()))
			continue
		}

		if bytes.Compare(va.Bytes(), vb.Bytes()) != 0 {
			v(fmt.Errorf("%d - %s %v - key: %v, expected: %v, world-state: %v", block, address, ErrAccountStorageValue, k, vb.String(), va.Hex()))
		}
	}
}

func slotHash(k []byte) common.Hash {
	validateHasher.Reset()
	validateHasher.Write(k)
	return common.BytesToHash(validateHasher.Sum(nil))
}
