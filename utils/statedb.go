package utils

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/Fantom-foundation/Aida/state"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/substate"
)

// MakeStateDB creates a new DB instance based on cli argument.
func MakeStateDB(directory string, cfg *Config, rootHash common.Hash, isExistingDB bool) (state.StateDB, error) {
	db, err := makeStateDBInternal(directory, cfg, rootHash, isExistingDB)
	if err != nil {
		return nil, err
	}
	if cfg.DbLogging {
		db = state.MakeLoggingStateDB(db)
	}
	return db, nil
}

// makeStateDB creates a DB instance with a potential shadow instance.
func makeStateDBInternal(directory string, cfg *Config, rootHash common.Hash, isExistingDB bool) (state.StateDB, error) {
	if cfg.ShadowImpl == "" {
		return makeStateDBVariant(directory, cfg.DbImpl, cfg.DbVariant, rootHash, cfg.ArchiveMode)
	}
	if isExistingDB {
		return nil, fmt.Errorf("Using an existing stateDB with a shadow DB is not supported.")
	}
	primeDir := directory + "/prime"
	if err := os.MkdirAll(primeDir, 0700); err != nil {
		return nil, err
	}
	shadowDir := directory + "/shadow"
	if err := os.MkdirAll(shadowDir, 0700); err != nil {
		return nil, err
	}
	prime, err := makeStateDBVariant(primeDir, cfg.DbImpl, cfg.DbVariant, rootHash, cfg.ArchiveMode)
	if err != nil {
		return nil, err
	}
	shadow, err := makeStateDBVariant(shadowDir, cfg.ShadowImpl, cfg.ShadowVariant, rootHash, cfg.ArchiveMode)
	if err != nil {
		return nil, err
	}
	return state.MakeShadowStateDB(prime, shadow), nil
}

// makeStateDBVariant creates a DB instance of the requested kind.
func makeStateDBVariant(directory, impl, variant string, rootHash common.Hash, archiveMode bool) (state.StateDB, error) {
	switch impl {
	case "memory":
		return state.MakeEmptyGethInMemoryStateDB(variant)
	case "geth":
		return state.MakeGethStateDB(directory, variant, rootHash, archiveMode)
	case "carmen":
		return state.MakeCarmenStateDB(directory, variant, archiveMode)
	case "flat":
		return state.MakeFlatStateDB(directory, variant, rootHash)
	}
	return nil, fmt.Errorf("unknown DB implementation (--%v): %v", StateDbImplementationFlag.Name, impl)
}

// PrimeStateDB primes database with accounts from the world state.
func PrimeStateDB(ws substate.SubstateAlloc, db state.StateDB, cfg *Config) {
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

	if cfg.PrimeRandom {
		//if 0, commit once after priming all accounts
		if cfg.PrimeThreshold == 0 {
			cfg.PrimeThreshold = len(ws)
		}
		PrimeStateDBRandom(ws, load, cfg, progressTracker)
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

// PrimeStateDBRandom primes database with accounts from the world state in random order.
func PrimeStateDBRandom(ws substate.SubstateAlloc, db state.BulkLoad, cfg *Config, afterLoad func()) {
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
		primeOneAccount(addr, account, db, afterLoad)

	}
}

// DeleteDestroyedAccountsFromWorldState removes previously suicided accounts from
// the world state.
func DeleteDestroyedAccountsFromWorldState(ws substate.SubstateAlloc, cfg *Config, target uint64) error {
	if !cfg.HasDeletedAccounts {
		log.Printf("Database not provided. Ignore deleted accounts.\n")
		return nil
	}
	src := substate.OpenDestroyedAccountDBReadOnly(cfg.DeletedAccountDir)
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
	if !cfg.HasDeletedAccounts {
		log.Printf("Database not provided. Ignore deleted accounts.\n")
		return nil
	}
	src := substate.OpenDestroyedAccountDBReadOnly(cfg.DeletedAccountDir)
	defer src.Close()
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

// GetDirectorySize computes the size of all files in the given directoy in bytes.
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

// PrepareStateDB creates stateDB or load existing stateDB
func PrepareStateDB(cfg *Config) (db state.StateDB, workingDirectory string, loadedExistingDB bool, err error) {
	var exists bool
	roothash := common.Hash{}
	loadedExistingDB = false

	//create a temporary working directory
	workingDirectory, err = ioutil.TempDir(cfg.StateDbTempDir, "state_db_tmp_*")
	if err != nil {
		err = fmt.Errorf("Failed to create a temporary directory. %v", err)
		return
	}

	// check if statedb_info.json files exist
	dbInfoFile := filepath.Join(cfg.StateDbSrcDir, DbInfoName)
	if _, err = os.Stat(dbInfoFile); err == nil {
		exists = true
	} else if errors.Is(err, os.ErrNotExist) {
		exists = false
		if cfg.StateDbSrcDir != "" {
			log.Printf("WARNING: File %v does not exist. Create an empty StateDB.\n", dbInfoFile)
		}
	} else {
		return
	}

	if exists {
		dbinfo, ferr := ReadStateDbInfo(dbInfoFile)
		if ferr != nil {
			err = fmt.Errorf("Failed to read %v. %v", dbInfoFile, ferr)
			return
		}
		if dbinfo.Impl != cfg.DbImpl {
			err = fmt.Errorf("Mismatch DB implementation.\n\thave %v\n\twant %v", dbinfo.Impl, cfg.DbImpl)
		} else if dbinfo.Variant != cfg.DbVariant {
			err = fmt.Errorf("Mismatch DB variant.\n\thave %v\n\twant %v", dbinfo.Variant, cfg.DbVariant)
		} else if dbinfo.Block+1 != cfg.First {
			err = fmt.Errorf("The first block is earlier than stateDB.\n\thave %v\n\twant %v", dbinfo.Block+1, cfg.First)
		} else if dbinfo.ArchiveMode != cfg.ArchiveMode {
			err = fmt.Errorf("Mismatch archive mode.\n\thave %v\n\twant %v", dbinfo.ArchiveMode, cfg.ArchiveMode)
		}
		if err != nil {
			return
		}

		// make a copy of stateDB directory
		copyDir(cfg.StateDbSrcDir, workingDirectory)
		loadedExistingDB = true

		// if this is an existing statedb, open
		roothash = dbinfo.RootHash
	}

	log.Printf("\tTemporary state DB directory: %v\n", workingDirectory)
	db, err = MakeStateDB(workingDirectory, cfg, roothash, loadedExistingDB)

	return
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
