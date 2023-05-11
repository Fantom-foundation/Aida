package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/big"
	"math/rand"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Fantom-foundation/Aida/state"
	substate "github.com/Fantom-foundation/Substate"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
)

const testAccountStorageSize = 10

type statedbTestCase struct {
	variant        string
	shadowImpl     string
	archiveMode    bool
	archiveVariant string
	primeRandom    bool
}

func getStatedbTestCases() []statedbTestCase {
	testCases := []statedbTestCase{
		{"geth", "", true, "", false},
		{"geth", "geth", true, "", false},
		{"carmen", "geth", false, "none", false},
		{"carmen", "geth", true, "ldb", false},
		{"carmen", "geth", true, "sqlite", false},
		{"flat", "geth", true, "sqlite", false},
		{"flat", "geth", true, "sqlite", true},
	}

	return testCases
}

// getInvalidStateDbInfo returns test cases with invalid StateDB info file data
// each individual changed value must collide with first statedbTestCase generated by getStatedbTestCases
func getInvalidStateDbInfo(cfg *Config) []StateDbInfo {
	tc := []StateDbInfo{
		{
			Impl:           "carmen",
			Variant:        cfg.DbVariant,
			ArchiveMode:    cfg.ArchiveMode,
			ArchiveVariant: cfg.ArchiveVariant,
			Schema:         0,
			Block:          cfg.First - 1,
		},
		{
			Impl:           cfg.DbImpl,
			Variant:        "cpp-file",
			ArchiveMode:    cfg.ArchiveMode,
			ArchiveVariant: cfg.ArchiveVariant,
			Schema:         0,
			Block:          cfg.First - 1,
		},
		{
			Impl:           cfg.DbImpl,
			Variant:        cfg.DbVariant,
			ArchiveMode:    false,
			ArchiveVariant: cfg.ArchiveVariant,
			Schema:         0,
			Block:          cfg.First - 1,
		},
		{
			Impl:           cfg.DbImpl,
			Variant:        cfg.DbVariant,
			ArchiveMode:    cfg.ArchiveMode,
			ArchiveVariant: "none",
			Schema:         0,
			Block:          cfg.First - 1,
		},
		{
			Impl:           cfg.DbImpl,
			Variant:        cfg.DbVariant,
			ArchiveMode:    cfg.ArchiveMode,
			ArchiveVariant: cfg.ArchiveVariant,
			Schema:         1,
			Block:          cfg.First - 1,
		},
		{
			Impl:           cfg.DbImpl,
			Variant:        cfg.DbVariant,
			ArchiveMode:    cfg.ArchiveMode,
			ArchiveVariant: cfg.ArchiveVariant,
			Schema:         0,
			Block:          cfg.First + 1,
		},
	}

	return tc
}

// makeRandomByteSlice creates byte slice of given length with randomized values
func makeRandomByteSlice(t *testing.T, bufferLength int) []byte {
	// make byte slice
	buffer := make([]byte, bufferLength)

	// fill the slice with random data
	_, err := rand.Read(buffer)
	if err != nil {
		t.Fatalf("failed test data; can not generate random byte slice; %s", err.Error())
	}

	return buffer
}

// getRandom generates random number in from given range
func getRandom(rangeLower int, rangeUpper int) int {
	// seed the PRNG
	rand.Seed(time.Now().UnixNano())

	// get randomized balance
	randInt := rangeLower + rand.Intn(rangeUpper-rangeLower+1)
	return randInt
}

// makeAccountStorage generates randomized account storage with testAccountStorageSize length
func makeAccountStorage(t *testing.T) map[common.Hash]common.Hash {
	// create storage map
	storage := map[common.Hash]common.Hash{}

	// fill the storage map
	for j := 0; j < testAccountStorageSize; j++ {
		k := common.BytesToHash(makeRandomByteSlice(t, 32))
		storage[k] = common.BytesToHash(makeRandomByteSlice(t, 32))
	}

	return storage
}

// makeTestConfig creates a config struct for testing
func makeTestConfig(testCase statedbTestCase) *Config {
	cfg := &Config{
		DbLogging:      true,
		DbImpl:         testCase.variant,
		DbVariant:      "",
		ShadowImpl:     testCase.shadowImpl,
		ShadowVariant:  "",
		ArchiveVariant: testCase.archiveVariant,
		ArchiveMode:    testCase.archiveMode,
		PrimeRandom:    testCase.primeRandom,
	}

	if testCase.variant == "flat" {
		cfg.DbVariant = "go-memory"
	}

	if testCase.primeRandom {
		cfg.PrimeThreshold = 0
		cfg.RandomSeed = int64(getRandom(1_000_000, 100_000_000))
	}

	return cfg
}

