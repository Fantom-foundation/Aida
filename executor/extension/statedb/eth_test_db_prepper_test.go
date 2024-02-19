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

func Test_ethStateTestDbPrepper_PreBlockPrimes(t *testing.T) {
	cfg := &utils.Config{
		DbImpl:   "geth",
		ChainID:  1,
		LogLevel: "critical",
	}
	ext := ethTestDbPrepper{cfg: cfg, log: logger.NewLogger(cfg.LogLevel, "EthStatePrepper")}

	testData := statetest.CreateTestData(t)
	st := executor.State[txcontext.TxContext]{Block: 1, Transaction: 1, Data: testData}
	ctx := &executor.Context{}
	err := ext.PreBlock(st, ctx)
	if err != nil {
		t.Fatalf("unexpected err; %v", err)
	}

	expectedAlloc := testData.GetInputState()
	if expectedAlloc.Len() == 0 {
		t.Fatalf("no expected state")
	}

	vmAlloc := ctx.State.GetSubstatePostAlloc()
	isEqual := expectedAlloc.Equal(vmAlloc)
	if !isEqual {
		if err != nil {
			t.Fatalf("failed to prime database with test data\ngot: %v\nwant: %v", vmAlloc.String(), expectedAlloc.String())
		}
	}
}

func Test_ethStateTestDbPrepper_PostBlockDeletesDatabase(t *testing.T) {
	cfg := &utils.Config{
		DbImpl:   "geth",
		ChainID:  1,
		LogLevel: "critical",
	}
	ext := ethTestDbPrepper{cfg: cfg, log: logger.NewLogger(cfg.LogLevel, "EthStatePrepper")}

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
