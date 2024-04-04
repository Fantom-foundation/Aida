package utils

import (
	"context"
	"errors"
	"fmt"

	substatecontext "github.com/Fantom-foundation/Aida/txcontext/substate"
	"github.com/Fantom-foundation/Aida/world-state/db/snapshot"
	"github.com/Fantom-foundation/Substate/db"
	"github.com/Fantom-foundation/Substate/substate"
	"github.com/Fantom-foundation/Substate/types"
	"github.com/syndtr/goleveldb/leveldb"
)

// GenerateUpdateSet generates an update set for a block range.
func GenerateUpdateSet(first uint64, last uint64, cfg *Config) (substate.WorldState, []types.Address, error) {
	var (
		deletedAccountDB *db.DestroyedAccountDB
		deletedAccounts  []types.Address
		err              error
	)
	sdb, err := db.NewDefaultSubstateDB(cfg.AidaDb)
	if err != nil {
		return nil, nil, fmt.Errorf("cannot open aida-db; %w", err)
	}
	defer sdb.Close()

	stateIter := sdb.NewSubstateIterator(int(first), cfg.Workers)
	update := make(substate.WorldState)
	defer stateIter.Release()

	// Todo rewrite in wrapping functions
	deletedAccountDB, err = db.OpenDestroyedAccountDBReadOnly(cfg.DeletionDb)
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
	update, _, err := GenerateUpdateSet(block, target, cfg)
	if err != nil {
		return nil, err
	}
	ws.Merge(update)
	err = DeleteDestroyedAccountsFromWorldState(substatecontext.NewWorldState(ws), cfg, target)
	return ws, err
}

// ClearAccountStorage clears storage of all input accounts.
func ClearAccountStorage(update substate.WorldState, accounts []types.Address) {
	for _, addr := range accounts {
		if _, found := update[addr]; found {
			update[addr].Storage = make(map[types.Hash]types.Hash)
		}
	}
}

// GenerateFirstOperaWorldState generates an initial world-state for a block.
func GenerateFirstOperaWorldState(worldStateDbDir string, cfg *Config) (substate.WorldState, error) {
	worldStateDB, err := snapshot.OpenStateDB(worldStateDbDir)
	if err != nil {
		return nil, err
	}
	defer snapshot.MustCloseStateDB(worldStateDB)
	ws, err := worldStateDB.ToWorldState(context.Background())
	return ws, err
}