// makeWorldState generates randomized world state containing 100 accounts
func makeWorldState(t *testing.T) (substate.SubstateAlloc, []common.Address) {
	// create list of addresses
	var addrList []common.Address

	// create world state
	ws := substate.SubstateAlloc{}

	for i := 0; i < 100; i++ {
		// create random address
		addr := common.BytesToAddress(makeRandomByteSlice(t, 40))

		// add to address list
		addrList = append(addrList, addr)

		// create account
		ws[addr] = &substate.SubstateAccount{
			Nonce:   uint64(getRandom(1, 1000*5000)),
			Balance: big.NewInt(int64(getRandom(1, 1000*5000))),
			Storage: makeAccountStorage(t),
			Code:    makeRandomByteSlice(t, 2048),
		}
	}

	return ws, addrList
}

// TestStatedb_InitCloseStateDB test closing db immediately after initialization
func TestStatedb_InitCloseStateDB(t *testing.T) {
	for _, tc := range getStatedbTestCases() {
		t.Run(fmt.Sprintf("DB variant: %s; shadowImpl: %s; archive variant: %s", tc.variant, tc.shadowImpl, tc.archiveVariant), func(t *testing.T) {
			cfg := makeTestConfig(tc)

			// Initialization of state DB
			sDB, err := MakeStateDB(t.TempDir(), cfg, common.Hash{}, false)

			if err != nil {
				t.Fatalf("failed to create state DB: %v", err)
			}

			// Closing of state DB
			err = sDB.Close()
			if err != nil {
				t.Fatalf("failed to close state DB: %v", err)
			}
		})
	}
}

// TestStatedb_PrimeStateDB tests priming fresh state DB with randomized world state data
func TestStatedb_PrimeStateDB(t *testing.T) {
	for _, tc := range getStatedbTestCases() {
		t.Run(fmt.Sprintf("DB variant: %s; shadowImpl: %s; archive variant: %s", tc.variant, tc.shadowImpl, tc.archiveVariant), func(t *testing.T) {
			cfg := makeTestConfig(tc)

			// Initialization of state DB
			sDB, err := MakeStateDB(t.TempDir(), cfg, common.Hash{}, false)

			if err != nil {
				t.Fatalf("failed to create state DB: %v", err)
			}

			// Closing of state DB
			defer func(sDB state.StateDB) {
				err = sDB.Close()
				if err != nil {
					t.Fatalf("failed to close state DB: %v", err)
				}
			}(sDB)

			// Generating randomized world state
			ws, _ := makeWorldState(t)

			log := NewLogger("INFO", "TestStateDb")

			// Priming state DB
			PrimeStateDB(ws, sDB, cfg, log)

			// Checks if state DB was primed correctly
			for key, account := range ws {
				if sDB.GetBalance(key).Cmp(account.Balance) != 0 {
					t.Fatalf("failed to prime account balance; Is: %v; Should be: %v", sDB.GetBalance(key), account.Balance)
				}

				if sDB.GetNonce(key) != account.Nonce {
					t.Fatalf("failed to prime account nonce; Is: %v; Should be: %v", sDB.GetNonce(key), account.Nonce)
				}

				if bytes.Compare(sDB.GetCode(key), account.Code) != 0 {
					t.Fatalf("failed to prime account code; Is: %v; Should be: %v", sDB.GetCode(key), account.Code)
				}

				for sKey, sValue := range account.Storage {
					if sDB.GetState(key, sKey) != sValue {
						t.Fatalf("failed to prime account storage; Is: %v; Should be: %v", sDB.GetState(key, sKey), sValue)
					}
				}
			}
		})
	}
}

