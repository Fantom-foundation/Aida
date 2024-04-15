// Copyright 2024 Fantom Foundation
// This file is part of Aida Testing Infrastructure for Sonic
//
// Aida is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Aida is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with Aida. If not, see <http://www.gnu.org/licenses/>.

package txcontext

import (
	"bytes"
	"fmt"
	"math/big"
	"sort"
	"strings"

	"github.com/ethereum/go-ethereum/common"
)

// Account represents an interface for interacting with an Ethereum-like account.
type Account interface {
	// GetNonce returns the current nonce of the account.
	GetNonce() uint64

	// GetBalance returns the current balance of the account.
	GetBalance() *big.Int

	HasStorageAt(key common.Hash) bool

	// GetStorageAt returns the value stored at the specified storage key of the account.
	GetStorageAt(key common.Hash) common.Hash

	// GetCode returns the bytecode of the account.
	GetCode() []byte

	// GetStorageSize returns the size of Accounts Storage.
	GetStorageSize() int

	// ForEachStorage iterates over each account's storage in the collection
	// and invokes the provided AccountHandler function for each account.
	ForEachStorage(StorageHandler)

	// String returns human-readable version of alloc.
	// Note: Have a look at AccountString
	String() string
}

func NewNilAccount() Account {
	return &account{}
}

func NewAccount(code []byte, storage map[common.Hash]common.Hash, balance *big.Int, nonce uint64) Account {
	return &account{Code: code, Storage: storage, Balance: balance, Nonce: nonce}
}

type account struct {
	Code    []byte
	Storage map[common.Hash]common.Hash
	Balance *big.Int
	Nonce   uint64
}

func (a *account) GetNonce() uint64 {
	return a.Nonce
}

func (a *account) GetBalance() *big.Int {
	return new(big.Int).Set(a.Balance)
}

func (a *account) HasStorageAt(key common.Hash) bool {
	_, ok := a.Storage[key]
	return ok
}

func (a *account) GetStorageAt(key common.Hash) common.Hash {
	return a.Storage[key]
}

func (a *account) GetCode() []byte {
	return a.Code
}

func (a *account) GetStorageSize() int {
	return len(a.Storage)
}

func (a *account) ForEachStorage(h StorageHandler) {
	for k, v := range a.Storage {
		h(k, v)
	}
}

func (a *account) String() string {
	return AccountString(a)
}

type StorageHandler func(keyHash common.Hash, valueHash common.Hash)

func AccountEqual(a, y Account) (isEqual bool) {
	if a == y {
		return true
	}

	if (a == nil || y == nil) && a != y {
		return false
	}

	// check values
	equal := a.GetNonce() == y.GetNonce() &&
		a.GetBalance().Cmp(y.GetBalance()) == 0 &&
		bytes.Equal(a.GetCode(), y.GetCode()) &&
		a.GetStorageSize() == y.GetStorageSize()
	if !equal {
		return false
	}

	zeroHash := common.Hash{}
	a.ForEachStorage(func(aKey common.Hash, aHash common.Hash) {
		yHash := y.GetStorageAt(aKey)
		if yHash == zeroHash {
			isEqual = false
			return
		}

		if yHash != aHash {
			isEqual = false
			return
		}
	})

	return true
}

func AccountString(a Account) string {
	builder := strings.Builder{}
	builder.WriteString(fmt.Sprintf("Account{\n\t\t\tnonce: %d\n\t\t\tbalance %v\n", a.GetNonce(), a.GetBalance()))

	builder.WriteString("\t\t\tStorage{\n")
	var keyHashes []common.Hash

	a.ForEachStorage(func(keyHash common.Hash, _ common.Hash) {
		keyHashes = append(keyHashes, keyHash)
	})

	sort.Slice(keyHashes, func(i, j int) bool { return keyHashes[i].String() < keyHashes[j].String() })
	for _, key := range keyHashes {
		builder.WriteString(fmt.Sprintf("\t\t\t\t%v=%v\n", key, a.GetStorageAt(key)))
	}
	builder.WriteString("\t\t\t}\n\t\t}")
	return builder.String()
}
