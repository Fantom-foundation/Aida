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

func Test_ethStateTestDbPrepper_PreTransactionPrepairsAStateDB(t *testing.T) {
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
