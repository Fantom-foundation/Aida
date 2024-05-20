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

package utils

import (
	"fmt"

	substatecontext "github.com/Fantom-foundation/Aida/txcontext/substate"
	"github.com/Fantom-foundation/Substate/db"
	"github.com/Fantom-foundation/Substate/substate"
	substatetypes "github.com/Fantom-foundation/Substate/types"
)

// GenerateUpdateSet generates an update set for a block range.
func GenerateUpdateSet(first uint64, last uint64, cfg *Config, aidaDb db.BaseDB) (substate.WorldState, []substatetypes.Address, error) {
	var (
		deletedAccountDB *db.DestroyedAccountDB
		deletedAccounts  []substatetypes.Address
	)
	sdb := db.MakeDefaultSubstateDBFromBaseDB(aidaDb)
	stateIter := sdb.NewSubstateIterator(int(first), cfg.Workers)
	update := make(substate.WorldState)
	defer stateIter.Release()

	// Todo rewrite in wrapping functions
	deletedAccountDB = db.MakeDefaultDestroyedAccountDBFromBaseDB(aidaDb)
	for stateIter.Next() {
		tx := stateIter.Value()
		// exceeded block range?
		if tx.Block > last {
			break
		}

		// if this transaction has suicided accounts, clear their states.
		destroyed, resurrected, err := deletedAccountDB.GetDestroyedAccounts(tx.Block, tx.Transaction)
		if err != nil {
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
		update.Merge(tx.OutputSubstate)
	}
	return update, deletedAccounts, nil
}

// GenerateWorldStateFromUpdateDB generates an initial world-state
// from pre-computed update-set
func GenerateWorldStateFromUpdateDB(cfg *Config, target uint64) (substate.WorldState, error) {
	ws := make(substate.WorldState)
	block := uint64(0)
	// load pre-computed update-set from update-set db
	udb, err := db.NewDefaultUpdateDB(cfg.AidaDb)
	if err != nil {
		return nil, err
	}
	defer udb.Close()
	updateIter := udb.NewUpdateSetIterator(block, target)
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
		ws.Merge(blk.WorldState)
		block++
	}
	updateIter.Release()

	// advance from the latest precomputed updateset to the target block
	update, _, err := GenerateUpdateSet(block, target, cfg, udb)
	if err != nil {
		return nil, err
	}
	ws.Merge(update)
	err = DeleteDestroyedAccountsFromWorldState(substatecontext.NewWorldState(ws), cfg, target)
	return ws, err
}

// ClearAccountStorage clears storage of all input accounts.
func ClearAccountStorage(update substate.WorldState, accounts []substatetypes.Address) {
	for _, addr := range accounts {
		if _, found := update[addr]; found {
			update[addr].Storage = make(map[substatetypes.Hash]substatetypes.Hash)
		}
	}
}
