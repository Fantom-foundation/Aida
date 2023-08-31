package utils

import (
	"flag"
	"fmt"
	"math"
	"math/big"
	"testing"

	"github.com/Fantom-foundation/Aida/logger"
	"github.com/urfave/cli/v2"

	"github.com/Fantom-foundation/Aida/state"
	"github.com/ethereum/go-ethereum/core/vm"
)

func prepareMockCliContext() *cli.Context {
	flagSet := flag.NewFlagSet("utils_config_test", 0)
	flagSet.Uint64(SyncPeriodLengthFlag.Name, 1000, "Number of blocks")
	flagSet.Bool(ValidateFlag.Name, true, "enables validation")
	flagSet.Bool(ValidateTxStateFlag.Name, true, "enables transaction state validation")
	flagSet.Bool(ContinueOnFailureFlag.Name, true, "continue execute after validation failure detected")
	flagSet.Bool(ValidateWorldStateFlag.Name, true, "enables end-state validation")
	flagSet.String(logger.LogLevelFlag.Name, "info", "Level of the logging of the app action (\"critical\", \"error\", \"warning\", \"notice\", \"info\", \"debug\"; default: INFO)")

	ctx := cli.NewContext(cli.NewApp(), flagSet, nil)

	command := &cli.Command{Name: "test_command"}
	ctx.Command = command

	return ctx
}

func TestUtilsConfig_GetChainConfig(t *testing.T) {
	testCases := []ChainID{
		TestnetChainID,
		MainnetChainID,
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("ChainID: %d", tc), func(t *testing.T) {
			chainConfig := GetChainConfig(tc)

			if tc == MainnetChainID && chainConfig.BerlinBlock.Cmp(new(big.Int).SetUint64(37455223)) != 0 {
				t.Fatalf("Incorrect Berlin fork block on chainID: %d; Block number: %d, should be: %d", MainnetChainID, chainConfig.BerlinBlock, 37455223)
			}

			if tc == MainnetChainID && chainConfig.LondonBlock.Cmp(new(big.Int).SetUint64(37534833)) != 0 {
				t.Fatalf("Incorrect London fork block on chainID: %d; Block number: %d, should be: %d", MainnetChainID, chainConfig.LondonBlock, 37534833)
			}

			if tc == TestnetChainID && chainConfig.BerlinBlock.Cmp(new(big.Int).SetUint64(1559470)) != 0 {
				t.Fatalf("Incorrect Berlin fork block on chainID: %d; Block number: %d, should be: %d", TestnetChainID, chainConfig.BerlinBlock, 1559470)
			}

			if tc == TestnetChainID && chainConfig.LondonBlock.Cmp(new(big.Int).SetUint64(7513335)) != 0 {
				t.Fatalf("Incorrect London fork block on chainID: %d; Block number: %d, should be: %d", TestnetChainID, chainConfig.LondonBlock, 7513335)
			}
		})
	}
}

func TestUtilsConfig_NewConfig(t *testing.T) {
	ctx := prepareMockCliContext()

	_, err := NewConfig(ctx, NoArgs)
	if err != nil {
		t.Fatalf("Failed to create new config: %v", err)
	}
}

