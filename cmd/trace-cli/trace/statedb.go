package trace

import (
	"fmt"
	"io/fs"
	"path/filepath"

	"github.com/Fantom-foundation/Aida/tracer/state"
	"github.com/ethereum/go-ethereum/substate"
)

// makeStateDB creates a new DB instance based on cli argument.
func makeStateDB(directory, impl, variant string) (state.StateDB, error) {
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
	for addr, account := range ws {
		db.CreateAccount(addr)
		db.AddBalance(addr, account.Balance)
		db.SetNonce(addr, account.Nonce)
		db.SetCode(addr, account.Code)
		for key, value := range account.Storage {
			db.SetState(addr, key, value)
		}
	}
	// intermediate root implecitly calls commit
	// don't delete empty objects
	db.Commit(false)
}

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
