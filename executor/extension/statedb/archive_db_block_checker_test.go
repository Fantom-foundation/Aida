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

package statedb

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/utils"
	"github.com/ethereum/go-ethereum/common"
)

func TestArchiveDbBlockChecker_PreRunReturnsErrorIfStateDbDoesNotHaveArchive(t *testing.T) {
	cfg := &utils.Config{}
	cfg.StateDbSrc = t.TempDir()
	err := utils.WriteStateDbInfo(cfg.StateDbSrc, cfg, 0, common.Hash{}, true)
	if err != nil {
		t.Fatalf("cannot create testing state db info; %v", err)
	}

	ext := MakeArchiveBlockChecker[any](cfg)
	err = ext.PreRun(executor.State[any]{}, nil)
	if err == nil {
		t.Fatalf("pre-run must return error")
	}

	wantedErr := fmt.Sprintf("state db %v does not contain archive", cfg.StateDbSrc)

	if strings.Compare(err.Error(), wantedErr) != 0 {
		t.Fatalf("unexpected err\ngot: %v\n want: %v", err, wantedErr)
	}
}

func TestArchiveDbBlockChecker_PreRunReturnsErrorIfStateDbDoesNotContainGivenBlockRange(t *testing.T) {
	cfg := &utils.Config{}
	cfg.Last = 11

	cfg.StateDbSrc = t.TempDir()
	cfg.ArchiveMode = true
	err := utils.WriteStateDbInfo(cfg.StateDbSrc, cfg, 10, common.Hash{}, true)
	if err != nil {
		t.Fatalf("cannot create testing state db info; %v", err)
	}

	ext := MakeArchiveBlockChecker[any](cfg)
	err = ext.PreRun(executor.State[any]{}, nil)
	if err != nil || cfg.Last != 10 {
		t.Fatalf("Failed to adjust last block")
	}
}

func TestArchiveDbBlockChecker_PreRunReturnsErrorIfPrimeStateDbDoesNotHaveArchive(t *testing.T) {
	cfg := &utils.Config{}
	cfg.ShadowDb = true
	cfg.StateDbSrc = t.TempDir()
	if err := os.Mkdir(cfg.StateDbSrc+utils.PathToPrimaryStateDb, os.ModePerm); err != nil {
		t.Fatal(err)
	}
	err := utils.WriteStateDbInfo(cfg.StateDbSrc+utils.PathToPrimaryStateDb, cfg, 0, common.Hash{}, true)
	if err != nil {
		t.Fatalf("cannot create testing state db info %v", err)
	}

	ext := MakeArchiveBlockChecker[any](cfg)
	err = ext.PreRun(executor.State[any]{}, nil)
	if err == nil {
		t.Fatalf("pre-run must return error")
	}

	wantedErr := fmt.Sprintf("prime state db %v does not contain archive", filepath.Join(cfg.StateDbSrc, utils.PathToPrimaryStateDb))

	if strings.Compare(err.Error(), wantedErr) != 0 {
		t.Fatalf("unexpected err\ngot: %v\n want: %v", err, wantedErr)
	}
}

func TestArchiveDbBlockChecker_PreRunReturnsErrorIfPrimeStateDbDoesNotContainGivenBlockRange(t *testing.T) {
	cfg := &utils.Config{}
	cfg.Last = 11

	cfg.StateDbSrc = t.TempDir()
	cfg.ArchiveMode = true
	cfg.ShadowDb = true
	cfg.StateDbSrc = t.TempDir()

	if err := os.Mkdir(cfg.StateDbSrc+utils.PathToPrimaryStateDb, os.ModePerm); err != nil {
		t.Fatal(err)
	}
	err := utils.WriteStateDbInfo(cfg.StateDbSrc+utils.PathToPrimaryStateDb, cfg, 10, common.Hash{}, true)
	if err != nil {
		t.Fatalf("cannot create testing state db info %v", err)
	}

	if err = os.Mkdir(cfg.StateDbSrc+utils.PathToShadowStateDb, os.ModePerm); err != nil {
		t.Fatal(err)
	}

	err = utils.WriteStateDbInfo(cfg.StateDbSrc+utils.PathToShadowStateDb, cfg, 12, common.Hash{}, true)
	if err != nil {
		t.Fatalf("cannot create testing state db info %v", err)
	}

	ext := MakeArchiveBlockChecker[any](cfg)
	err = ext.PreRun(executor.State[any]{}, nil)
	if err != nil || cfg.Last != 10 {
		t.Fatalf("Failed to adjust last block")
	}
}

func TestArchiveDbBlockChecker_PreRunReturnsErrorIfShadowStateDbDoesNotContainGivenBlockRange(t *testing.T) {
	cfg := &utils.Config{}
	cfg.Last = 11

	cfg.StateDbSrc = t.TempDir()
	cfg.ArchiveMode = true
	cfg.ShadowDb = true
	cfg.StateDbSrc = t.TempDir()

	if err := os.Mkdir(cfg.StateDbSrc+utils.PathToPrimaryStateDb, os.ModePerm); err != nil {
		t.Fatal(err)
	}
	err := utils.WriteStateDbInfo(cfg.StateDbSrc+utils.PathToPrimaryStateDb, cfg, 12, common.Hash{}, true)
	if err != nil {
		t.Fatalf("cannot create testing state db info %v", err)
	}

	if err = os.Mkdir(cfg.StateDbSrc+utils.PathToShadowStateDb, os.ModePerm); err != nil {
		t.Fatal(err)
	}

	err = utils.WriteStateDbInfo(cfg.StateDbSrc+utils.PathToShadowStateDb, cfg, 10, common.Hash{}, true)
	if err != nil {
		t.Fatalf("cannot create testing state db info %v", err)
	}

	ext := MakeArchiveBlockChecker[any](cfg)
	err = ext.PreRun(executor.State[any]{}, nil)
	if err != nil || cfg.Last != 10 {
		t.Fatalf("fail to adjust last block")
	}
}

func TestArchiveDbBlockChecker_PreRunReturnsErrorIfShadowStateDbDoesNotHaveArchive(t *testing.T) {
	cfg := &utils.Config{}
	cfg.ShadowDb = true
	cfg.ArchiveMode = false
	cfg.StateDbSrc = t.TempDir()
	if err := os.Mkdir(cfg.StateDbSrc+utils.PathToShadowStateDb, os.ModePerm); err != nil {
		t.Fatal(err)
	}
	err := utils.WriteStateDbInfo(cfg.StateDbSrc+utils.PathToShadowStateDb, cfg, 0, common.Hash{}, true)
	if err != nil {
		t.Fatalf("cannot create testing state db info %v", err)
	}

	cfg.ArchiveMode = true

	if err = os.Mkdir(cfg.StateDbSrc+utils.PathToPrimaryStateDb, os.ModePerm); err != nil {
		t.Fatal(err)
	}
	err = utils.WriteStateDbInfo(cfg.StateDbSrc+utils.PathToPrimaryStateDb, cfg, 0, common.Hash{}, true)
	if err != nil {
		t.Fatalf("cannot create testing state db info %v", err)
	}

	ext := MakeArchiveBlockChecker[any](cfg)
	err = ext.PreRun(executor.State[any]{}, nil)
	if err == nil {
		t.Fatalf("pre-run must return error")
	}

	wantedErr := fmt.Sprintf("shadow state db %v does not contain archive", filepath.Join(cfg.StateDbSrc, utils.PathToShadowStateDb))

	if strings.Compare(err.Error(), wantedErr) != 0 {
		t.Fatalf("unexpected err\ngot: %v\n want: %v", err, wantedErr)
	}
}
