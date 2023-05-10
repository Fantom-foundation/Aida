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

const bulkLoadCap = 100_000

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
func PrimeStateDB(ws substate.SubstateAlloc, db state.StateDB, block uint64, cfg *Config, log *logging.Logger) error {
	numValues := 0 // number of storage values
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
		if err := PrimeStateDBRandom(ws, db, block, cfg, pt); err != nil {
			return fmt.Errorf("failed to prime StateDB: %v", err)
		}
	} else {
		load := db.StartBulkLoad(block)
		step := 0
		for addr, account := range ws {
			primeOneAccount(addr, account, db, load, pt, cfg.PrimeForce)
			step++
			// commit to stateDB after process n accounts
			if step%bulkLoadCap == 0 {
				if err := load.Close(); err != nil {
					return fmt.Errorf("failed to prime StateDB: %v", err)
				}
				block++
				step = 0
				load = db.StartBulkLoad(block)
			}
		}
		if err := load.Close(); err != nil {
			return fmt.Errorf("failed to prime StateDB: %v", err)
		}
	}
	log.Infof("\t\tPriming completed ...\n")
	return nil
}

// primeOneAccount initializes an account on stateDB with substate
func primeOneAccount(addr common.Address, account *substate.SubstateAccount, db state.StateDB, load state.BulkLoad, pt *ProgressTracker, force bool) {
	// if an account was previously primed, skip account creation.
	if force || !db.Exist(addr) {
		load.CreateAccount(addr)
	}
	load.SetBalance(addr, account.Balance)
	load.SetNonce(addr, account.Nonce)
	load.SetCode(addr, account.Code)
	for key, value := range account.Storage {
		load.SetState(addr, key, value)
		pt.PrintProgress()
	}
}

// PrimeStateDBRandom primes database with accounts from the world state in random order.
func PrimeStateDBRandom(ws substate.SubstateAlloc, db state.StateDB, block uint64, cfg *Config, pt *ProgressTracker) error {
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

	load := db.StartBulkLoad(block)
	step := 0
	for _, c := range contracts {
		addr := common.HexToAddress(c)
		account := ws[addr]
		primeOneAccount(addr, account, db, load, pt, cfg.PrimeForce)
		step++
		// commit to stateDB after process n accounts and start a new buck load
		if step%bulkLoadCap == 0 {
			if err := load.Close(); err != nil {
				return err
			}
			block++
			step = 0
			load = db.StartBulkLoad(block)
		}

	}
	err := load.Close()
	return err
}

// GenerateWorldStateAndPrime
func LoadWorldStateAndPrime(db state.StateDB, cfg *Config, target uint64) error {
	log := NewLogger(cfg.LogLevel, "Priming")

	var (
		totalSize  uint64
		maxSize    uint64 = cfg.UpdateCacheSize
		blockPos   uint64 = FirstSubstateBlock - 1
		primeBlock uint64
	)

	if target < blockPos {
		return fmt.Errorf("the target block, %v, is earlier than the initial world state block, %v. The world state is not loaded.\n", target, blockPos)
	}
	// load pre-computed update-set from update-set db
	udb := substate.OpenUpdateDBReadOnly(cfg.UpdateDb)
	defer udb.Close()
	updateIter := substate.NewUpdateSetIterator(udb, blockPos, target)
	update := make(substate.SubstateAlloc)

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
			if err := PrimeStateDB(update, db, primeBlock, cfg, log); err != nil {
				return fmt.Errorf("failed to prime StateDB: %v", err)
			}
			totalSize = 0
			primeBlock = newSet.Block
			update = make(substate.SubstateAlloc)
		}

		// Reset accessed storage locationas of suicided accounts prior to updateset block.
		// The known accessed storage locations in the updateset range has already been
		// reset when generating the update set database.
		ClearAccountStorage(update, newSet.DeletedAccounts)
		// if exists in DB, suicide
		// TODO may aggregate list and delete only once before priming
		SuicideAccounts(db, newSet.DeletedAccounts, primeBlock, log)
		primeBlock++

		update.Merge(*newSet.UpdateSet)
		totalSize += incrementalSize
		log.Infof("\tMerge update set at block %v. New toal size %v MiB (+%v MiB)", newSet.Block, totalSize>>20, incrementalSize>>20)
	}
	// prime the remaining from updateset
	if err := PrimeStateDB(update, db, primeBlock, cfg, log); err != nil {
		return fmt.Errorf("failed to prime StateDB: %v", err)
	}
	primeBlock = blockPos - 1 //need at least 2 blocks for priming from substate
	updateIter.Release()
	update = make(substate.SubstateAlloc)

	// advance from the latest precomputed state to the target block
	if blockPos < target {
		log.Infof("\tPriming from substate from block %v", blockPos)
		update, deletedAccounts := generateUpdateSet(blockPos+1, target, cfg)
		SuicideAccounts(db, deletedAccounts, primeBlock, log)
		primeBlock++
		if err := PrimeStateDB(update, db, primeBlock, cfg, log); err != nil {
			return fmt.Errorf("failed to prime StateDB: %v", err)
		}
	}

	// delete destroyed accounts from stateDB
	log.Notice("Delete destroyed accounts")
	// remove destroyed accounts until one block before the first block
	err := DeleteDestroyedAccountsFromStateDB(db, cfg, cfg.First-1)

	return err
}

// SuicideAccounts clears storage of all input accounts.
func SuicideAccounts(db state.StateDB, accounts []common.Address, block uint64, log *logging.Logger) {
	log.Info("Remove suicided accounts from stateDB.")
	count := 0
	db.BeginSyncPeriod(0)
	db.BeginBlock(block)
	db.BeginTransaction(0)
	for _, addr := range accounts {
		if db.Exist(addr) {
			db.Suicide(addr)
			count++
		}
	}
	db.EndTransaction()
	db.EndBlock()
	db.EndSyncPeriod()
	log.Infof("\t %v accounts were removed.", count)
}
