package trace

import (
	"fmt"
	"io/fs"
	"path/filepath"

	"github.com/Fantom-foundation/Aida/tracer/state"
	"github.com/ethereum/go-ethereum/substate"
	"github.com/urfave/cli/v2"
)

const FirstBlock = 4564026
/////////////////////////////////////////////////
// World State generation
/////////////////////////////////////////////////

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
		// Refine it so that we don't have redundant 
		// entries by double-checking the InputAlloc
		// Calculate the difference between 
		// tx.Substate.InputAlloc and tx.Substate.OutputAlloc
		// => In the past we observed redundant entries
		// caused by reads. Note that we need to keep 
		// 0 entries in Output (deletions!).
		// Unfortunately, we don't have a diff semantics 
		// for Nonce, Balance, and Code. This will be always
		// overwritten. 
		// Optimisation: If there is the same (key,value) pair
		// in input and output, we can remove the (key, value) pair
		// from Storage in output.
		update.Merge(tx.Substate.OutputAlloc)
	}
	return update
}

// generateWorldState generates an initial world-state for a block.
func  generateWorldState(block uint64, numWorkers int) substate.SubstateAlloc {
	// Todo load initial worldstate for block 4.5M
	// ws := loadInitWorldState()
	ws := make(substate.SubstateAlloc)
	
	// generate world state for block 
	update := generateUpdateSet(FirstBlock, block, numWorkers)
	ws.Merge(update)
	return ws
}

// advanceWorldState updates the world state to the state of the last block.
func advanceWorldState(ws substate.SubstateAlloc, first uint64, block uint64, numWorkers int) {
	update := generateUpdateSet(first, block, numWorkers)
	ws.Merge(update)
}

/////////////////////////////////////////////////
// State DB generation
/////////////////////////////////////////////////

// makeStateDB creates a new DB instance based on cli argument.
func makeStateDB(directory string, cliCtx *cli.Context) (state.StateDB, error) {
	impl := cliCtx.String(stateDbImplementation.Name)
	variant := cliCtx.String(stateDbVariant.Name)
	switch impl {
	case "memory":
		return state.MakeGethInMemoryStateDB(variant)
	case "geth":
		return state.MakeGethStateDB(directory, variant)
	case "carmen":
		return state.MakeCarmenStateDB(directory, variant)
	}
	return nil, fmt.Errorf("Unknown DB implementation (--%v): %v", stateDbImplementation.Name, impl)
}

// primeStateDB primes database with accounts from the world state.
func primeStateDB(ws substate.SubstateAlloc, db state.StateDB) {
	// TODO: Extend so that priming order is randomized
	for  addr, account := range ws {
		db.CreateAccount(addr)	
		db.AddBalance(addr, account.Balance)
		db.SetNonce(addr, account.Nonce)
		//db.SetCode(addr, account.Code)
		for key, value := range account.Storage {
			db.SetState(addr, key, value)
		}
	}
}

/////////////////////////////////////////////////
// Validation
/////////////////////////////////////////////////
// validateDatabase validates whether the world-state is contained in the db object
// NB: We can only check what must be in the db (but cannot check 
// whether db stores more)
// Perhaps reuse some of the code from 
func validateStateDB(ws substate.SubstateAlloc, db state.StateDB) error {
	// TODO: Extend so that priming order is randomized
	for  addr, account := range ws {
		if  !db.Exist(addr) {
			return fmt.Errorf("Account %v does not exist", addr.Hex())
		}
		if account.Balance.Cmp(db.GetBalance(addr)) != 0 {
			// TODO: print more detail
			return fmt.Errorf("Failed to validate balance for account %v", addr.Hex())
		}
		if db.GetNonce(addr) != account.Nonce {
			// TODO: print more detail
			return fmt.Errorf("Failed to validate nonce for account %v", addr.Hex())
		}
		/*
		if  db.GetCode(addr) != account.GetCode() {
			// TODO: print more detail
			log.Fatalf("Failed to validate code for account %v", addr.Hex())
		}
		*/
		for key, value := range account.Storage {
			if db.GetState(addr, key) != value {
				// TODO: print more detail
				return fmt.Errorf("Failed to validate nonce for account %v", addr.Hex())
			}
		}

	}
	return nil
}

// Compare state after replaying traces with recorded state.
func compareSubstateStorage(recordedAlloc substate.SubstateAlloc, traceAlloc substate.SubstateAlloc) error {
	for account, recordAccount := range recordedAlloc {
		// account exists in both substate
		if replayAccout, exist := traceAlloc[account]; exist {
			for k, xv := range recordAccount.Storage {
				// mismatched value or key dones't exist
				if yv, exist := replayAccout.Storage[k]; !exist || xv != yv {
					return fmt.Errorf("Error: mismatched value at storage key %v. want %v have %v\n", k, xv, yv)
				}
			}
			for k, yv := range replayAccout.Storage {
				// key exists when expecting nil
				if xv, exist := recordAccount.Storage[k]; !exist {
					return fmt.Errorf("Error: mismatched value at storage key %v. want %v have %v\n", k, xv, yv)
				}
			}
		} else {
			if len(recordAccount.Storage) > 0 {
				return fmt.Errorf("Error: account %v doesn't exist\n", account)
			}
			//else ignores accounts which has no storage
		}
	}

	// checks for unexpected accounts in replayed substate
	for account := range traceAlloc {
		if _, exist := recordedAlloc[account]; !exist {
			return fmt.Errorf("Error: unexpected account %v\n", account)
		}
	}
	return nil
}


/////////////////////////////////////////////////
// Utility functions
/////////////////////////////////////////////////

// getDirectorySize computes the size of all files in the given directoy in bytes.
func getDirectorySize(directory string) int64 {
	var sum int64 = 0
	filepath.Walk(directory, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() {
			sum += info.Size()
		}
		return nil
	})
	return sum
}
