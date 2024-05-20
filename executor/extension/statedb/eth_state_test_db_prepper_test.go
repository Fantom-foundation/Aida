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
	"os"
	"testing"

	"github.com/Fantom-foundation/Aida/ethtest"
	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/txcontext"
	"github.com/Fantom-foundation/Aida/utils"
)

func Test_ethStateTestDbPrepper_PreTransactionPreparesAStateDB(t *testing.T) {
	cfg := &utils.Config{
		DbImpl:   "geth",
		ChainID:  1,
		LogLevel: "critical",
	}
	ext := ethStateTestDbPrepper{cfg: cfg, log: logger.NewLogger(cfg.LogLevel, "EthStatePrepper")}

	testData := ethtest.CreateTestData(t)
	st := executor.State[txcontext.TxContext]{Block: 1, Transaction: 1, Data: testData}
	ctx := &executor.Context{}
	err := ext.PreTransaction(st, ctx)
	if err != nil {
		t.Fatalf("unexpected err; %v", err)
	}

	if ctx.State == nil {
		t.Fatalf("failed to initialize a DB instance")
	}
}

func Test_ethStateTestDbPrepper_CleaningTmpDir(t *testing.T) {
	cfg := &utils.Config{
		DbImpl:   "geth",
		ChainID:  1,
		LogLevel: "critical",
	}
	ext := ethStateTestDbPrepper{cfg: cfg, log: logger.NewLogger(cfg.LogLevel, "EthStatePrepper")}

	testData := ethtest.CreateTestData(t)
	st := executor.State[txcontext.TxContext]{Block: 1, Transaction: 1, Data: testData}
	ctx := &executor.Context{}
	err := ext.PreTransaction(st, ctx)
	if err != nil {
		t.Fatalf("unexpected err; %v", err)
	}
	dirPath := ctx.StateDbPath
	// check if exists before removing
	if _, err := os.Stat(dirPath); os.IsNotExist(err) {
		t.Fatalf("tmp dir not found")
	}

	err = ext.PostTransaction(st, ctx)

	// check if tmp dir is removed
	if _, err := os.Stat(dirPath); !os.IsNotExist(err) {
		t.Fatalf("tmp dir not removed")
	}
}
