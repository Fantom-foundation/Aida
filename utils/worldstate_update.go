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

package utils

import (
	"errors"
	"fmt"

	substatecontext "github.com/Fantom-foundation/Aida/txcontext/substate"
	substate "github.com/Fantom-foundation/Substate"
	"github.com/ethereum/go-ethereum/common"
	"github.com/syndtr/goleveldb/leveldb"
)

// GenerateUpdateSet generates an update set for a block range.
func GenerateUpdateSet(first uint64, last uint64, cfg *Config) (substate.SubstateAlloc, []common.Address, error) {
	var (
		deletedAccountDB *substate.DestroyedAccountDB
		deletedAccounts  []common.Address
		err              error
	)
	stateIter := substate.NewSubstateIterator(first, cfg.Workers)
	update := make(substate.SubstateAlloc)
	defer stateIter.Release()

	// Todo rewrite in wrapping functions
	deletedAccountDB, err = substate.OpenDestroyedAccountDBReadOnly(cfg.DeletionDb)
	if err != nil {
		return nil, nil, err
	}
	defer deletedAccountDB.Close()

	for stateIter.Next() {
		tx := stateIter.Value()
		// exceeded block range?
		if tx.Block > last {
			break
		}

		// if this transaction has suicided accounts, clear their states.
		destroyed, resurrected, err := deletedAccountDB.GetDestroyedAccounts(tx.Block, tx.Transaction)

		if !(err == nil || errors.Is(err, leveldb.ErrNotFound)) {
			return update, deletedAccounts, fmt.Errorf("failed to get deleted account. %v", err)
		}
		// reset storagea
		if len(destroyed) > 0 {
			deletedAccounts = append(deletedAccounts, destroyed...)
		}
		if len(resurrected) > 0 {
			deletedAccounts = append(deletedAccounts, resurrected...)
			ClearAccountStorage(update, resurrected)
		}

		// merge output substate to update
		update.Merge(tx.Substate.OutputAlloc)
	}
	return update, deletedAccounts, nil
}

// GenerateWorldStateFromUpdateDB generates an initial world-state
// from pre-computed update-set
func GenerateWorldStateFromUpdateDB(cfg *Config, target uint64) (substate.SubstateAlloc, error) {
	ws := make(substate.SubstateAlloc)
	block := uint64(0)
	// load pre-computed update-set from update-set db
	db, err := substate.OpenUpdateDBReadOnly(cfg.AidaDb)
	if err != nil {
		return nil, err
	}
	defer db.Close()
	updateIter := substate.NewUpdateSetIterator(db, block, target)
	for updateIter.Next() {
		blk := updateIter.Value()
		if blk.Block > target {
			break
		}
		block = blk.Block
		// Reset accessed storage locations of suicided accounts prior to updateset block.
		// The known accessed storage locations in the updateset range has already been
		// reset when generating the update set database.
		ClearAccountStorage(ws, blk.DeletedAccounts)
		ws.Merge(*blk.UpdateSet)
		block++
	}
	updateIter.Release()

	// advance from the latest precomputed updateset to the target block
	update, _, err := GenerateUpdateSet(block, target, cfg)
	if err != nil {
		return nil, err
	}
	ws.Merge(update)
	err = DeleteDestroyedAccountsFromWorldState(substatecontext.NewWorldState(ws), cfg, target)
	return ws, err
}

// ClearAccountStorage clears storage of all input accounts.
func ClearAccountStorage(update substate.SubstateAlloc, accounts []common.Address) {
	for _, addr := range accounts {
		if _, found := update[addr]; found {
			update[addr].Storage = make(map[common.Hash]common.Hash)
		}
	}
}
