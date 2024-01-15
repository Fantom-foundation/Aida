package transaction

import (
	"github.com/ethereum/go-ethereum/common"
)

// Alloc represents an interface for managing and interacting with a collection of Ethereum-like accounts.
type Alloc interface {
	// Get retrieves the account associated with the given address.
	// Get should return nil if account is not found.
	Get(addr common.Address) Account

	Has(addr common.Address) bool

	// Add adds the provided account to the collection, associated with the given address.
	Add(addr common.Address, acc Account)

	// ForEach iterates over each account in the collection and
	// invokes the provided accountHandler function for each account.
	ForEach(accountHandler)

	// Len returns the number of accounts in the collection.
	Len() int

	// Equal checks if the current allocation is equal to the provided allocation.
	// Two allocations are considered equal if they have the same accounts associated with
	// the same addresses. If any account is missing, allocs are considered non-equal.
	// Note: Have a look at allocEqual()
	Equal(Alloc) bool

	// Delete the record for given address
	Delete(addr common.Address)
}

type accountHandler func(addr common.Address, acc Account)

func allocEqual(x, y Alloc) (isEqual bool) {
	if x.Len() != y.Len() {
		return false
	}

	x.ForEach(func(addr common.Address, acc Account) {
		yVal := y.Get(addr)
		if yVal == nil {
			isEqual = false
			return
		}

		if !yVal.Equal(yVal) {
			isEqual = false
			return
		}
	})

	return true
}
