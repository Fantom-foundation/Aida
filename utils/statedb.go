package utils

import (
	"bytes"
	"fmt"
	"io/fs"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/Fantom-foundation/Aida/state"
	substate "github.com/Fantom-foundation/Substate"
	"github.com/ethereum/go-ethereum/common"
	"github.com/google/martian/log"
	"github.com/op/go-logging"
)

const (
	pathToPrimeDb  = "/prime"
	pathToShadowDb = "/shadow"
)

// PrepareStateDB creates stateDB or load existing stateDB
// Use this function when both opening existing and creating new StateDB
func PrepareStateDB(cfg *Config) (state.StateDB, string, error) {
	var (
		db     state.StateDB
		err    error
		dbPath string
	)

	// db source was specified
	if cfg.StateDbSrc != "" {
		db, dbPath, err = useExistingStateDB(cfg)
	} else {
		db, dbPath, err = makeNewStateDB(cfg)
	}

	if err != nil {
		return nil, "", err
	}

	if cfg.DbLogging {
		db = state.MakeLoggingStateDB(db)
	}

	return db, dbPath, nil
}

// useExistingStateDB uses already existing DB to create a DB instance with a potential shadow instance.
func useExistingStateDB(cfg *Config) (state.StateDB, string, error) {
	var (
		err         error
		primeDb     state.StateDB
		primeDbInfo StateDbInfo
		primeDbPath string
	)

	// no shadow db
	if cfg.ShadowDb {
		primeDbPath = filepath.Join(cfg.StateDbSrc, pathToPrimeDb)
	} else {
		primeDbPath = cfg.StateDbSrc
	}

	primeDbInfoFile := filepath.Join(primeDbPath, PathToDbInfo)
	primeDbInfo, err = ReadStateDbInfo(primeDbInfoFile)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read %v. %v", primeDbInfoFile, err)
	}

	// open primary db
	primeDb, err = makeStateDBVariant(primeDbPath, primeDbInfo.Impl, primeDbInfo.Variant, primeDbInfo.ArchiveVariant, primeDbInfo.RootHash, cfg)
	if err != nil {
		return nil, "", err
	}

	if cfg.ShadowDb {
		return primeDb, primeDbPath, nil
	}

	var (
		shadowDb     state.StateDB
		shadowDbInfo StateDbInfo
		shadowDbPath string
	)

	shadowDbPath = filepath.Join(cfg.StateDbSrc, pathToShadowDb)
	shadowDbInfoFile := filepath.Join(shadowDbPath, PathToDbInfo)
	shadowDbInfo, err = ReadStateDbInfo(shadowDbInfoFile)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read %v. %v", shadowDbInfoFile, err)
	}

	// open shadow db
	shadowDb, err = makeStateDBVariant(shadowDbPath, shadowDbInfo.Impl, shadowDbInfo.Variant, shadowDbInfo.ArchiveVariant, shadowDbInfo.RootHash, cfg)
	if err != nil {
		return nil, "", err
	}

	return state.MakeShadowStateDB(primeDb, shadowDb), cfg.StateDbSrc, nil
}

// makeNewStateDB creates a DB instance with a potential shadow instance.
func makeNewStateDB(cfg *Config) (state.StateDB, string, error) {
	var (
		err           error
		primaryDb     state.StateDB
		primaryDbPath string
		tmpDir        string
	)

	// create a temporary working directory
	tmpDir, err = os.MkdirTemp(cfg.DbTmp, "state_db_tmp_*")
	if err != nil {
		err = fmt.Errorf("failed to create a temporary directory. %v", err)
		return nil, "", err
	}

	log.Infof("Temporary state DB directory: %v", tmpDir)

	primaryDbPath = tmpDir

	// no shadow db
	if cfg.ShadowDb {
		primaryDbPath = filepath.Join(primaryDbPath, pathToPrimeDb)
	}

	// create primary db
	primaryDb, err = makeStateDBVariant(primaryDbPath, cfg.DbImpl, cfg.DbVariant, cfg.ArchiveVariant, common.Hash{}, cfg)
	if err != nil {
		return nil, "", err
	}

	if !cfg.ShadowDb {
		return primaryDb, primaryDbPath, nil
	}

	var (
		shadowDb     state.StateDB
		shadowDbPath string
	)

	shadowDbPath = filepath.Join(cfg.StateDbSrc, pathToShadowDb)

	// open shadow db
	shadowDb, err = makeStateDBVariant(shadowDbPath, cfg.ShadowImpl, cfg.ShadowVariant, cfg.ArchiveVariant, common.Hash{}, cfg)
	if err != nil {
		return nil, "", err
	}

	return state.MakeShadowStateDB(primaryDb, shadowDb), tmpDir, nil
}

