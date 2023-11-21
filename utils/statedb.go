package utils

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/state"
	"github.com/Fantom-foundation/Aida/state/proxy"
	substate "github.com/Fantom-foundation/Substate"
	"github.com/ethereum/go-ethereum/common"
	"github.com/google/martian/log"
)

const (
	PathToPrimaryStateDb = "/prime"
	PathToShadowStateDb  = "/shadow"
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
		cfg.IsExistingStateDb = true
	} else {
		db, dbPath, err = makeNewStateDB(cfg)
	}

	if err != nil {
		return nil, "", err
	}

	if cfg.DbLogging {
		db = proxy.NewLoggerProxy(db, cfg.LogLevel)
	}
	return db, dbPath, nil
}

// useExistingStateDB uses already existing DB to create a DB instance with a potential shadow instance.
func useExistingStateDB(cfg *Config) (state.StateDB, string, error) {
	var (
		err            error
		stateDb        state.StateDB
		stateDbInfo    StateDbInfo
		tmpStateDbPath string
		log            = logger.NewLogger(cfg.LogLevel, "StateDB-Creation")
	)

	// make a copy of source statedb
	if !cfg.SrcDbReadonly {
		// does path to state db exist?
		if _, err = os.Stat(cfg.StateDbSrc); os.IsNotExist(err) {
			return nil, "", fmt.Errorf("%v does not exist", cfg.StateDbSrc)
		}

		tmpStateDbPath, err = os.MkdirTemp(cfg.DbTmp, "state_db_tmp_*")
		if err != nil {
			return nil, "", fmt.Errorf("failed to create a temporary directory; %v", err)
		}

		size, err := FindDirSize(cfg.StateDbSrc)
		if err != nil {
			return nil, "", err
		}

		log.Infof("Copying your StateDb. Size: %.2f MB", float64(size)/float64(1000000))
		if err = CopyDir(cfg.StateDbSrc, tmpStateDbPath); err != nil {
			return nil, "", fmt.Errorf("failed to copy source statedb to temporary directory; %v", err)
		}
		cfg.PathToStateDb = tmpStateDbPath
	} else {
		// when not using ShadowDb, StateDbSrc is path to the StateDb itself
		cfg.PathToStateDb = cfg.StateDbSrc
	}

	// using ShadowDb?
	if cfg.ShadowDb {
		cfg.PathToStateDb = filepath.Join(cfg.PathToStateDb, PathToPrimaryStateDb)
	}

	stateDbInfoFile := filepath.Join(cfg.PathToStateDb, PathToDbInfo)
	stateDbInfo, err = ReadStateDbInfo(stateDbInfoFile)
	if err != nil {
		return nil, "", fmt.Errorf("cannot read StateDb cfg file '%v'; %v", stateDbInfoFile, err)
	}

	// do we have an archive inside loaded StateDb?
	cfg.ArchiveMode = stateDbInfo.ArchiveMode

	// open primary db
	stateDb, err = makeStateDBVariant(cfg.PathToStateDb, stateDbInfo.Impl, stateDbInfo.Variant, stateDbInfo.ArchiveVariant, stateDbInfo.Schema, stateDbInfo.RootHash, cfg)
	if err != nil {
		return nil, "", fmt.Errorf("cannot create StateDb; %v", err)
	}

	if !cfg.ShadowDb {
		return stateDb, cfg.PathToStateDb, nil
	}

	var (
		shadowDb     state.StateDB
		shadowDbInfo StateDbInfo
		shadowDbPath string
	)

	shadowDbPath = filepath.Join(cfg.StateDbSrc, PathToShadowStateDb)
	shadowDbInfoFile := filepath.Join(shadowDbPath, PathToDbInfo)
	shadowDbInfo, err = ReadStateDbInfo(shadowDbInfoFile)
	if err != nil {
		return nil, "", fmt.Errorf("cannot read ShadowDb cfg file '%v'; %v", shadowDbInfoFile, err)
	}

	// open shadow db
	shadowDb, err = makeStateDBVariant(shadowDbPath, shadowDbInfo.Impl, shadowDbInfo.Variant, shadowDbInfo.ArchiveVariant, shadowDbInfo.Schema, shadowDbInfo.RootHash, cfg)
	if err != nil {
		return nil, "", fmt.Errorf("cannot create ShadowDb; %v", err)
	}

	return proxy.NewShadowProxy(stateDb, shadowDb), cfg.StateDbSrc, nil
}