// TestStatedb_DeleteDestroyedAccountsFromWorldState tests removal of destroyed accounts from given world state
func TestStatedb_DeleteDestroyedAccountsFromWorldState(t *testing.T) {
	for _, tc := range getStatedbTestCases() {
		t.Run(fmt.Sprintf("DB variant: %s; shadowImpl: %s; archive variant: %s", tc.variant, tc.shadowImpl, tc.archiveVariant), func(t *testing.T) {
			cfg := makeTestConfig(tc)
			// Generating randomized world state
			ws, addrList := makeWorldState(t)
			// Init directory for destroyed accounts DB
			deletionDb := t.TempDir()
			// Pick two account which will represent destroyed ones
			destroyedAccounts := []common.Address{
				addrList[0],
				addrList[50],
			}

			// Update config to enable removal of destroyed accounts
			cfg.HasDeletedAccounts = true
			cfg.DeletionDb = deletionDb

			// Initializing backend DB for storing destroyed accounts
			daBackend, err := rawdb.NewLevelDBDatabase(deletionDb, 1024, 100, "destroyed_accounts", false)
			if err != nil {
				t.Fatalf("failed to create backend DB: %s; %v", deletionDb, err)
			}

			// Creating new destroyed accounts DB
			daDB := substate.NewDestroyedAccountDB(daBackend)

			// Storing two picked accounts from destroyedAccounts slice to destroyed accounts DB
			err = daDB.SetDestroyedAccounts(5, 1, destroyedAccounts, []common.Address{})
			if err != nil {
				t.Fatalf("failed to set destroyed accounts into DB: %v", err)
			}

			// Closing destroyed accounts DB
			err = daDB.Close()
			if err != nil {
				t.Fatalf("failed to close destroyed accounts DB: %v", err)
			}

			// Call for removal of destroyed accounts from given world state
			err = DeleteDestroyedAccountsFromWorldState(ws, cfg, 5)
			if err != nil {
				t.Fatalf("failed to delete accounts from the world state: %v", err)
			}

			// check if accounts are not present anymore
			if ws[destroyedAccounts[0]] != nil || ws[destroyedAccounts[1]] != nil {
				t.Fatalf("failed to delete accounts from the world state")
			}
		})
	}
}

// TestStatedb_DeleteDestroyedAccountsFromWorldState tests removal of deleted accounts from given state DB
func TestStatedb_DeleteDestroyedAccountsFromStateDB(t *testing.T) {
	for _, tc := range getStatedbTestCases() {
		t.Run(fmt.Sprintf("DB variant: %s; shadowImpl: %s; archive variant: %s", tc.variant, tc.shadowImpl, tc.archiveVariant), func(t *testing.T) {
			cfg := makeTestConfig(tc)
			// Generating randomized world state
			ws, addrList := makeWorldState(t)
			// Init directory for destroyed accounts DB
			deletedAccountsDir := t.TempDir()
			// Pick two account which will represent destroyed ones
			destroyedAccounts := []common.Address{
				addrList[0],
				addrList[50],
			}

			// Update config to enable removal of destroyed accounts
			cfg.HasDeletedAccounts = true
			cfg.DeletionDb = deletedAccountsDir

			// Initializing backend DB for storing destroyed accounts
			daBackend, err := rawdb.NewLevelDBDatabase(deletedAccountsDir, 1024, 100, "destroyed_accounts", false)
			if err != nil {
				t.Fatalf("failed to create backend DB: %s; %v", deletedAccountsDir, err)
			}

			// Creating new destroyed accounts DB
			daDB := substate.NewDestroyedAccountDB(daBackend)

			// Storing two picked accounts from destroyedAccounts slice to destroyed accounts DB
			err = daDB.SetDestroyedAccounts(5, 1, destroyedAccounts, []common.Address{})
			if err != nil {
				t.Fatalf("failed to set destroyed accounts into DB: %v", err)
			}

			// Closing destroyed accounts DB
			err = daDB.Close()
			if err != nil {
				t.Fatalf("failed to close destroyed accounts DB: %v", err)
			}

			// Initialization of state DB
			sDB, err := MakeStateDB(t.TempDir(), cfg, common.Hash{}, false)
			if err != nil {
				t.Fatalf("failed to create state DB: %v", err)
			}

			// Closing of state DB
			defer func(sDB state.StateDB) {
				err = sDB.Close()
				if err != nil {
					t.Fatalf("failed to close state DB: %v", err)
				}
			}(sDB)

			log := NewLogger("INFO", "TestStateDb")

			// Priming state DB with given world state
			PrimeStateDB(ws, sDB, cfg, log)

			// Call for removal of destroyed accounts from state DB
			err = DeleteDestroyedAccountsFromStateDB(sDB, cfg, 5)
			if err != nil {
				t.Fatalf("failed to delete accounts from the state DB: %v", err)
			}

			// check if accounts are not present anymore
			for _, da := range destroyedAccounts {
				if sDB.Exist(da) {
					t.Fatalf("failed to delete destroyed accounts from the state DB")
				}
			}
		})
	}
}

