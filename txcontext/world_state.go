// Copyright 2024 Fantom Foundation
// This file is part of Aida Testing Infrastructure for Sonic
//
// Aida is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Aida is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with Aida. If not, see <http://www.gnu.org/licenses/>.

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
