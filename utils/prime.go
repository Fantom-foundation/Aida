package utils

import (
	"fmt"
	"math/rand"
	"sort"
	"time"

	"github.com/Fantom-foundation/Aida/state"
	substate "github.com/Fantom-foundation/Substate"
	"github.com/ethereum/go-ethereum/common"
	"github.com/op/go-logging"
)

type ProgressTracker struct {
	step   int             // step counter
	target int             // total number of steps
	start  time.Time       // start time
	last   time.Time       // last reported time
	rate   float64         // priming rate
	log    *logging.Logger // Message logger
}

// NewProgressTracker creates a new progress tracer
func NewProgressTracker(target int, log *logging.Logger) *ProgressTracker {
	now := time.Now()
	return &ProgressTracker{
		step:   0,
		target: target,
		start:  now,
		last:   now,
		rate:   0.0,
		log:    log,
	}
}

// PrintProgress reports priming progress
func (pt *ProgressTracker) PrintProgress() {
	const printFrequency = 500_000 // report after x steps
	pt.step++
	if pt.step%printFrequency == 0 {
		now := time.Now()
		currentRate := printFrequency / now.Sub(pt.last).Seconds()
		pt.rate = currentRate*0.1 + pt.rate*0.9
		pt.last = now
		progress := float32(pt.step) / float32(pt.target)
		time := int(now.Sub(pt.start).Seconds())
		eta := int(float64(pt.target-pt.step) / pt.rate)
		pt.log.Infof("\t\tLoading state ... %8.1f slots/s, %5.1f%%, time: %d:%02d, ETA: %d:%02d", currentRate, progress*100, time/60, time%60, eta/60, eta%60)
	}
}

// PrimeStateDB primes database with accounts from the world state.
func PrimeStateDB(ws substate.SubstateAlloc, db state.StateDB, cfg *Config, log *logging.Logger) {
	load := db.StartBulkLoad()

	numValues := 0
	for _, account := range ws {
		numValues += len(account.Storage)
	}
	log.Infof("\tLoading %d accounts with %d values ..\n", len(ws), numValues)

	pt := NewProgressTracker(numValues, log)
	if cfg.PrimeRandom {
		//if 0, commit once after priming all accounts
		if cfg.PrimeThreshold == 0 {
			cfg.PrimeThreshold = len(ws)
		}
		PrimeStateDBRandom(ws, db, load, cfg, pt)
	} else {
		for addr, account := range ws {
			primeOneAccount(addr, account, db, load, pt)
		}
	}
	log.Infof("\t\tHashing and flushing ...\n")
	if err := load.Close(); err != nil {
		panic(fmt.Errorf("failed to prime StateDB: %v", err))
	}
}

// primeOneAccount initializes an account on stateDB with substate
func primeOneAccount(addr common.Address, account *substate.SubstateAccount, db state.StateDB, load state.BulkLoad, pt *ProgressTracker) {
	load.CreateAccount(addr)
	load.SetBalance(addr, account.Balance)
	load.SetNonce(addr, account.Nonce)
	load.SetCode(addr, account.Code)
	for key, value := range account.Storage {
		load.SetState(addr, key, value)
		pt.PrintProgress()
	}
}

// PrimeStateDBRandom primes database with accounts from the world state in random order.
func PrimeStateDBRandom(ws substate.SubstateAlloc, db state.StateDB, load state.BulkLoad, cfg *Config, pt *ProgressTracker) {
	contracts := make([]string, 0, len(ws))
	for addr := range ws {
		contracts = append(contracts, addr.Hex())
	}

	sort.Strings(contracts)
	// shuffle contract order
	rand.NewSource(cfg.PrimeSeed)
	rand.Shuffle(len(contracts), func(i, j int) {
		contracts[i], contracts[j] = contracts[j], contracts[i]
	})

	for _, c := range contracts {
		addr := common.HexToAddress(c)
		account := ws[addr]
		primeOneAccount(addr, account, db, load, pt)

	}
}

// GenerateWorldStateAndPrime
func LoadWorldStateAndPrime(db state.StateDB, cfg *Config, target uint64) error {
	log := NewLogger(cfg.LogLevel, "Priming")

	blockPos := uint64(FirstSubstateBlock - 1)
	if target < blockPos {
		return fmt.Errorf("the target block, %v, is earlier than the initial world state block, %v. The world state is not loaded.\n", target, blockPos)
	}
	// load pre-computed update-set from update-set db
	udb := substate.OpenUpdateDBReadOnly(cfg.UpdateDb)
	defer udb.Close()
	updateIter := substate.NewUpdateSetIterator(udb, blockPos, target)
	var (
		totalSize uint64
		maxSize   uint64 = cfg.UpdateCacheSize
	)
	update := make(substate.SubstateAlloc)
	setCount := 0

	for updateIter.Next() {
		newSet := updateIter.Value()
		if newSet.Block > target {
			break
		}
		blockPos = newSet.Block

		// Prime StateDB
		incrementalSize := update.EstimateIncrementalSize(*newSet.UpdateSet)
		if totalSize+incrementalSize > maxSize {
			log.Infof("\tPriming...")
			PrimeStateDB(update, db, cfg, log)
			totalSize = 0
			update = make(substate.SubstateAlloc)
		}

		// Reset accessed storage locationas of suicided accounts prior to updateset block.
		// The known accessed storage locations in the updateset range has already been
		// reset when generating the update set database.
		ClearAccountStorage(update, newSet.DeletedAccounts)
		// if exists in DB, suicide
		// TODO may aggregate list and delete only once before priming
		SuicideAccounts(db, newSet.DeletedAccounts, setCount)

		update.Merge(*newSet.UpdateSet)
		totalSize += incrementalSize
		log.Infof("\tMerge update set at block %v. New toal size %v MiB (+%v MiB)", newSet.Block, totalSize>>20, incrementalSize>>20)
		setCount++
	}
	// prime the remaining from updateset
	PrimeStateDB(update, db, cfg, log)
	updateIter.Release()

	// advance from the latest precomputed state to the target block
	if blockPos < target {
		log.Infof("\tPriming from substate from block %v", blockPos)
		update = generateUpdateSet(blockPos+1, target, cfg)
		PrimeStateDB(update, db, cfg, log)
	}

	return nil
}

// SuicideAccounts clears storage of all input accounts.
func SuicideAccounts(db state.StateDB, accounts []common.Address, offset int) {
	db.BeginSyncPeriod(0)
	db.BeginBlock(uint64(4000000 + offset)) //TODO change block
	db.BeginTransaction(0)
	for _, addr := range accounts {
		if db.Exist(addr) {
			db.Suicide(addr)
		}
	}
	db.EndTransaction()
	db.EndBlock()
	db.EndSyncPeriod()
}
