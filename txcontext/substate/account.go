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

package substate

import (
	"math/big"

	"github.com/Fantom-foundation/Aida/txcontext"
	oldSubstate "github.com/Fantom-foundation/Substate"
	"github.com/ethereum/go-ethereum/common"
)

// Deprecated: This is a workaround before oldSubstate repository is migrated to new structure.
// Use NewSubstateAccount instead.
func NewAccount(acc *oldSubstate.SubstateAccount) txcontext.Account {
	return &account{acc}
}

// Deprecated: This is a workaround before oldSubstate repository is migrated to new structure.
// Use substateAccount instead.
type account struct {
	*oldSubstate.SubstateAccount
}

func (a *account) GetNonce() uint64 {
	return a.Nonce
}

func (a *account) GetBalance() *big.Int {
	return a.Balance
}

func (a *account) HasStorageAt(key common.Hash) bool {
	_, ok := a.Storage[key]
	return ok
}

func (a *account) GetStorageAt(hash common.Hash) common.Hash {
	return a.Storage[hash]
}

func (a *account) GetCode() []byte {
	return a.Code
}

func (a *account) GetStorageSize() int {
	return len(a.Storage)
}

func (a *account) ForEachStorage(h txcontext.StorageHandler) {
	for keyHash, valueHash := range a.Storage {
		h(keyHash, valueHash)
	}
}

func (a *account) String() string {
	return txcontext.AccountString(a)
}
