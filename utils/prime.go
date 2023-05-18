package utils

import (
	"fmt"
	"math/rand"
	"sort"
	"time"

	"github.com/Fantom-foundation/Aida/logger"
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

// PrimeContext structure keeps context used over iterations of priming
type PrimeContext struct {
	cfg   *Config
	log   *logging.Logger
	block uint64
	exist map[common.Address]bool // account exists in db
}

func NewPrimeContext(cfg *Config, log *logging.Logger) *PrimeContext {
	return &PrimeContext{cfg: cfg, log: log, block: 0, exist: make(map[common.Address]bool)}
}

// PrimeStateDB primes database with accounts from the world state.
func (pc *PrimeContext) PrimeStateDB(ws substate.SubstateAlloc, db state.StateDB) error {
	numValues := 0 // number of storage values
	for _, account := range ws {
		numValues += len(account.Storage)
	}
	pc.log.Infof("\tLoading %d accounts with %d values ..", len(ws), numValues)

	pt := NewProgressTracker(numValues, pc.log)
	if pc.cfg.PrimeRandom {
		//if 0, commit once after priming all accounts
		if pc.cfg.PrimeThreshold == 0 {
			pc.cfg.PrimeThreshold = len(ws)
		}
		if err := pc.PrimeStateDBRandom(ws, db, pt); err != nil {
			return fmt.Errorf("failed to prime StateDB: %v", err)
		}
	} else {
		load := db.StartBulkLoad(pc.block)
		step := 0
		for addr, account := range ws {
			pc.primeOneAccount(addr, account, load, pt)
			step++
			// commit to stateDB after process n accounts
			if step%bulkLoadCap == 0 {
				if err := load.Close(); err != nil {
					return fmt.Errorf("failed to prime StateDB: %v", err)
				}
				pc.block++
				step = 0
				load = db.StartBulkLoad(pc.block)
			}
		}
		if err := load.Close(); err != nil {
			return fmt.Errorf("failed to prime StateDB: %v", err)
		}
		pc.block++
	}
	pc.log.Infof("\t\tPriming completed ...")
	return nil
}

// primeOneAccount initializes an account on stateDB with substate
func (pc *PrimeContext) primeOneAccount(addr common.Address, account *substate.SubstateAccount, load state.BulkLoad, pt *ProgressTracker) {
	// if an account was previously primed, skip account creation.
	if exist, found := pc.exist[addr]; !found || !exist {
		load.CreateAccount(addr)
		pc.exist[addr] = true
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
func (pc *PrimeContext) PrimeStateDBRandom(ws substate.SubstateAlloc, db state.StateDB, pt *ProgressTracker) error {
	contracts := make([]string, 0, len(ws))
	for addr := range ws {
		contracts = append(contracts, addr.Hex())
	}

	sort.Strings(contracts)
	// shuffle contract order
	rand.NewSource(pc.cfg.RandomSeed)
	rand.Shuffle(len(contracts), func(i, j int) {
		contracts[i], contracts[j] = contracts[j], contracts[i]
	})

	load := db.StartBulkLoad(pc.block)
	step := 0
	for _, c := range contracts {
		addr := common.HexToAddress(c)
		account := ws[addr]
		pc.primeOneAccount(addr, account, load, pt)
		step++
		// commit to stateDB after process n accounts and start a new buck load
		if step%bulkLoadCap == 0 {
			if err := load.Close(); err != nil {
				return err
			}
			pc.block++
			step = 0
			load = db.StartBulkLoad(pc.block)
		}

	}
	err := load.Close()
	pc.block++
	return err
}

// SuicideAccounts clears storage of all input accounts.
func (pc *PrimeContext) SuicideAccounts(db state.StateDB, accounts []common.Address) {
	pc.log.Info("Remove suicided accounts from stateDB.")
	count := 0
	db.BeginSyncPeriod(0)
	db.BeginBlock(pc.block)
	db.BeginTransaction(0)
	for _, addr := range accounts {
		if db.Exist(addr) {
			db.Suicide(addr)
			count++
			pc.exist[addr] = false
		}
	}
	db.EndTransaction()
	db.EndBlock()
	db.EndSyncPeriod()
	pc.block++
	pc.log.Infof("\t %v accounts were removed.", count)
}

// GenerateWorldStateAndPrime
func LoadWorldStateAndPrime(db state.StateDB, cfg *Config, target uint64) error {
	log := logger.NewLogger(cfg.LogLevel, "Priming")
	pc := NewPrimeContext(cfg, log)

	var (
		totalSize uint64
		maxSize   uint64 = cfg.UpdateBufferSize
		blockPos  uint64 = FirstSubstateBlock - 1
	)

	if target < blockPos {
		return fmt.Errorf("the target block, %v, is earlier than the initial world state block, %v. The world state is not loaded.", target, blockPos)
	}
	// load pre-computed update-set from update-set db
	udb, err := substate.OpenUpdateDBReadOnly(cfg.UpdateDb)
	if err != nil {
		return err
	}
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
			if err := pc.PrimeStateDB(update, db); err != nil {
				return fmt.Errorf("failed to prime StateDB: %v", err)
			}
			totalSize = 0
			update = make(substate.SubstateAlloc)
		}

		// Reset accessed storage locationas of suicided accounts prior to updateset block.
		// The known accessed storage locations in the updateset range has already been
		// reset when generating the update set database.
		ClearAccountStorage(update, newSet.DeletedAccounts)
		// if exists in DB, suicide
		// TODO may aggregate list and delete only once before priming
		pc.SuicideAccounts(db, newSet.DeletedAccounts)

		update.Merge(*newSet.UpdateSet)
		totalSize += incrementalSize
		log.Infof("\tMerge update set at block %v. New toal size %v MiB (+%v MiB)", newSet.Block, totalSize>>20, incrementalSize>>20)
	}
	// prime the remaining from updateset
	if err := pc.PrimeStateDB(update, db); err != nil {
		return fmt.Errorf("failed to prime StateDB: %v", err)
	}
	updateIter.Release()
	update = make(substate.SubstateAlloc)

	// advance from the latest precomputed state to the target block
	if blockPos < target {
		log.Infof("\tPriming from substate from block %v", blockPos)
		update, deletedAccounts, err := generateUpdateSet(blockPos+1, target, cfg)
		if err != nil {
			return err
		}
		pc.SuicideAccounts(db, deletedAccounts)
		if err := pc.PrimeStateDB(update, db); err != nil {
			return fmt.Errorf("failed to prime StateDB: %v", err)
		}
	}

	// delete destroyed accounts from stateDB
	log.Notice("Delete destroyed accounts")
	// remove destroyed accounts until one block before the first block
	err = DeleteDestroyedAccountsFromStateDB(db, cfg, cfg.First-1)

	return err
}