func TestUtilsConfig_SetBlockRange(t *testing.T) {
	first, last, err := SetBlockRange("0", "40000000", 0)
	if err != nil {
		t.Fatalf("Failed to set block range (0-40000000): %v", err)
	}

	if first != uint64(0) {
		t.Fatalf("Failed to parse first block; expected: %d, have: %d", 0, first)
	}

	if last != uint64(40_000_000) {
		t.Fatalf("Failed to parse last block; expected: %d, have: %d", 40_000_000, last)
	}

	first, last, err = SetBlockRange("OpeRa", "berlin", 250)
	if err != nil {
		t.Fatalf("Failed to set block range (opera-berlin on mainnet): %v", err)
	}

	if first != uint64(4_564_026) {
		t.Fatalf("Failed to parse first block; expected: %d, have: %d", 4_564_026, first)
	}

	if last != uint64(37_455_223) {
		t.Fatalf("Failed to parse last block; expected: %d, have: %d", 37_455_223, last)
	}

	first, last, err = SetBlockRange("zero", "London", 4002)
	if err != nil {
		t.Fatalf("Failed to set block range (zero-london on testnet): %v", err)
	}

	if first != uint64(0) {
		t.Fatalf("Failed to parse first block; expected: %d, have: %d", 0, first)
	}

	if last != uint64(7_513_335) {
		t.Fatalf("Failed to parse last block; expected: %d, have: %d", 7_513_335, last)
	}

	// test addition/subtraction
	first, last, err = SetBlockRange("opera+23456", "London-100", 4002)
	if err != nil {
		t.Fatalf("Failed to set block range (opera+23456-London-100 on mainnet): %v", err)
	}

	if first != uint64(502_783) {
		t.Fatalf("Failed to parse first block; expected: %d, have: %d", 502_783, first)
	}

	if last != uint64(7_513_235) {
		t.Fatalf("Failed to parse last block; expected: %d, have: %d", 7_513_235, last)
	}

	// test upper/lower cases
	first, last, err = SetBlockRange("berlin-1000", "LonDoN", 250)
	if err != nil {
		t.Fatalf("Failed to set block range (berlin-1000-LonDoN on mainnet): %v", err)
	}

	if first != uint64(37_454_223) {
		t.Fatalf("Failed to parse first block; expected: %d, have: %d", 37_454_223, first)
	}

	if last != uint64(37_534_833) {
		t.Fatalf("Failed to parse last block; expected: %d, have: %d", 37_534_833, last)
	}

	// test first and last keyword. Since no metadata, default values are expected
	first, last, err = SetBlockRange("first", "last", 250)
	if err != nil {
		t.Fatalf("Failed to set block range (first-last on mainnet): %v", err)
	}

	if first != uint64(0) {
		t.Fatalf("Failed to parse first block; expected: %d, have: %d", 0, first)
	}

	if last != math.MaxUint64 {
		t.Fatalf("Failed to parse last block; expected: %v, have: %v", uint64(math.MaxUint64), last)
	}

	// test lastpatch and last keyword
	first, last, err = SetBlockRange("lastpatch", "last", 250)
	if err != nil {
		t.Fatalf("Failed to set block range (lastpatch-last on mainnet): %v", err)
	}

	if first != uint64(0) {
		t.Fatalf("Failed to parse first block; expected: %d, have: %d", 0, first)
	}

	if last != math.MaxUint64 {
		t.Fatalf("Failed to parse last block; expected: %v, have: %v", uint64(math.MaxUint64), last)
	}
}

func TestUtilsConfig_SetInvalidBlockRange(t *testing.T) {
	_, _, err := SetBlockRange("test", "40000000", 0)
	if err == nil {
		t.Fatalf("Failed to throw an error")
	}

	_, _, err = SetBlockRange("1000", "0", 4002)
	if err == nil {
		t.Fatalf("Failed to throw an error")
	}

	_, _, err = SetBlockRange("tokyo", "berlin", 250)
	if err == nil {
		t.Fatalf("Failed to throw an error")
	}

	_, _, err = SetBlockRange("tokyo", "berlin", 4002)
	if err == nil {
		t.Fatalf("Failed to throw an error")
	}

	_, _, err = SetBlockRange("london-opera", "opera+london", 250)
	if err == nil {
		t.Fatalf("Failed to throw an error")
	}

	_, _, err = SetBlockRange("london-opera", "opera+london", 4002)
	if err == nil {
		t.Fatalf("Failed to throw an error")
	}
}

func TestUtilsConfig_SetBlockRangeLastSmallerThanFirst(t *testing.T) {
	_, _, err := SetBlockRange("5", "0", 0)
	if err == nil {
		t.Fatalf("Failed to throw an error when last block number is smaller than first")
	}
}

// TestUtilsConfig_VmImplsAreRegistered checks if interpreters are correctly registered
func TestUtilsConfig_VmImplsAreRegistered(t *testing.T) {
	checkedImpls := []string{"lfvm", "lfvm-si", "geth"}

	statedb := state.MakeInMemoryStateDB(nil, 0)
	defer func(statedb state.StateDB) {
		err := statedb.Close()
		if err != nil {
			t.Errorf("Unable to close stateDB: %v", err)
		}
	}(statedb)
	chainConfig := GetChainConfig(0xFA)

	for _, interpreterImpl := range checkedImpls {
		evm := vm.NewEVM(vm.BlockContext{}, vm.TxContext{}, statedb, chainConfig, vm.Config{
			InterpreterImpl: interpreterImpl,
		})
		if evm == nil {
			t.Errorf("Unable to create EVM with InterpreterImpl %s", interpreterImpl)
		}
	}
}
