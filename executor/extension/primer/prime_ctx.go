package primer

import (
	"fmt"
	"math/rand"
	"sort"

	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/state"
	"github.com/Fantom-foundation/Aida/utils"
	substate "github.com/Fantom-foundation/Substate"
	"github.com/ethereum/go-ethereum/common"
)

func newPrimeContext(cfg *utils.Config, db state.StateDB, log logger.Logger) *primeContext {
	return &primeContext{cfg: cfg, log: log, block: 0, db: db, exist: make(map[common.Address]bool)}
}

// primeContext structure keeps context used over iterations of priming
type primeContext struct {
	cfg        *utils.Config
	log        logger.Logger
	block      uint64
	load       state.BulkLoad
	db         state.StateDB
	exist      map[common.Address]bool // account exists in db
	operations int                     // number of operations processed without commit
}

// mayApplyBulkLoad closes and reopen bulk load if it has over n operations.
func (pc *primeContext) mayApplyBulkLoad() error {
	if pc.operations >= utils.OperationThreshold {
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
func (pc *primeContext) PrimeStateDB(ws substate.SubstateAlloc, db state.StateDB) error {
	numValues := 0 // number of storage values
	for _, account := range ws {
		numValues += len(account.Storage)
	}
	pc.log.Infof("\tLoading %d accounts with %d values ..", len(ws), numValues)

	pt := utils.NewProgressTracker(numValues, pc.log)
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
func (pc *primeContext) primeOneAccount(addr common.Address, account *substate.SubstateAccount, pt *utils.ProgressTracker) error {
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
func (pc *primeContext) PrimeStateDBRandom(ws substate.SubstateAlloc, db state.StateDB, pt *utils.ProgressTracker) error {
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
func (pc *primeContext) SuicideAccounts(db state.StateDB, accounts []common.Address) {
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
