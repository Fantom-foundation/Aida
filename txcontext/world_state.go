package txcontext

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

func NewWorldState(m map[common.Address]Account) WorldState {
	return AidaWorldState(m)
}

type AidaWorldState map[common.Address]Account

func (a AidaWorldState) String() string {
	return WorldStateString(a)
}

func (a AidaWorldState) Has(addr common.Address) bool {
	_, ok := a[addr]
	return ok
}

func (a AidaWorldState) Equal(y WorldState) bool {
	return WorldStateEqual(a, y)
}

func (a AidaWorldState) Get(addr common.Address) Account {
	acc, ok := a[addr]
	if !ok {
		return nil
	}

	return acc
}

func (a AidaWorldState) ForEachAccount(h AccountHandler) {
	for addr, acc := range a {
		h(addr, acc)
	}
}

func (a AidaWorldState) Len() int {
	return len(a)
}

func (a AidaWorldState) Delete(addr common.Address) {
	delete(a, addr)
}
func WorldStateEqual(x, y WorldState) (isEqual bool) {
	if x.Len() != y.Len() {
		return false
	}

	x.ForEachAccount(func(addr common.Address, acc Account) {
		yAcc := y.Get(addr)
		if yAcc == nil {
			isEqual = false
			return
		}

		if !AccountEqual(acc, yAcc) {
			isEqual = false
			return
		}
	})

	return true
}

func WorldStateString(a WorldState) string {
	builder := strings.Builder{}
	builder.WriteString(fmt.Sprintf("World State {\n\tsize: %d\n", a.Len()))
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
