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