// TestStatedb_ValidateStateDB tests validation of state DB by comparing it to valid world state
func TestStatedb_ValidateStateDB(t *testing.T) {
	for _, tc := range getStatedbTestCases() {
		t.Run(fmt.Sprintf("DB variant: %s; shadowImpl: %s; archive variant: %s", tc.variant, tc.shadowImpl, tc.archiveVariant), func(t *testing.T) {
			cfg := makeTestConfig(tc)

			// Initialization of state DB
			sDB, err := MakeStateDB(t.TempDir(), cfg, common.Hash{}, false)
			if err != nil {
				t.Fatalf("failed to create state DB: %v", err)
			}

			// Closing of state DB
			defer func(sDB state.StateDB) {
				err = sDB.Close()
				if err != nil {
					t.Fatalf("failed to close state DB: %v", err)
				}
			}(sDB)

			// Generating randomized world state
			ws, _ := makeWorldState(t)

			log := NewLogger("INFO", "TestStateDb")

			// Priming state DB with given world state
			PrimeStateDB(ws, sDB, cfg, log)

			// Call for state DB validation and subsequent check for error
			err = ValidateStateDB(ws, sDB, false)
			if err != nil {
				t.Fatalf("failed to validate state DB: %v", err)
			}
		})
	}
}

// TestStatedb_ValidateStateDBWithUpdate test state DB validation comparing it to valid world state
// given state DB should be updated if world state contains different data
func TestStatedb_ValidateStateDBWithUpdate(t *testing.T) {
	for _, tc := range getStatedbTestCases() {
		t.Run(fmt.Sprintf("DB variant: %s; shadowImpl: %s; archive variant: %s", tc.variant, tc.shadowImpl, tc.archiveVariant), func(t *testing.T) {
			cfg := makeTestConfig(tc)

			// Initialization of state DB
			sDB, err := MakeStateDB(t.TempDir(), cfg, common.Hash{}, false)
			if err != nil {
				t.Fatalf("failed to create state DB: %v", err)
			}

			// Closing of state DB
			defer func(sDB state.StateDB) {
				err = sDB.Close()
				if err != nil {
					t.Fatalf("failed to close state DB: %v", err)
				}
			}(sDB)

			// Generating randomized world state
			ws, _ := makeWorldState(t)

			log := NewLogger("INFO", "TestStateDb")

			// Priming state DB with given world state
			PrimeStateDB(ws, sDB, cfg, log)

			// create new random address
			addr := common.BytesToAddress(makeRandomByteSlice(t, 40))

			// create new account
			ws[addr] = &substate.SubstateAccount{
				Nonce:   uint64(getRandom(1, 1000*5000)),
				Balance: big.NewInt(int64(getRandom(1, 1000*5000))),
				Storage: makeAccountStorage(t),
				Code:    makeRandomByteSlice(t, 2048),
			}

			// Call for state DB validation with update enabled and subsequent checks if the update was made correctly
			err = ValidateStateDB(ws, sDB, true)
			if err == nil {
				t.Fatalf("failed to throw errors while validating state DB: %v", err)
			}

			if sDB.GetBalance(addr).Cmp(ws[addr].Balance) != 0 {
				t.Fatalf("failed to prime account balance; Is: %v; Should be: %v", sDB.GetBalance(addr), ws[addr].Balance)
			}

			if sDB.GetNonce(addr) != ws[addr].Nonce {
				t.Fatalf("failed to prime account nonce; Is: %v; Should be: %v", sDB.GetNonce(addr), ws[addr].Nonce)
			}

			if bytes.Compare(sDB.GetCode(addr), ws[addr].Code) != 0 {
				t.Fatalf("failed to prime account code; Is: %v; Should be: %v", sDB.GetCode(addr), ws[addr].Code)
			}

			for sKey, sValue := range ws[addr].Storage {
				if sDB.GetState(addr, sKey) != sValue {
					t.Fatalf("failed to prime account storage; Is: %v; Should be: %v", sDB.GetState(addr, sKey), sValue)
				}
			}
		})
	}
}