// makeStateDBVariant creates a DB instance of the requested kind.
func makeStateDBVariant(directory, impl, variant, archiveVariant string, rootHash common.Hash, cfg *Config) (state.StateDB, error) {
	switch impl {
	case "memory":
		return state.MakeEmptyGethInMemoryStateDB(variant)
	case "geth":
		return state.MakeGethStateDB(directory, variant, rootHash, cfg.ArchiveMode)
	case "carmen":
		// Disable archive if not enabled.
		if !cfg.ArchiveMode {
			archiveVariant = "none"
		}
		return state.MakeCarmenStateDB(directory, variant, archiveVariant, cfg.CarmenSchema)
	case "flat":
		return state.MakeFlatStateDB(directory, variant, rootHash)
	}
	return nil, fmt.Errorf("unknown DB implementation (--%v): %v", StateDbImplementationFlag.Name, impl)
}

type ProgressTracker struct {
	step   int       // step counter
	target int       // total number of steps
	start  time.Time // start time
	last   time.Time // last reported time
	rate   float64   // priming rate
	log    *logging.Logger
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
	const printFrequency = 100000 // report after x steps
	pt.step++
	if pt.step%printFrequency == 0 {
		now := time.Now()
		currentRate := printFrequency / now.Sub(pt.last).Seconds()
		pt.rate = currentRate*0.1 + pt.rate*0.9
		pt.last = now
		progress := float32(pt.step) / float32(pt.target)
		time := int(now.Sub(pt.start).Seconds())
		eta := int(float64(pt.target-pt.step) / pt.rate)
		pt.log.Infof("Loading state ... %8.1f slots/s, %5.1f%%, time: %d:%02d, ETA: %d:%02d", currentRate, progress*100, time/60, time%60, eta/60, eta%60)
	}
}

// PrimeStateDB primes database with accounts from the world state.
func PrimeStateDB(ws substate.SubstateAlloc, db state.StateDB, cfg *Config, log *logging.Logger) {
	load := db.StartBulkLoad()

	numValues := 0
	for _, account := range ws {
		numValues += len(account.Storage)
	}
	log.Infof("Loading %d accounts with %d values ...", len(ws), numValues)

	pt := NewProgressTracker(numValues, log)
	if cfg.PrimeRandom {
		//if 0, commit once after priming all accounts
		if cfg.PrimeThreshold == 0 {
			cfg.PrimeThreshold = len(ws)
		}
		PrimeStateDBRandom(ws, load, cfg, pt)
	} else {
		for addr, account := range ws {
			primeOneAccount(addr, account, load, pt)
		}

	}
	log.Noticef("Hashing and flushing ...")
	if err := load.Close(); err != nil {
		panic(fmt.Errorf("failed to prime StateDB: %v", err))
	}
}

// primeOneAccount initializes an account on stateDB with substate
func primeOneAccount(addr common.Address, account *substate.SubstateAccount, db state.BulkLoad, pt *ProgressTracker) {
	db.CreateAccount(addr)
	db.SetBalance(addr, account.Balance)
	db.SetNonce(addr, account.Nonce)
	db.SetCode(addr, account.Code)
	for key, value := range account.Storage {
		db.SetState(addr, key, value)
		pt.PrintProgress()
	}
}

