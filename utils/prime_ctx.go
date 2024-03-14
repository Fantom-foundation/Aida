package utils

import (
	"fmt"
	"math/rand"
	"sort"

	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/state"
	"github.com/Fantom-foundation/Aida/txcontext"
	"github.com/ethereum/go-ethereum/common"
)

func NewPrimeContext(cfg *Config, db state.StateDB, block uint64, log logger.Logger) *PrimeContext {
	return &PrimeContext{cfg: cfg, log: log, block: block, db: db, exist: make(map[common.Address]bool)}
}

// PrimeContext structure keeps context used over iterations of priming
type PrimeContext struct {
	cfg        *Config
	log        logger.Logger
	block      uint64
	load       state.BulkLoad
	db         state.StateDB
	exist      map[common.Address]bool // account exists in db
	operations int                     // number of operations processed without commit
}

// mayApplyBulkLoad closes and reopen bulk load if it has over n operations.
func (pc *PrimeContext) mayApplyBulkLoad() error {
	if pc.operations >= OperationThreshold {
		pc.log.Debugf("\t\tApply bulk load with %v operations...", pc.operations)
		pc.operations = 0
		if err := pc.load.Close(); err != nil {
			return fmt.Errorf("failed to prime StateDB: %v", err)
		}
		pc.block++

		var err error
		pc.load, err = pc.db.StartBulkLoad(pc.block)
		if err != nil {
			return err
		}
	}
	return nil
}

// PrimeStateDB primes database with accounts from the world state.
func (pc *PrimeContext) PrimeStateDB(ws txcontext.WorldState, db state.StateDB) error {
	numValues := 0 // number of storage values
	ws.ForEachAccount(func(address common.Address, account txcontext.Account) {
		numValues += account.GetStorageSize()
	})

	pc.log.Debugf("\tLoading %d accounts with %d values ..", ws.Len(), numValues)

	pt := NewProgressTracker(numValues, pc.log)
	if pc.cfg.PrimeRandom {
		//if 0, commit once after priming all accounts
		if pc.cfg.PrimeThreshold == 0 {
			pc.cfg.PrimeThreshold = ws.Len()
		}
		if err := pc.PrimeStateDBRandom(ws, db, pt); err != nil {
			return fmt.Errorf("failed to prime StateDB: %v", err)
		}
	} else {
		var err error
		pc.load, err = db.StartBulkLoad(pc.block)
		if err != nil {
			return err
		}

		var forEachError error
		ws.ForEachAccount(func(addr common.Address, acc txcontext.Account) {
			if err := pc.primeOneAccount(addr, acc, pt); err != nil {
				forEachError = err
				return
			}
			// commit to stateDB after process n operations
			if err := pc.mayApplyBulkLoad(); err != nil {
				forEachError = err
				return
			}
		})

		if forEachError != nil {
			return forEachError
		}

		if err := pc.load.Close(); err != nil {
			return fmt.Errorf("failed to prime StateDB: %v", err)
		}
		pc.block++
	}
	pc.log.Debugf("\t\tPriming completed ...")
	return nil
}

// primeOneAccount initializes an account on stateDB with substate
func (pc *PrimeContext) primeOneAccount(addr common.Address, acc txcontext.Account, pt *ProgressTracker) error {
	exist, found := pc.exist[addr]
	// do not create empty accounts
	if !exist && acc.GetBalance().Sign() == 0 && acc.GetNonce() == 0 && len(acc.GetCode()) == 0 {
		return nil
	}
	// if an account was previously primed, skip account creation.
	if !found || !exist {
		pc.load.CreateAccount(addr)
		pc.exist[addr] = true
		pc.operations++
	}
	pc.load.SetBalance(addr, acc.GetBalance())
	pc.load.SetNonce(addr, acc.GetNonce())
	pc.load.SetCode(addr, acc.GetCode())
	pc.operations = pc.operations + 3

	var forEachError error
	acc.ForEachStorage(func(keyHash common.Hash, valueHash common.Hash) {
		pc.load.SetState(addr, keyHash, valueHash)
		pt.PrintProgress()
		pc.operations++
		if err := pc.mayApplyBulkLoad(); err != nil {
			forEachError = err
			return
		}
	})

	if forEachError != nil {
		return forEachError
	}

	return nil
}

// PrimeStateDBRandom primes database with accounts from the world state in random order.
func (pc *PrimeContext) PrimeStateDBRandom(ws txcontext.WorldState, db state.StateDB, pt *ProgressTracker) error {
	contracts := make([]string, 0, ws.Len())
	ws.ForEachAccount(func(addr common.Address, _ txcontext.Account) {
		contracts = append(contracts, addr.Hex())
	})

	sort.Strings(contracts)
	// shuffle contract order
	rand.NewSource(pc.cfg.RandomSeed)
	rand.Shuffle(len(contracts), func(i, j int) {
		contracts[i], contracts[j] = contracts[j], contracts[i]
	})

	var err error
	pc.load, err = pc.db.StartBulkLoad(pc.block)
	if err != nil {
		return err
	}
	for _, c := range contracts {
		addr := common.HexToAddress(c)
		account := ws.Get(addr)
		if err := pc.primeOneAccount(addr, account, pt); err != nil {
			return err
		}
		// commit to stateDB after process n accounts and start a new buck load
		if err := pc.mayApplyBulkLoad(); err != nil {
			return err
		}

	}
	err = pc.load.Close()
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
			pc.log.Debugf("\t\t Perform suicide on %v", addr)
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
