package statedb

import (
	"os"
	"testing"

	statetest "github.com/Fantom-foundation/Aida/ethtest/statetest"
	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/txcontext"
	"github.com/Fantom-foundation/Aida/utils"
)

func Test_ethStateTestDbPrepper_PreBlockPreparesAStateDB(t *testing.T) {
	cfg := &utils.Config{
		DbImpl:   "geth",
		ChainID:  1,
		LogLevel: "critical",
	}
	ext := makeEthTestDbPrepper(logger.NewLogger(cfg.LogLevel, "EthStatePrepper"), cfg)

	testData := statetest.CreateTestData(t)
	st := executor.State[txcontext.TxContext]{Block: 1, Transaction: 1, Data: testData}
	ctx := &executor.Context{}
	err := ext.PreBlock(st, ctx)
	if err != nil {
		t.Fatalf("unexpected err; %v", err)
	}

	if ctx.State == nil {
		t.Fatalf("failed to initialize a DB instance")
	}
}

func Test_ethStateTestDbPrepper_PostBlockDeletesDatabase(t *testing.T) {
	cfg := &utils.Config{
		DbImpl:   "geth",
		ChainID:  1,
		LogLevel: "critical",
	}
	ext := makeEthTestDbPrepper(logger.NewLogger(cfg.LogLevel, "EthStatePrepper"), cfg)

	testData := statetest.CreateTestData(t)
	st := executor.State[txcontext.TxContext]{Block: 1, Transaction: 1, Data: testData}
	ctx := &executor.Context{}
	err := ext.PreBlock(st, ctx)
	if err != nil {
		t.Fatalf("unexpected err; %v", err)
	}
	dirPath := ctx.StateDbPath
	// check if exists before removing
	if _, err := os.Stat(dirPath); os.IsNotExist(err) {
		t.Fatal("tmp dir not found")
	}

	err = ext.PostBlock(st, ctx)

	// check if tmp dir is removed
	if _, err := os.Stat(dirPath); !os.IsNotExist(err) {
		t.Fatal("tmp dir not removed")
	}
}
