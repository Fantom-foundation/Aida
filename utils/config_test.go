package utils

import (
	"flag"
	"fmt"
	"math/big"
	"testing"

	"github.com/urfave/cli/v2"

	"github.com/Fantom-foundation/Aida/state"
	"github.com/ethereum/go-ethereum/core/vm"
)

const mainNetChainId int = 250
const testNetChainId int = 4002

func prepareMockCliContext() *cli.Context {
	flagSet := flag.NewFlagSet("utils_config_test", 0)
	flagSet.Uint64(NumberOfBlocksFlag.Name, 1000, "Number of blocks")
	flagSet.Bool(ValidateFlag.Name, true, "enables validation")
	flagSet.Bool(ValidateTxStateFlag.Name, true, "enables transaction state validation")
	flagSet.Bool(ContinueOnFailureFlag.Name, true, "continue execute after validation failure detected")
	flagSet.Bool(ValidateWorldStateFlag.Name, true, "enables end-state validation")

	ctx := cli.NewContext(cli.NewApp(), flagSet, nil)

	command := &cli.Command{Name: "test_command"}
	ctx.Command = command

	return ctx
}

func TestUtilsConfig_GetChainConfig(t *testing.T) {
	testCases := []int{
		testNetChainId,
		mainNetChainId,
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("ChainID: %d", tc), func(t *testing.T) {
			chainConfig := GetChainConfig(tc)

			if tc == mainNetChainId && chainConfig.BerlinBlock.Cmp(new(big.Int).SetUint64(37455223)) != 0 {
				t.Fatalf("Incorrect Berlin fork block on chainID: %d; Block number: %d, should be: %d", mainNetChainId, chainConfig.BerlinBlock, 37455223)
			}

			if tc == mainNetChainId && chainConfig.LondonBlock.Cmp(new(big.Int).SetUint64(37534833)) != 0 {
				t.Fatalf("Incorrect London fork block on chainID: %d; Block number: %d, should be: %d", mainNetChainId, chainConfig.LondonBlock, 37534833)
			}

			if tc == testNetChainId && chainConfig.BerlinBlock.Cmp(new(big.Int).SetUint64(1559470)) != 0 {
				t.Fatalf("Incorrect Berlin fork block on chainID: %d; Block number: %d, should be: %d", testNetChainId, chainConfig.BerlinBlock, 1559470)
			}

			if tc == testNetChainId && chainConfig.LondonBlock.Cmp(new(big.Int).SetUint64(7513335)) != 0 {
				t.Fatalf("Incorrect London fork block on chainID: %d; Block number: %d, should be: %d", testNetChainId, chainConfig.LondonBlock, 7513335)
			}
		})
	}
}

func TestUtilsConfig_NewConfig(t *testing.T) {
	ctx := prepareMockCliContext()

	_, err := NewConfig(ctx, 0)
	if err != nil {
		t.Fatalf("Failed to create new config: %v", err)
	}
}

func TestUtilsConfig_SetBlockRange(t *testing.T) {
	first, last, err := SetBlockRange("0", "40000000")
	if err != nil {
		t.Fatalf("Failed to set block range: %v", err)
	}

	if first != uint64(0) {
		t.Fatalf("Failed to parse first block; Should be: %d, but is: %d", 0, first)
	}

	if last != uint64(40_000_000) {
		t.Fatalf("Failed to parse last block; Should be: %d, but is: %d", 40_000_000, last)
	}
}

func TestUtilsConfig_SetBlockRangeLastSmallerThanFirst(t *testing.T) {
	_, _, err := SetBlockRange("5", "0")
	if err == nil {
		t.Fatalf("Failed to throw an error when last block number is smaller than first")
	}
}

// TestUtilsConfig_VmImplsAreRegistered checks if interpreters are correctly registered
func TestUtilsConfig_VmImplsAreRegistered(t *testing.T) {
	checkedImpls := []string{"lfvm", "lfvm-si", "geth"}

	statedb := state.MakeGethInMemoryStateDB(nil, 0)
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
