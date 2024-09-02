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
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

// TestStatedbInfo_WriteReadStateDbInfo tests creation of state DB info json file,
// writing into it and subsequent reading from it
func TestStatedbInfo_WriteReadStateDbInfo(t *testing.T) {
	for _, tc := range GetStateDbTestCases() {
		t.Run(fmt.Sprintf("DB variant: %s; shadowImpl: %s; archive variant: %s", tc.Variant, tc.ShadowImpl, tc.ArchiveVariant), func(t *testing.T) {
			cfg := MakeTestConfig(tc)
			// Update config for state DB preparation by providing additional information
			cfg.DbTmp = t.TempDir()
			cfg.StateDbSrc = t.TempDir()

			// Call for json creation and writing into it
			err := WriteStateDbInfo(cfg.StateDbSrc, cfg, 2, common.Hash{}, true)
			if err != nil {
				t.Fatalf("failed to write into DB info json file: %v", err)
			}

			// Getting the DB info file path and call for reading from it
			dbInfoFile := filepath.Join(cfg.StateDbSrc, PathToDbInfo)
			dbInfo, err := ReadStateDbInfo(dbInfoFile)
			if err != nil {
				t.Fatalf("failed to read from DB info json file: %v", err)
			}

			// Subsequent checks if all given information have been written and read correctly
			if dbInfo.Impl != cfg.DbImpl {
				t.Fatalf("failed to write DbImpl into DB info json file correctly; Is: %s; Should be: %s", dbInfo.Impl, cfg.DbImpl)
			}
			if dbInfo.ArchiveMode != cfg.ArchiveMode {
				t.Fatalf("failed to write ArchiveMode into DB info json file correctly; Is: %v; Should be: %v", dbInfo.ArchiveMode, cfg.ArchiveMode)
			}
			if dbInfo.ArchiveVariant != cfg.ArchiveVariant {
				t.Fatalf("failed to write ArchiveVariant into DB info json file correctly; Is: %s; Should be: %s", dbInfo.ArchiveVariant, cfg.ArchiveVariant)
			}
			if dbInfo.Variant != cfg.DbVariant {
				t.Fatalf("failed to write DbVariant into DB info json file correctly; Is: %s; Should be: %s", dbInfo.Variant, cfg.DbVariant)
			}
			if dbInfo.Schema != cfg.CarmenSchema {
				t.Fatalf("failed to write CarmenSchema into DB info json file correctly; Is: %d; Should be: %d", dbInfo.Schema, cfg.CarmenSchema)
			}
			if dbInfo.Block != 2 {
				t.Fatalf("failed to write Block into DB info json file correctly; Is: %d; Should be: %d", dbInfo.Block, 2)
			}
			if dbInfo.RootHash != (common.Hash{}) {
				t.Fatalf("failed to write RootHash into DB info json file correctly; Is: %d; Should be: %d", dbInfo.RootHash, common.Hash{})
			}
			if dbInfo.GitCommit != GitCommit {
				t.Fatalf("failed to write GitCommit into DB info json file correctly; Is: %s; Should be: %s", dbInfo.GitCommit, GitCommit)
			}
			if !dbInfo.HasFinished {
				t.Fatalf("failed to write HasFinished into DB info json file correctly; Is: %v; Should be: %v", dbInfo.HasFinished, !dbInfo.HasFinished)
			}
		})
	}
}

// TestStatedbInfo_RenameTempStateDbDirectory tests renaming temporary state DB directory into something more meaningful
func TestStatedbInfo_RenameTempStateDbDirectory(t *testing.T) {
	for _, tc := range GetStateDbTestCases() {
		t.Run(fmt.Sprintf("DB variant: %s; shadowImpl: %s; archive variant: %s", tc.Variant, tc.ShadowImpl, tc.ArchiveVariant), func(t *testing.T) {
			cfg := MakeTestConfig(tc)
			// Update config for state DB preparation by providing additional information
			cfg.DbTmp = t.TempDir()
			oldDirectory := t.TempDir()
			block := uint64(2)

			// Call for renaming temporary state DB directory
			RenameTempStateDbDirectory(cfg, oldDirectory, block)

			// Generating directory name in the same format
			var newName string
			if cfg.DbImpl != "geth" {
				newName = fmt.Sprintf("state_db_%v_%v_%v", cfg.DbImpl, cfg.DbVariant, block)
			} else {
				newName = fmt.Sprintf("state_db_%v_%v", cfg.DbImpl, block)
			}

			// trying to find renamed directory
			newName = filepath.Join(cfg.DbTmp, newName)
			if _, err := os.Stat(newName); os.IsNotExist(err) {
				t.Fatalf("failed to rename temporary state DB directory")
			}
		})
	}
}

// TestStatedbInfo_RenameTempStateDbDirectory tests renaming temporary state DB directory into a custom name.
func TestStatedbInfo_RenameTempStateDbDirectoryToCustomName(t *testing.T) {
	cfg := &Config{
		DbImpl:       "geth",
		DbVariant:    "",
		CustomDbName: "TestName",
	}
	// Update config for state DB preparation by providing additional information
	cfg.DbTmp = t.TempDir()
	oldDirectory := t.TempDir()
	block := uint64(2)

	// Call for renaming temporary state DB directory
	RenameTempStateDbDirectory(cfg, oldDirectory, block)

	// trying to find renamed directory
	newName := filepath.Join(cfg.DbTmp, cfg.CustomDbName)
	if _, err := os.Stat(newName); os.IsNotExist(err) {
		t.Fatalf("failed to rename temporary state DB directory")
	}
}