// PrimeStateDBRandom primes database with accounts from the world state in random order.
func PrimeStateDBRandom(ws substate.SubstateAlloc, db state.BulkLoad, cfg *Config, pt *ProgressTracker) {
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
		primeOneAccount(addr, account, db, pt)

	}
}

// DeleteDestroyedAccountsFromWorldState removes previously suicided accounts from
// the world state.
func DeleteDestroyedAccountsFromWorldState(ws substate.SubstateAlloc, cfg *Config, target uint64) error {
	log := NewLogger(cfg.LogLevel, "DelDestAcc")

	if !cfg.HasDeletedAccounts {
		log.Warning("Database not provided. Ignore deleted accounts")
		return nil
	}
	src := substate.OpenDestroyedAccountDBReadOnly(cfg.DeletionDb)
	defer src.Close()
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

// DeleteDestroyedAccountsFromStateDB performs suicide operations on previously
// self-destructed accounts.
func DeleteDestroyedAccountsFromStateDB(db state.StateDB, cfg *Config, target uint64) error {
	log := NewLogger(cfg.LogLevel, "DelDestAcc")

	if !cfg.HasDeletedAccounts {
		log.Warning("Database not provided. Ignore deleted accounts.")
		return nil
	}
	src := substate.OpenDestroyedAccountDBReadOnly(cfg.DeletionDb)
	defer src.Close()
	list, err := src.GetAccountsDestroyedInRange(0, target)
	if err != nil {
		return err
	}
	log.Noticef("Deleting %d accounts ...", len(list))
	db.BeginSyncPeriod(0)
	db.BeginBlock(target) // block 0 is the priming, block (first-1) the deletion
	db.BeginTransaction(0)
	for _, cur := range list {
		db.Suicide(cur)
	}
	db.Finalise(true)
	db.EndTransaction()
	db.EndBlock()
	db.EndSyncPeriod()
	return nil
}

// GetDirectorySize computes the size of all files in the given directory in bytes.
func GetDirectorySize(directory string) int64 {
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

// ValidateStateDB validates whether the world-state is contained in the db object.
// NB: We can only check what must be in the db (but cannot check whether db stores more).
func ValidateStateDB(ws substate.SubstateAlloc, db state.StateDB, updateOnFail bool) error {
	var err string
	for addr, account := range ws {
		if !db.Exist(addr) {
			err += fmt.Sprintf("  Account %v does not exist\n", addr.Hex())
			if updateOnFail {
				db.CreateAccount(addr)
			}
		}
		if balance := db.GetBalance(addr); account.Balance.Cmp(balance) != 0 {
			err += fmt.Sprintf("  Failed to validate balance for account %v\n"+
				"    have %v\n"+
				"    want %v\n",
				addr.Hex(), balance, account.Balance)
			if updateOnFail {
				db.SubBalance(addr, balance)
				db.AddBalance(addr, account.Balance)
			}
		}
		if nonce := db.GetNonce(addr); nonce != account.Nonce {
			err += fmt.Sprintf("  Failed to validate nonce for account %v\n"+
				"    have %v\n"+
				"    want %v\n",
				addr.Hex(), nonce, account.Nonce)
			if updateOnFail {
				db.SetNonce(addr, account.Nonce)
			}
		}
		if code := db.GetCode(addr); bytes.Compare(code, account.Code) != 0 {
			err += fmt.Sprintf("  Failed to validate code for account %v\n"+
				"    have len %v\n"+
				"    want len %v\n",
				addr.Hex(), len(code), len(account.Code))
			if updateOnFail {
				db.SetCode(addr, account.Code)
			}
		}
		for key, value := range account.Storage {
			if db.GetState(addr, key) != value {
				err += fmt.Sprintf("  Failed to validate storage for account %v, key %v\n"+
					"    have %v\n"+
					"    want %v\n",
					addr.Hex(), key.Hex(), db.GetState(addr, key).Hex(), value.Hex())
				if updateOnFail {
					db.SetState(addr, key, value)
				}
			}
		}
	}
	if len(err) > 0 {
		return fmt.Errorf(err)
	}
	return nil
}
