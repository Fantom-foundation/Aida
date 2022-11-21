package trace

import (
	"fmt"
	"io/fs"
	"log"
	"math/rand"
	"path/filepath"
	"sort"

	"github.com/Fantom-foundation/Aida/tracer/state"
	"github.com/ethereum/go-ethereum/common"
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
func primeStateDB(ws substate.SubstateAlloc, db state.StateDB, cfg *TraceConfig) {
	if cfg.primeRandom {
		//if 0, commit once after priming all accounts
		if cfg.primeThreshold == 0 {
			cfg.primeThreshold = len(ws)
		}
		log.Printf("Prime stateDB with seed %v and k %v\n", cfg.primeSeed, cfg.primeThreshold)
		primeStateDBRandom(ws, db, cfg)
	} else {
		log.Printf("Prime stateDB\n")
		for addr, account := range ws {
			primeOneAccount(addr, account, db)
		}
		// don't delete empty objects
		db.Commit(false)
	}
}

// primeOneAccount initializes an account on stateDB with substate
func primeOneAccount(addr common.Address, account *substate.SubstateAccount, db state.StateDB) {
	db.CreateAccount(addr)
	db.AddBalance(addr, account.Balance)
	db.SetNonce(addr, account.Nonce)
	db.SetCode(addr, account.Code)
	for key, value := range account.Storage {
		db.SetState(addr, key, value)
	}
}

// primeStateDBRandom primes database with accounts from the world state in random order.
func primeStateDBRandom(ws substate.SubstateAlloc, db state.StateDB, cfg *TraceConfig) {
	contracts := make([]string, 0, len(ws))
	for addr := range ws {
		contracts = append(contracts, addr.Hex())
	}

	sort.Strings(contracts)
	// shuffle contract order
	rand.NewSource(cfg.primeSeed)
	rand.Shuffle(len(contracts), func(i, j int) {
		contracts[i], contracts[j] = contracts[j], contracts[i]
	})

	for i, c := range contracts {
		if i%cfg.primeThreshold == 0 && i != 0 {
			db.Commit(false)
		}
		addr := common.HexToAddress(c)
		account := ws[addr]
		primeOneAccount(addr, account, db)
		// commit after k accounts have been primed

	}
	// commit the rest of accounts
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
