package transaction

import (
	"fmt"
	"sort"
	"strings"

	"github.com/ethereum/go-ethereum/common"
)

// WorldState represents an interface for managing and interacting with a collection of Ethereum-like accounts.
type WorldState interface {
	// Get retrieves the account associated with the given address.
	// Get should return nil if account is not found.
	Get(addr common.Address) Account

	Has(addr common.Address) bool

	// Add adds the provided account to the collection, associated with the given address.
	Add(addr common.Address, acc Account)

	// ForEachAccount iterates over each account in the collection and
	// invokes the provided AccountHandler function for each account.
	ForEachAccount(AccountHandler)

	// Len returns the number of accounts in the collection.
	Len() int

	// Equal checks if the current allocation is equal to the provided allocation.
	// Two allocations are considered equal if they have the same accounts associated with
	// the same addresses. If any account is missing, allocs are considered non-equal.
	// Note: Have a look at WorldStateEqual()
	Equal(WorldState) bool

	// Delete the record for given address
	Delete(addr common.Address)

	// String returns human-readable version of alloc.
	// Note: Have a look at WorldStateString()
	String() string
}

type AccountHandler func(addr common.Address, acc Account)

func WorldStateEqual(x, y WorldState) (isEqual bool) {
	if x.Len() != y.Len() {
		return false
	}

	x.ForEachAccount(func(addr common.Address, acc Account) {
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

func WorldStateString(a WorldState) string {
	builder := strings.Builder{}
	builder.WriteString(fmt.Sprintf("SubstateAlloc{\n\tsize: %d\n", a.Len()))
	var addresses []common.Address

	a.ForEachAccount(func(addr common.Address, acc Account) {
		addresses = append(addresses, addr)
	})

	sort.Slice(addresses, func(i, j int) bool { return addresses[i].String() < addresses[j].String() })

	builder.WriteString("\tAccounts:\n")
	for _, addr := range addresses {
		builder.WriteString(fmt.Sprintf("\t\t%x: %v\n", addr, a.Get(addr).String()))
	}
	builder.WriteString("}")
	return builder.String()
}