// TestStatedb_PrepareStateDB tests preparation and initialization of existing state DB
func TestStatedb_PrepareStateDB(t *testing.T) {
	for _, tc := range getStatedbTestCases() {
		t.Run(fmt.Sprintf("DB variant: %s; shadowImpl: %s; archive variant: %s", tc.variant, tc.shadowImpl, tc.archiveVariant), func(t *testing.T) {
			cfg := makeTestConfig(tc)
			// Update config for state DB preparation by providing additional information
			cfg.ShadowImpl = ""
			cfg.DbTmp = t.TempDir()
			cfg.StateDbSrc = t.TempDir()
			cfg.First = 2

			// Create state DB info of existing state DB
			dbInfo := StateDbInfo{
				Impl:           cfg.DbImpl,
				Variant:        cfg.DbVariant,
				ArchiveMode:    cfg.ArchiveMode,
				ArchiveVariant: cfg.ArchiveVariant,
				Schema:         0,
				Block:          cfg.First - 1,
				RootHash:       common.Hash{},
				GitCommit:      GitCommit,
				CreateTime:     time.Now().UTC().Format(time.UnixDate),
			}

			// Create json file for the existing state DB info
			dbInfoJson, err := json.Marshal(dbInfo)
			if err != nil {
				t.Fatalf("failed to create DB info json: %v", err)
			}

			// Fill the json file with the info
			err = os.WriteFile(filepath.Join(cfg.StateDbSrc, DbInfoName), dbInfoJson, 0644)
			if err != nil {
				t.Fatalf("failed to write into DB info json file: %v", err)
			}

			// remove files after test ends
			defer func(path string) {
				err = os.RemoveAll(path)
				if err != nil {

				}
			}(cfg.StateDbSrc)

			// Call for state DB preparation and subsequent check if it finished successfully
			sDB, _, _, err := PrepareStateDB(cfg)
			if err != nil {
				t.Fatalf("failed to create state DB: %v", err)
			}

			// Closing of state DB
			defer func(sDB state.StateDB) {
				err = sDB.Close()
				if err != nil {
					t.Fatalf("failed to close state DB: %v", err)
				}
			}(sDB)
		})
	}
}

// TestStatedb_PrepareStateDB tests preparation and initialization of existing state DB as empty
// because of missing DbInfoName file
func TestStatedb_PrepareStateDBEmpty(t *testing.T) {
	tc := getStatedbTestCases()[0]
	cfg := makeTestConfig(tc)
	// Update config for state DB preparation by providing additional information
	cfg.ShadowImpl = ""
	cfg.DbTmp = t.TempDir()
	cfg.StateDbSrc = t.TempDir()
	cfg.First = 2

	// Call for state DB preparation and subsequent check if it finished successfully
	sDB, _, _, err := PrepareStateDB(cfg)
	if err != nil {
		t.Fatalf("failed to create state DB: %v", err)
	}

	// Closing of state DB
	defer func(sDB state.StateDB) {
		err = sDB.Close()
		if err != nil {
			t.Fatalf("failed to close state DB: %v", err)
		}
	}(sDB)
}

// TestStatedb_PrepareStateDBInvalid tests preparation and initialization of existing state DBs
// with multiple invalid DbInfoName files
func TestStatedb_PrepareStateDBInvalid(t *testing.T) {
	// Pick only one test case as other are not necessary
	tc := getStatedbTestCases()[0]
	cfg := makeTestConfig(tc)
	// Update config for state DB preparation by providing additional information
	cfg.ShadowImpl = ""
	cfg.DbTmp = t.TempDir()
	cfg.StateDbSrc = t.TempDir()
	cfg.First = 2

	// Generate invalid state DB info file data
	dbInfos := getInvalidStateDbInfo(cfg)

	for _, dbInfo := range dbInfos {
		t.Run(fmt.Sprintf("DB info: %v", dbInfo), func(t *testing.T) {
			// Add missing data to state DB info file
			dbInfo.RootHash = common.Hash{}
			dbInfo.GitCommit = GitCommit
			dbInfo.CreateTime = time.Now().UTC().Format(time.UnixDate)

			// Create json file for the existing state DB info
			dbInfoJson, err := json.Marshal(dbInfo)
			if err != nil {
				t.Fatalf("failed to create DB info json: %v", err)
			}

			// Fill the json file with the info
			err = os.WriteFile(filepath.Join(cfg.StateDbSrc, DbInfoName), dbInfoJson, 0644)
			if err != nil {
				t.Fatalf("failed to write into DB info json file: %v", err)
			}

			// Call for state DB preparation and subsequent check if it finished successfully
			_, _, _, err = PrepareStateDB(cfg)
			if err == nil {
				t.Fatal("failed to throw an error while creating state DB with invalid config")
			}
		})
	}

	// remove files after test ends
	err := os.RemoveAll(cfg.StateDbSrc)
	if err != nil {
		t.Fatalf("failed to remove state DB innfo file: %v", err)
	}
}
