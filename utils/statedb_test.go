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

func getRandom(rangeLower int, rangeUpper int) int {
	// seed the PRNG
	rand.Seed(time.Now().UnixNano())

	// get randomized balance
	randInt := rangeLower + rand.Intn(rangeUpper-rangeLower+1)
	return randInt
}

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
		cfg.PrimeSeed = int64(getRandom(1_000_000, 100_000_000))
	}

	return cfg
}

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

			sDB, err := MakeStateDB(t.TempDir(), cfg, common.Hash{}, false)

			if err != nil {
				t.Fatalf("failed to create state DB: %v", err)
			}

			err = sDB.Close()
			if err != nil {
				t.Fatalf("failed to close state DB: %v", err)
			}
		})
	}
}

func TestStatedb_PrimeStateDB(t *testing.T) {
	for _, tc := range getStatedbTestCases() {
		t.Run(fmt.Sprintf("DB variant: %s; shadowImpl: %s; archive variant: %s", tc.variant, tc.shadowImpl, tc.archiveVariant), func(t *testing.T) {
			cfg := makeTestConfig(tc)

			sDB, err := MakeStateDB(t.TempDir(), cfg, common.Hash{}, false)

			if err != nil {
				t.Fatalf("failed to create state DB: %v", err)
			}

			defer func(sDB state.StateDB) {
				err = sDB.Close()
				if err != nil {
					t.Fatalf("failed to close state DB: %v", err)
				}
			}(sDB)

			ws, _ := makeWorldState(t)

			PrimeStateDB(ws, sDB, cfg)

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

func TestStatedb_DeleteDestroyedAccountsFromWorldState(t *testing.T) {
	for _, tc := range getStatedbTestCases() {
		t.Run(fmt.Sprintf("DB variant: %s; shadowImpl: %s; archive variant: %s", tc.variant, tc.shadowImpl, tc.archiveVariant), func(t *testing.T) {
			ws, addrList := makeWorldState(t)
			cfg := makeTestConfig(tc)
			deletedAccountsDir := t.TempDir()
			destroyedAccounts := []common.Address{
				addrList[0],
				addrList[50],
			}

			cfg.HasDeletedAccounts = true
			cfg.DeletedAccountDir = deletedAccountsDir

			daBackend, err := rawdb.NewLevelDBDatabase(deletedAccountsDir, 1024, 100, "destroyed_accounts", false)
			if err != nil {
				t.Fatalf("failed to create backend DB: %s; %v", deletedAccountsDir, err)
			}
			daDB := substate.NewDestroyedAccountDB(daBackend)

			err = daDB.SetDestroyedAccounts(5, 1, destroyedAccounts, []common.Address{})
			if err != nil {
				t.Fatalf("failed to set destroyed accounts into DB: %v", err)
			}

			err = daDB.Close()
			if err != nil {
				t.Fatalf("failed to close destroyed accounts DB: %v", err)
			}

			err = DeleteDestroyedAccountsFromWorldState(ws, cfg, 5)
			if err != nil {
				t.Fatalf("failed to delete accounts from the world state: %v", err)
			}

			if ws[destroyedAccounts[0]] != nil || ws[destroyedAccounts[1]] != nil {
				t.Fatalf("failed to delete accounts from the world state")
			}
		})
	}
}

func TestStatedb_DeleteDestroyedAccountsFromStateDB(t *testing.T) {
	for _, tc := range getStatedbTestCases() {
		t.Run(fmt.Sprintf("DB variant: %s; shadowImpl: %s; archive variant: %s", tc.variant, tc.shadowImpl, tc.archiveVariant), func(t *testing.T) {
			cfg := makeTestConfig(tc)
			ws, addrList := makeWorldState(t)
			deletedAccountsDir := t.TempDir()
			destroyedAccounts := []common.Address{
				addrList[0],
				addrList[50],
			}

			cfg.HasDeletedAccounts = true
			cfg.DeletedAccountDir = deletedAccountsDir

			daBackend, err := rawdb.NewLevelDBDatabase(deletedAccountsDir, 1024, 100, "destroyed_accounts", false)
			if err != nil {
				t.Fatalf("failed to create backend DB: %s; %v", deletedAccountsDir, err)
			}
			daDB := substate.NewDestroyedAccountDB(daBackend)

			err = daDB.SetDestroyedAccounts(5, 1, destroyedAccounts, []common.Address{})
			if err != nil {
				t.Fatalf("failed to set destroyed accounts into DB: %v", err)
			}

			err = daDB.Close()
			if err != nil {
				t.Fatalf("failed to close destroyed accounts DB: %v", err)
			}

			sDB, err := MakeStateDB(t.TempDir(), cfg, common.Hash{}, false)
			if err != nil {
				t.Fatalf("failed to create state DB: %v", err)
			}

			defer func(sDB state.StateDB) {
				err = sDB.Close()
				if err != nil {
					t.Fatalf("failed to close state DB: %v", err)
				}
			}(sDB)

			PrimeStateDB(ws, sDB, cfg)

			err = DeleteDestroyedAccountsFromStateDB(sDB, cfg, 5)
			if err != nil {
				t.Fatalf("failed to delete accounts from the state DB: %v", err)
			}

			for _, da := range destroyedAccounts {
				if sDB.Exist(da) {
					t.Fatalf("failed to delete destroyed accounts from the state DB")
				}
			}
		})
	}
}

func TestStatedb_ValidateStateDB(t *testing.T) {
	for _, tc := range getStatedbTestCases() {
		t.Run(fmt.Sprintf("DB variant: %s; shadowImpl: %s; archive variant: %s", tc.variant, tc.shadowImpl, tc.archiveVariant), func(t *testing.T) {
			cfg := makeTestConfig(tc)

			sDB, err := MakeStateDB(t.TempDir(), cfg, common.Hash{}, false)
			if err != nil {
				t.Fatalf("failed to create state DB: %v", err)
			}

			defer func(sDB state.StateDB) {
				err = sDB.Close()
				if err != nil {
					t.Fatalf("failed to close state DB: %v", err)
				}
			}(sDB)

			ws, _ := makeWorldState(t)

			PrimeStateDB(ws, sDB, cfg)

			err = ValidateStateDB(ws, sDB, false)
			if err != nil {
				t.Fatalf("failed to validate state DB: %v", err)
			}
		})
	}
}

func TestStatedb_ValidateStateDBWithUpdate(t *testing.T) {
	for _, tc := range getStatedbTestCases() {
		t.Run(fmt.Sprintf("DB variant: %s; shadowImpl: %s; archive variant: %s", tc.variant, tc.shadowImpl, tc.archiveVariant), func(t *testing.T) {
			cfg := makeTestConfig(tc)

			sDB, err := MakeStateDB(t.TempDir(), cfg, common.Hash{}, false)
			if err != nil {
				t.Fatalf("failed to create state DB: %v", err)
			}

			defer func(sDB state.StateDB) {
				err = sDB.Close()
				if err != nil {
					t.Fatalf("failed to close state DB: %v", err)
				}
			}(sDB)

			ws, _ := makeWorldState(t)

			PrimeStateDB(ws, sDB, cfg)

			// create new random address
			addr := common.BytesToAddress(makeRandomByteSlice(t, 40))

			// create new account
			ws[addr] = &substate.SubstateAccount{
				Nonce:   uint64(getRandom(1, 1000*5000)),
				Balance: big.NewInt(int64(getRandom(1, 1000*5000))),
				Storage: makeAccountStorage(t),
				Code:    makeRandomByteSlice(t, 2048),
			}

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

func TestStatedb_PrepareStateDB(t *testing.T) {
	for _, tc := range getStatedbTestCases() {
		t.Run(fmt.Sprintf("DB variant: %s; shadowImpl: %s; archive variant: %s", tc.variant, tc.shadowImpl, tc.archiveVariant), func(t *testing.T) {
			cfg := makeTestConfig(tc)
			cfg.ShadowImpl = ""
			cfg.StateDbTempDir = t.TempDir()
			cfg.StateDbSrcDir = t.TempDir()
			cfg.First = 2

			dbInfo := StateDbInfo{
				Impl:           cfg.DbImpl,
				Variant:        cfg.DbVariant,
				ArchiveMode:    cfg.ArchiveMode,
				ArchiveVariant: cfg.ArchiveVariant,
				Schema:         0,
				Block:          cfg.First - 1,
				RootHash:       common.Hash{},
				GitCommit:      "add7feb04cea97ec770af7b179cf3ba83f87fa70",
				CreateTime:     time.Now().String(),
			}

			dbInfoJson, err := json.Marshal(dbInfo)
			if err != nil {
				t.Fatalf("failed to create DB info json: %v", err)
			}
			err = os.WriteFile(filepath.Join(cfg.StateDbSrcDir, DbInfoName), dbInfoJson, 0644)
			if err != nil {
				t.Fatalf("failed to write into DB info json file: %v", err)
			}
			defer func(path string) {
				err = os.RemoveAll(path)
				if err != nil {

				}
			}(cfg.StateDbSrcDir)

			sDB, _, _, err := PrepareStateDB(cfg)
			if err != nil {
				t.Fatalf("failed to create state DB: %v", err)
			}

			defer func(sDB state.StateDB) {
				err = sDB.Close()
				if err != nil {
					t.Fatalf("failed to close state DB: %v", err)
				}
			}(sDB)
		})
	}
}
