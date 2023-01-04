package trace

import (
	"fmt"
	"io/fs"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/Fantom-foundation/Aida/tracer/state"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/substate"
)

// MakeStateDB creates a new DB instance based on cli argument.
func MakeStateDB(directory string, cfg *TraceConfig) (state.StateDB, error) {
	db, err := makeStateDBInternal(directory, cfg)
	if err != nil {
		return nil, err
	}
	if cfg.dbLogging {
		db = state.MakeLoggingStateDB(db)
	}
	return db, nil
}

// makeStateDB creates a DB instance with a potential shadow instance.
func makeStateDBInternal(directory string, cfg *TraceConfig) (state.StateDB, error) {
	if cfg.shadowImpl == "" {
		return makeStateDBVariant(directory, cfg.dbImpl, cfg.dbVariant, cfg)
	}
	primeDir := directory + "/prime"
	if err := os.MkdirAll(primeDir, 0700); err != nil {
		return nil, err
	}
	shadowDir := directory + "/shadow"
	if err := os.MkdirAll(shadowDir, 0700); err != nil {
		return nil, err
	}
	prime, err := makeStateDBVariant(primeDir, cfg.dbImpl, cfg.dbVariant, cfg)
	if err != nil {
		return nil, err
	}
	shadow, err := makeStateDBVariant(shadowDir, cfg.shadowImpl, cfg.shadowVariant, cfg)
	if err != nil {
		return nil, err
	}
	return state.MakeShadowStateDB(prime, shadow), nil
}

// makeStateDBVariant creates a DB instance of the requested kind.
func makeStateDBVariant(directory, impl, variant string, cfg *TraceConfig) (state.StateDB, error) {
	switch impl {
	case "memory":
		return state.MakeGethInMemoryStateDB(variant)
	case "geth":
		return state.MakeGethStateDB(directory, variant, cfg.archiveMode)
	case "carmen":
		return state.MakeCarmenStateDB(directory, variant)
	case "flat":
		return state.MakeFlatStateDB(directory, variant)
	}
	return nil, fmt.Errorf("unknown DB implementation (--%v): %v", stateDbImplementationFlag.Name, impl)
}

// primeStateDB primes database with accounts from the world state.
func primeStateDB(ws substate.SubstateAlloc, db state.StateDB, cfg *TraceConfig) {
	load := db.StartBulkLoad()

	numValues := 0
	for _, account := range ws {
		numValues += len(account.Storage)
	}
	log.Printf("\tLoading %d accounts with %d values ..\n", len(ws), numValues)

	j := 0
	start := time.Now()
	last := start
	rate := 0.0
	progressTracker := func() {
		const step = 100000
		j++
		if j%step == 0 {
			now := time.Now()
			currentRate := step / now.Sub(last).Seconds()
			rate = currentRate*0.1 + rate*0.9
			last = now
			progress := float32(j) / float32(numValues)
			time := int(now.Sub(start).Seconds())
			eta := int(float64(numValues-j) / rate)
			log.Printf("\t\tLoading state ... %8.1f slots/s, %5.1f%%, time: %d:%02d, ETA: %d:%02d\n", currentRate, progress*100, time/60, time%60, eta/60, eta%60)
		}
	}

	if cfg.primeRandom {
		//if 0, commit once after priming all accounts
		if cfg.primeThreshold == 0 {
			cfg.primeThreshold = len(ws)
		}
		primeStateDBRandom(ws, load, cfg, progressTracker)
	} else {
		for addr, account := range ws {
			primeOneAccount(addr, account, load, progressTracker)
		}

	}
	log.Printf("\t\tHashing and flushing ...\n")
	load.Close()
}

// primeOneAccount initializes an account on stateDB with substate
func primeOneAccount(addr common.Address, account *substate.SubstateAccount, db state.BulkLoad, afterLoad func()) {
	db.CreateAccount(addr)
	db.SetBalance(addr, account.Balance)
	db.SetNonce(addr, account.Nonce)
	db.SetCode(addr, account.Code)
	for key, value := range account.Storage {
		db.SetState(addr, key, value)
		if afterLoad != nil {
			afterLoad()
		}
	}
}

// primeStateDBRandom primes database with accounts from the world state in random order.
func primeStateDBRandom(ws substate.SubstateAlloc, db state.BulkLoad, cfg *TraceConfig, afterLoad func()) {
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

	for _, c := range contracts {
		addr := common.HexToAddress(c)
		account := ws[addr]
		primeOneAccount(addr, account, db, afterLoad)

	}
}

// deleteDestroyedAccountsFromWorldState removes previously suicided accounts from
// the world state.
func deleteDestroyedAccountsFromWorldState(ws substate.SubstateAlloc, directory string, target uint64) error {
	if _, err := os.Stat(directory); os.IsNotExist(err) {
		log.Printf("WARNING: deleted-account-dir is not provided or does not exist")
		return nil
	}
	src := substate.OpenDestroyedAccountDBReadOnly(directory)
	list, err := src.GetAccountsDestroyedInRange(0, target)
	if err != nil {
		return err
	}
	for _, cur := range list {
		if _, found := ws[cur]; found {
			delete(ws, cur)
		}
	}
	return nil
}

// deleteDestroyedAccountsFromStateDB performs suicide operations on previously
// self-destructed accounts.
func deleteDestroyedAccountsFromStateDB(db state.StateDB, directory string, target uint64) error {
	if _, err := os.Stat(directory); os.IsNotExist(err) {
		log.Printf("WARNING: deleted-account-dir is not provided or does not exist")
		return nil
	}
	src := substate.OpenDestroyedAccountDBReadOnly(directory)
	list, err := src.GetAccountsDestroyedInRange(0, target)
	if err != nil {
		return err
	}
	log.Printf("Deleting %d accounts ..\n", len(list))
	db.BeginEpoch(0)
	db.BeginBlock(0)
	db.BeginTransaction(0)
	for _, cur := range list {
		db.Suicide(cur)
	}
	db.Finalise(true)
	db.EndTransaction()
	db.EndBlock()
	db.EndEpoch()
	return nil
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
