// Copyright 2024 Fantom Foundation
// This file is part of Aida Testing Infrastructure for Sonic
//
// Aida is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Aida is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with Aida. If not, see <http://www.gnu.org/licenses/>.

package utils

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/state"
	substatecontext "github.com/Fantom-foundation/Aida/txcontext/substate"
	substate "github.com/Fantom-foundation/Substate"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
)

// TestStatedb_InitCloseStateDB test closing db immediately after initialization
func TestStatedb_InitCloseStateDB(t *testing.T) {
	for _, tc := range GetStateDbTestCases() {
		t.Run(fmt.Sprintf("DB variant: %s; shadowImpl: %s; archive variant: %s", tc.Variant, tc.ShadowImpl, tc.ArchiveVariant), func(t *testing.T) {
			cfg := MakeTestConfig(tc)

			// Initialization of state DB
			sDB, _, err := PrepareStateDB(cfg)

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

// TestStatedb_DeleteDestroyedAccountsFromWorldState tests removal of destroyed accounts from given world state
func TestStatedb_DeleteDestroyedAccountsFromWorldState(t *testing.T) {
	for _, tc := range GetStateDbTestCases() {
		t.Run(fmt.Sprintf("DB variant: %s; shadowImpl: %s; archive variant: %s", tc.Variant, tc.ShadowImpl, tc.ArchiveVariant), func(t *testing.T) {
			cfg := MakeTestConfig(tc)
			// Generating randomized world state
			alloc, addrList := MakeWorldState(t)
			ws := substatecontext.NewWorldState(alloc)
			// Init directory for destroyed accounts DB
			deletionDb := t.TempDir()
			// Pick two account which will represent destroyed ones
			destroyedAccounts := []common.Address{
				addrList[0],
				addrList[50],
			}

			// Update config to enable removal of destroyed accounts
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
			if ws.Get(destroyedAccounts[0]) != nil || ws.Get(destroyedAccounts[1]) != nil {
				t.Fatalf("failed to delete accounts from the world state")
			}
		})
	}
}

// TestStatedb_DeleteDestroyedAccountsFromWorldState tests removal of deleted accounts from given state DB
func TestStatedb_DeleteDestroyedAccountsFromStateDB(t *testing.T) {
	for _, tc := range GetStateDbTestCases() {
		t.Run(fmt.Sprintf("DB variant: %s; shadowImpl: %s; archive variant: %s", tc.Variant, tc.ShadowImpl, tc.ArchiveVariant), func(t *testing.T) {
			cfg := MakeTestConfig(tc)
			// Generating randomized world state
			alloc, addrList := MakeWorldState(t)
			ws := substatecontext.NewWorldState(alloc)
			// Init directory for destroyed accounts DB
			deletedAccountsDir := t.TempDir()
			// Pick two account which will represent destroyed ones
			destroyedAccounts := []common.Address{
				addrList[0],
				addrList[50],
			}

			// Update config to enable removal of destroyed accounts
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
			sDB, _, err := PrepareStateDB(cfg)
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

			log := logger.NewLogger("INFO", "TestStateDb")

			// Create new prime context
			pc := NewPrimeContext(cfg, sDB, log)
			// Priming state DB with given world state
			pc.PrimeStateDB(ws, sDB)

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

// TestStatedb_PrepareStateDB tests preparation and initialization of existing state DB
func TestStatedb_PrepareStateDB(t *testing.T) {
	for _, tc := range GetStateDbTestCases() {
		t.Run(fmt.Sprintf("DB variant: %s; shadowImpl: %s; archive variant: %s", tc.Variant, tc.ShadowImpl, tc.ArchiveVariant), func(t *testing.T) {
			cfg := MakeTestConfig(tc)
			// Update config for state DB preparation by providing additional information
			cfg.DbTmp = t.TempDir()
			cfg.StateDbSrc = t.TempDir()
			cfg.First = 2
			cfg.Last = 4

			// Create state DB info of existing state DB
			dbInfo := StateDbInfo{
				Impl:           cfg.DbImpl,
				Variant:        cfg.DbVariant,
				ArchiveMode:    cfg.ArchiveMode,
				ArchiveVariant: cfg.ArchiveVariant,
				Schema:         0,
				Block:          cfg.Last,
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
			err = os.WriteFile(filepath.Join(cfg.StateDbSrc, PathToDbInfo), dbInfoJson, 0755)
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
			sDB, _, err := PrepareStateDB(cfg)
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
// because of missing PathToDbInfo file
func TestStatedb_PrepareStateDBEmpty(t *testing.T) {
	tc := GetStateDbTestCases()[0]
	cfg := MakeTestConfig(tc)
	// Update config for state DB preparation by providing additional information
	cfg.ShadowImpl = ""
	cfg.DbTmp = t.TempDir()
	cfg.First = 2

	// Call for state DB preparation and subsequent check if it finished successfully
	sDB, _, err := PrepareStateDB(cfg)
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
