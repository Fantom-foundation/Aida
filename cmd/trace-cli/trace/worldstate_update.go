package trace

import (
	"context"

	"github.com/Fantom-foundation/Aida/world-state/db/snapshot"
	"github.com/ethereum/go-ethereum/substate"
)

const FirstSubstateBlock = 4564026

// generateUpdateDatabase generates an update set for a block range.
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

// generateWorldState generates an initial world-state for a block.
func generateWorldState(path string, block uint64, numWorkers int) (substate.SubstateAlloc, error) {
	// Todo load initial worldstate for block 4.5M
	worldStateDB, err := snapshot.OpenStateDB(path)
	if err != nil {
		return nil, err
	}
	defer snapshot.MustCloseStateDB(worldStateDB)
	ws, err := worldStateDB.ToSubstateAlloc(context.Background())
	if err != nil {
		return nil, err
	}

	update := generateUpdateSet(FirstSubstateBlock, block, numWorkers)
	// generate world state for block
	ws.Merge(update)
	return ws, nil
}

// advanceWorldState updates the world state to the state of the last block.
func advanceWorldState(ws substate.SubstateAlloc, first uint64, block uint64, numWorkers int) {
	// generate an update from the current block to the last block
	update := generateUpdateSet(first, block, numWorkers)
	ws.Merge(update)
}
