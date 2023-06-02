package runvm

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/Fantom-foundation/rc-testing/test/vmtest/world-state/db/snapshot"
	substate "github.com/Fantom-foundation/Substate"
	"github.com/ethereum/go-ethereum/common"
	"github.com/syndtr/goleveldb/leveldb"
)

// generateUpdateSet generates an update set for a block range.
func generateUpdateSet(first uint64, last uint64, cfg *Config) (substate.SubstateAlloc, []common.Address) {
	var (
		deletedAccountDB *substate.DestroyedAccountDB
		deletedAccounts  []common.Address
	)
	stateIter := substate.NewSubstateIterator(first, cfg.Workers)
	defer stateIter.Release()
	if cfg.HasDeletedAccounts {
		deletedAccountDB, err := substate.OpenDestroyedAccountDBReadOnly(cfg.DeletionDb)
		if err != nil {
			panic(err)
		}
		defer deletedAccountDB.Close()
	}

	update := make(substate.SubstateAlloc)
	for stateIter.Next() {
		tx := stateIter.Value()
		// exceeded block range?
		if tx.Block > last {
			break
		}

		// if this transaction has suicided accounts, clear their states.
		if cfg.HasDeletedAccounts {
			destroyed, resurrected, err := deletedAccountDB.GetDestroyedAccounts(tx.Block, tx.Transaction)

			if !(err == nil || errors.Is(err, leveldb.ErrNotFound)) {
				log.Fatalf("failed to get deleted account. %v", err)
			}
			// reset storage
			deletedAccounts = append(deletedAccounts, destroyed...)
			deletedAccounts = append(deletedAccounts, destroyed...)

			ClearAccountStorage(update, destroyed)
			ClearAccountStorage(update, resurrected)
		}

		// merge output substate to update
		update.Merge(tx.Substate.OutputAlloc)
	}
	return update, deletedAccounts
}

// GenerateWorldStateFromUpdateDB generates an initial world-state
// from pre-computed update-set
func GenerateWorldStateFromUpdateDB(cfg *Config, target uint64) (substate.SubstateAlloc, error) {
	ws := make(substate.SubstateAlloc)
	blockPos := uint64(FirstSubstateBlock - 1)
	if target < blockPos {
		return nil, fmt.Errorf("Error: the target block, %v, is earlier than the initial world state block, %v. The world state is not loaded.\n", target, blockPos)
	}
	// load pre-computed update-set from update-set db
	db, err := substate.OpenUpdateDBReadOnly(cfg.UpdateDb)
	if err != nil {
		return nil, err
	}
	defer db.Close()
	updateIter := substate.NewUpdateSetIterator(db, blockPos, target)
	for updateIter.Next() {
		blk := updateIter.Value()
		if blk.Block > target {
			break
		}
		blockPos = blk.Block
		// Reset accessed storage locations of suicided accounts prior to updateset block.
		// The known accessed storage locations in the updateset range has already been
		// reset when generating the update set database.
		ClearAccountStorage(ws, blk.DeletedAccounts)
		ws.Merge(*blk.UpdateSet)
	}
	updateIter.Release()

	// advance from the latest precomputed block to the target block
	update, _ := generateUpdateSet(blockPos+1, target, cfg)
	ws.Merge(update)

	return ws, nil
}

// ClearAccountStorage clears storage of all input accounts.
func ClearAccountStorage(update substate.SubstateAlloc, accounts []common.Address) {
	for _, addr := range accounts {
		if _, found := update[addr]; found {
			update[addr].Storage = make(map[common.Hash]common.Hash)
		}
	}
}

// GenerateWorldState generates an initial world-state for a block.
func GenerateWorldState(path string, block uint64, cfg *Config) (substate.SubstateAlloc, error) {
	worldStateDB, err := snapshot.OpenStateDB(path)
	if err != nil {
		return nil, err
	}
	defer snapshot.MustCloseStateDB(worldStateDB)
	ws, err := worldStateDB.ToSubstateAlloc(context.Background())
	if err != nil {
		return nil, err
	}

	// advance from the first block from substateDB to the target block
	update, _ := generateUpdateSet(FirstSubstateBlock, block, cfg)
	ws.Merge(update)

	return ws, nil
}
