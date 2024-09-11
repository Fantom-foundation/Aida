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

package state

import (
	"fmt"

	"github.com/Fantom-foundation/Aida/txcontext"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
)

// NewOffTheChainStateDB returns an empty in-memory *state.StateDB without disk caches
func NewOffTheChainStateDB() *state.StateDB {
	db := rawdb.NewMemoryDatabase()
	statedb, _ := state.New(types.EmptyRootHash, state.NewDatabase(db), nil)
	return statedb
}

// MakeOffTheChainStateDB returns an in-memory *state.StateDB initialized with ws
func MakeOffTheChainStateDB(alloc txcontext.WorldState, block uint64, chainConduit *ChainConduit) (StateDB, error) {
	statedb := NewOffTheChainStateDB()
	alloc.ForEachAccount(func(addr common.Address, acc txcontext.Account) {
		code := acc.GetCode()
		statedb.SetCode(addr, code)
		statedb.SetNonce(addr, acc.GetNonce())
		statedb.SetBalance(addr, acc.GetBalance(), 0)
		// DON'T USE SetStorage because it makes REVERT and dirtyStorage unavailble
		acc.ForEachStorage(func(keyHash common.Hash, valueHash common.Hash) {
			statedb.SetState(addr, keyHash, valueHash)
		})
	})

	// Commit and re-open to start with a clean state.
	_, err := statedb.Commit(block, false)
	if err != nil {
		return nil, fmt.Errorf("cannot commit offTheChainDb; %v", err)
	}

	return &gethStateDB{db: statedb, block: block, chainConduit: chainConduit}, nil
}