// makeNewStateDB creates a DB instance with a potential shadow instance.
func makeNewStateDB(cfg *Config) (state.StateDB, string, error) {
	var (
		err         error
		stateDb     state.StateDB
		stateDbPath string
		tmpDir      string
	)

	// create a temporary working directory
	tmpDir, err = os.MkdirTemp(cfg.DbTmp, "state_db_tmp_*")
	if err != nil {
		return nil, "", fmt.Errorf("failed to create a temporary directory; %v", err)
	}

	log.Infof("Temporary StateDb directory: %v", tmpDir)

	stateDbPath = tmpDir

	// no shadow db
	if cfg.ShadowDb {
		stateDbPath = filepath.Join(stateDbPath, PathToPrimaryStateDb)
	}

	// create primary db
	stateDb, err = makeStateDBVariant(stateDbPath, cfg.DbImpl, cfg.DbVariant, cfg.ArchiveVariant, cfg.CarmenSchema, common.Hash{}, cfg)
	if err != nil {
		return nil, "", fmt.Errorf("cannnot make stateDb; %v", err)
	}

	if !cfg.ShadowDb {
		return stateDb, stateDbPath, nil
	}

	var (
		shadowDb     state.StateDB
		shadowDbPath string
	)

	shadowDbPath = filepath.Join(tmpDir, PathToShadowStateDb)

	// open shadow db
	shadowDb, err = makeStateDBVariant(shadowDbPath, cfg.ShadowImpl, cfg.ShadowVariant, cfg.ArchiveVariant, cfg.CarmenSchema, common.Hash{}, cfg)
	if err != nil {
		return nil, "", fmt.Errorf("cannnot make shadowDb; %v", err)
	}

	return proxy.NewShadowProxy(stateDb, shadowDb), tmpDir, nil
}

// makeStateDBVariant creates a DB instance of the requested kind.
func makeStateDBVariant(directory, impl, variant, archiveVariant string, carmenSchema int, rootHash common.Hash, cfg *Config) (state.StateDB, error) {
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
		return state.MakeCarmenStateDB(directory, variant, archiveVariant, carmenSchema)
	case "opera":
		return state.MakeOperaStateDB(directory, variant, cfg.LogLevel)
	}
	return nil, fmt.Errorf("unknown Db implementation: %v", impl)
}

// DeleteDestroyedAccountsFromWorldState removes previously suicided accounts from
// the world state.
func DeleteDestroyedAccountsFromWorldState(ws substate.SubstateAlloc, cfg *Config, target uint64) error {
	log := logger.NewLogger(cfg.LogLevel, "DelDestAcc")

	if !cfg.HasDeletedAccounts {
		log.Warning("Database not provided. Ignore deleted accounts")
		return nil
	}
	src, err := substate.OpenDestroyedAccountDBReadOnly(cfg.DeletionDb)
	if err != nil {
		return err
	}
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
	log := logger.NewLogger(cfg.LogLevel, "DelDestAcc")

	if !cfg.HasDeletedAccounts {
		log.Warning("Database not provided. Ignore deleted accounts.")
		return nil
	}
	src, err := substate.OpenDestroyedAccountDBReadOnly(cfg.DeletionDb)
	if err != nil {
		return err
	}
	defer src.Close()
	accounts, err := src.GetAccountsDestroyedInRange(0, target)
	if err != nil {
		return err
	}
	log.Noticef("Deleting %d accounts ...", len(accounts))
	if len(accounts) == 0 {
		// nothing to delete, skip
		return nil
	}
	db.BeginSyncPeriod(0)
	db.BeginBlock(target)
	db.BeginTransaction(0)
	for _, addr := range accounts {
		db.Suicide(addr)
		log.Debugf("Perform suicide on %v", addr)
	}
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
func ValidateStateDB(ws substate.SubstateAlloc, db state.VmStateDB, updateOnFail bool) error {
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

// FindDirSize iterates over all files inside given directory (including subdirectories) and returns size in bytes.
func FindDirSize(path string) (int64, error) {
	var size int64
	err := filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return err
	})
	return size, err
}
