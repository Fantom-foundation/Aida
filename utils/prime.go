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

// threshold for wrapping a bulk load and reporting a priming progress
const operationThreshold = 1_000_000

// PrintProgress reports a priming rates and estimated time after n operations has been executed.
func (pt *ProgressTracker) PrintProgress() {
	pt.step++
	if pt.step%operationThreshold == 0 {
		now := time.Now()
		currentRate := operationThreshold / now.Sub(pt.last).Seconds()
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
	cfg        *Config
	log        *logging.Logger
	block      uint64
	load       state.BulkLoad
	db         state.StateDB
	exist      map[common.Address]bool // account exists in db
	operations int                     // number of operations processed without commit
}

func NewPrimeContext(cfg *Config, db state.StateDB, log *logging.Logger) *PrimeContext {
	return &PrimeContext{cfg: cfg, log: log, block: 0, db: db, exist: make(map[common.Address]bool)}
}

// mayApplyBulkLoad closes and reopen bulk load if it has over n operations.
func (pc *PrimeContext) mayApplyBulkLoad() error {

	if pc.operations >= operationThreshold {
		pc.log.Debugf("\t\tApply bulk load with %v operations...", pc.operations)
		pc.operations = 0
		if err := pc.load.Close(); err != nil {
			return fmt.Errorf("failed to prime StateDB: %v", err)
		}
		pc.block++
		pc.load = pc.db.StartBulkLoad(pc.block)
	}
	return nil
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
		pc.load = db.StartBulkLoad(pc.block)
		for addr, account := range ws {
			if err := pc.primeOneAccount(addr, account, pt); err != nil {
				return err
			}
			// commit to stateDB after process n operations
			if err := pc.mayApplyBulkLoad(); err != nil {
				return err
			}
		}
		if err := pc.load.Close(); err != nil {
			return fmt.Errorf("failed to prime StateDB: %v", err)
		}
		pc.block++
	}
	pc.log.Infof("\t\tPriming completed ...")
	return nil
}

// primeOneAccount initializes an account on stateDB with substate
func (pc *PrimeContext) primeOneAccount(addr common.Address, account *substate.SubstateAccount, pt *ProgressTracker) error {
	// if an account was previously primed, skip account creation.
	if exist, found := pc.exist[addr]; !found || !exist {
		pc.load.CreateAccount(addr)
		pc.exist[addr] = true
		pc.operations++
	}
	pc.load.SetBalance(addr, account.Balance)
	pc.load.SetNonce(addr, account.Nonce)
	pc.load.SetCode(addr, account.Code)
	pc.operations = pc.operations + 3
	for key, value := range account.Storage {
		pc.load.SetState(addr, key, value)
		pt.PrintProgress()
		pc.operations++
		if err := pc.mayApplyBulkLoad(); err != nil {
			return err
		}
	}
	return nil
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

	pc.load = db.StartBulkLoad(pc.block)
	for _, c := range contracts {
		addr := common.HexToAddress(c)
		account := ws[addr]
		if err := pc.primeOneAccount(addr, account, pt); err != nil {
			return err
		}
		// commit to stateDB after process n accounts and start a new buck load
		if err := pc.mayApplyBulkLoad(); err != nil {
			return err
		}

	}
	err := pc.load.Close()
	pc.block++
	return err
}

// SuicideAccounts clears storage of all input accounts.
func (pc *PrimeContext) SuicideAccounts(db state.StateDB, accounts []common.Address) {
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
	pc.log.Infof("\t\t %v suicided accounts were removed from statedb (before priming).", count)
}

// GenerateWorldStateAndPrime
func LoadWorldStateAndPrime(db state.StateDB, cfg *Config, target uint64) error {
	log := logger.NewLogger(cfg.LogLevel, "Priming")
	pc := NewPrimeContext(cfg, db, log)

	var (
		totalSize uint64 // total size of unprimed update set
		maxSize   uint64 // maximum size of update set before priming
		block     uint64 // current block position
		hasPrimed bool   // if true, db has been primed
	)

	maxSize = cfg.UpdateBufferSize
	// load pre-computed update-set from update-set db
	udb, err := substate.OpenUpdateDBReadOnly(cfg.UpdateDb)
	if err != nil {
		return err
	}
	defer udb.Close()
	updateIter := substate.NewUpdateSetIterator(udb, block, target)
	update := make(substate.SubstateAlloc)

	for updateIter.Next() {
		newSet := updateIter.Value()
		if newSet.Block > target {
			break
		}
		block = newSet.Block

		incrementalSize := update.EstimateIncrementalSize(*newSet.UpdateSet)
		// Prime StateDB
		if totalSize+incrementalSize > maxSize {
			log.Infof("\tPriming...")
			if err := pc.PrimeStateDB(update, db); err != nil {
				return err
			}
			totalSize = 0
			update = make(substate.SubstateAlloc)
			hasPrimed = true
		}

		// Reset accessed storage locationas of suicided accounts prior to updateset block.
		// The known accessed storage locations in the updateset range has already been
		// reset when generating the update set database.
		ClearAccountStorage(update, newSet.DeletedAccounts)
		// if exists in DB, suicide
		if hasPrimed {
			pc.SuicideAccounts(db, newSet.DeletedAccounts)
		}

		update.Merge(*newSet.UpdateSet)
		totalSize += incrementalSize
		log.Infof("\tMerge update set at block %v. New toal size %v MB (+%v MB)",
			newSet.Block, totalSize/1_000_000,
			incrementalSize/1_000_000)
	}
	// if update set is not empty, prime the remaining
	if len(update) > 0 {
		if err := pc.PrimeStateDB(update, db); err != nil {
			return err
		}
		update = make(substate.SubstateAlloc)
		hasPrimed = true
	}
	updateIter.Release()

	// advance from the latest precomputed state to the target block
	if block < target || target == 0 {
		log.Infof("\tPriming from substate from block %v", block)
		update, deletedAccounts, err := GenerateUpdateSet(block, target, cfg)
		if err != nil {
			return err
		}
		if hasPrimed {
			pc.SuicideAccounts(db, deletedAccounts)
		}
		if err := pc.PrimeStateDB(update, db); err != nil {
			return err
		}
	}

	// delete destroyed accounts from stateDB
	log.Notice("Delete destroyed accounts")
	// remove destroyed accounts until one block before the first block
	err = DeleteDestroyedAccountsFromStateDB(db, cfg, pc.block)
	return err
}
