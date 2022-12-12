package trace

import (
	"context"
	"fmt"

	"github.com/Fantom-foundation/Aida/world-state/db/snapshot"
	"github.com/ethereum/go-ethereum/substate"
)

// generateUpdateSet generates an update set for a block range.
func generateUpdateSet(first uint64, last uint64, numWorkers int) substate.SubstateAlloc {
	stateIter := substate.NewSubstateIterator(first, numWorkers)
	defer stateIter.Release()
	update := make(substate.SubstateAlloc)
	for stateIter.Next() {
		tx := stateIter.Value()
		// exceeded block range?
		if tx.Block > last {
			break
		}
		// merge output sub-state to update
		update.Merge(tx.Substate.OutputAlloc)
	}
	return update
}

// generateWorldStateFromUpdateDB generates an initial world-state
// from pre-computed update-set
func generateWorldStateFromUpdateDB(path string, target uint64, numWorkers int) (substate.SubstateAlloc, error) {
	ws := make(substate.SubstateAlloc)
	blockPos := uint64(FirstSubstateBlock - 1)
	if target < blockPos {
		return nil, fmt.Errorf("Error: the target block, %v, is earlier than the initial world state block, %v. The world state is not loaded.\n", target, blockPos)
	}
	// load pre-computed update-set from update-set db
	db := substate.OpenUpdateDBReadOnly(path)
	defer db.Close()
	updateIter := substate.NewUpdateSetIterator(db, blockPos, 1)
	for updateIter.Next() {
		blk := updateIter.Value()
		if blk.Block > target {
			break
		}
		blockPos = blk.Block
		ws.Merge(*blk.UpdateSet)
	}
	updateIter.Release()

	// advance from the latest precomputed block to the target block
	advanceWorldState(ws, blockPos+1, target, numWorkers)

	return ws, nil
}

// generateWorldState generates an initial world-state for a block.
func generateWorldState(path string, block uint64, numWorkers int) (substate.SubstateAlloc, error) {
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
	advanceWorldState(ws, FirstSubstateBlock, block, numWorkers)

	return ws, nil
}

// advanceWorldState updates the world state to the state of the last block.
func advanceWorldState(ws substate.SubstateAlloc, first uint64, block uint64, numWorkers int) {
	// generate an update from the current block to the last block
	update := generateUpdateSet(first, block, numWorkers)
	ws.Merge(update)
}
