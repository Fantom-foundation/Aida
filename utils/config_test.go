package utils

import (
	"flag"
	"fmt"
	"math"
	"math/big"
	"os"
	"strconv"
	"testing"

	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/state"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/urfave/cli/v2"
)

func prepareMockCliContext() *cli.Context {
	flagSet := flag.NewFlagSet("utils_config_test", 0)
	flagSet.Uint64(SyncPeriodLengthFlag.Name, 1000, "Number of blocks")
	flagSet.Bool(ValidateFlag.Name, true, "enables validation")
	flagSet.Bool(ValidateTxStateFlag.Name, true, "enables transaction state validation")
	flagSet.Bool(ContinueOnFailureFlag.Name, true, "continue execute after validation failure detected")
	flagSet.Bool(ValidateWorldStateFlag.Name, true, "enables end-state validation")
	flagSet.String(AidaDbFlag.Name, "./test.db", "set substate, updateset and deleted accounts directory")
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

func TestUtilsConfig_adjustBlockRange(t *testing.T) {
	var (
		chainId           ChainID
		first, last       uint64
		firstArg, lastArg uint64
		err               error
	)
	chainId = MainnetChainID
	keywordBlocks[chainId]["first"] = 1000
	keywordBlocks[chainId]["last"] = 2000

	firstArg = 1100
	lastArg = 1900
	first, last, err = adjustBlockRange(chainId, firstArg, lastArg)
	if first != firstArg && last != lastArg {
		t.Fatalf("wrong block range; expected %v:%v, have %v:%v", firstArg, lastArg, first, last)
	}

	firstArg = 3000
	lastArg = 4000
	first, last, err = adjustBlockRange(chainId, firstArg, lastArg)
	if err == nil {
		t.Fatalf("Ranges not overlapped. Expected an error.")
	}

	// check corner cases
	firstArg = 100
	lastArg = 1000
	first, last, err = adjustBlockRange(chainId, firstArg, lastArg)
	if first != firstArg && last != lastArg {
		t.Fatalf("wrong block range; expected %v:%v, have %v:%v", lastArg, lastArg, first, last)
	}

	firstArg = 2000
	lastArg = 2200
	first, last, err = adjustBlockRange(chainId, firstArg, lastArg)
	if first != firstArg && last != lastArg {
		t.Fatalf("wrong block range; expected %v:%v, have %v:%v", firstArg, firstArg, first, last)
	}
	// reset keywords for the following tests
	keywordBlocks[chainId]["first"] = 0
	keywordBlocks[chainId]["last"] = math.MaxUint64
}

func TestUtilsConfig_getMdBlockRange(t *testing.T) {
	// prepare components
	// create new leveldb
	var (
		logLevel   = "INFO"
		firstBlock = uint64(4564026)
		lastBlock  = uint64(20001704)
		firstEpoch = uint64(100)
		lastEpoch  = uint64(200)
		chainId    = MainnetChainID
	)
	log := logger.NewLogger(logLevel, "Test-Log")
	testDb, err := rawdb.NewLevelDBDatabase("./test.db", 1024, 100, "test-db", false)
	if err != nil {
		t.Fatalf("cannot open patch db; %v", err)
	}
	defer os.RemoveAll("./test.db")
	// create fake metadata
	err = ProcessPatchLikeMetadata(testDb, logLevel, firstBlock, lastBlock, firstEpoch, lastEpoch, chainId, true, nil)
	if err != nil {
		t.Fatalf("cannot create a metadata; %v", err)
	}
	err = testDb.Close()
	if err != nil {
		t.Fatalf("cannot close db; %v", err)
	}

	// test getMdBlockRange
	// getMdBlockRange returns default values if unble to open
	first, last, lastpatch, ok, err := getMdBlockRange("./test-wrong.db", MainnetChainID, log)
	if ok || first != uint64(0) || last != math.MaxUint64 {
		t.Fatalf("wrong block range; expected %v:%v, have %v:%v", 0, uint64(math.MaxUint64), first, last)
	} else if err != nil {
		t.Fatalf("unexpected error; %v", err)
	} else if lastpatch != uint64(0) {
		t.Fatalf("wrong last patch block; expected %v, have %v", 0, lastpatch)
	}

	// open an existing AidaDb
	setAidaDbRepositoryUrl(chainId)
	first, last, lastpatch, ok, err = getMdBlockRange("./test.db", MainnetChainID, log)
	if !ok || first != firstBlock || last != lastBlock {
		t.Fatalf("wrong block range; expected %v:%v, have %v:%v", firstBlock, lastBlock, first, last)
	} else if err != nil {
		t.Fatalf("unexpected error; %v", err)
	} else if lastpatch != uint64(45640256) {
		t.Fatalf("wrong last patch block; expected %v, have %v", 45640256, lastpatch)
	}

	// aida url is not set; expected lastpatch is 0.
	AidaDbRepositoryUrl = ""
	first, last, lastpatch, ok, err = getMdBlockRange("./test.db", MainnetChainID, log)
	if !ok || first != firstBlock || last != lastBlock {
		t.Fatalf("wrong block range; expected %v:%v, have %v:%v", firstBlock, lastBlock, first, last)
	} else if err != nil {
		t.Fatalf("unexpected error; %v", err)
	} else if lastpatch != uint64(0) {
		t.Fatalf("wrong last patch block; expected %v, have %v", 0, lastpatch)
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

// TestUtilsConfig_getChainIdFromDB tests if chainID can be loaded from AidaDB correctly
func TestUtilsConfig_getChainIdFromDB(t *testing.T) {
	// prepare components
	// create new leveldb
	var (
		logLevel         = "INFO"
		firstBlock       = uint64(4564026)
		lastBlock        = uint64(20001704)
		firstEpoch       = uint64(100)
		lastEpoch        = uint64(200)
		chainId          = MainnetChainID
		extractedChainId = UnknownChainID
	)

	testDb, err := rawdb.NewLevelDBDatabase("./test.db", 1024, 100, "test-db", false)
	if err != nil {
		t.Fatalf("cannot open patch db; %v", err)
	}
	defer func() {
		err := os.RemoveAll("./test.db")
		if err != nil {

		}
	}()

	// create fake metadata
	err = ProcessPatchLikeMetadata(testDb, logLevel, firstBlock, lastBlock, firstEpoch, lastEpoch, chainId, true, nil)
	if err != nil {
		t.Fatalf("cannot create a metadata; %v", err)
	}
	err = testDb.Close()
	if err != nil {
		t.Fatalf("cannot close db; %v", err)
	}

	// prepare mock config
	cfg := &Config{AidaDb: "./test.db", LogLevel: "info"}

	// prepare logger
	log := logger.NewLogger(cfg.LogLevel, "Utils_config_test")

	// test getChainId function
	extractedChainId, err = getChainId(cfg, log)
	if err != nil {
		t.Fatalf("cannot get chain ID; %v", err)
	}

	if extractedChainId != chainId {
		t.Fatalf("failed to get chainId correctly from AidaDB; Is: %v; Should be: %v", extractedChainId, chainId)
	}
}

// TestUtilsConfig_getDefaultChainId tests if unknown chainID will be replaced with the mainnet chainID
func TestUtilsConfig_getDefaultChainId(t *testing.T) {
	// prepare components
	var (
		err              error
		chainId          = MainnetChainID
		extractedChainId = UnknownChainID
	)

	// prepare mock config
	cfg := &Config{AidaDb: "./test.db", LogLevel: "info"}

	// prepare logger
	log := logger.NewLogger(cfg.LogLevel, "Utils_config_test")

	// test getChainId function
	extractedChainId, err = getChainId(cfg, log)
	if err != nil {
		t.Fatalf("cannot get chain ID; %v", err)
	}

	if extractedChainId != chainId {
		t.Fatalf("failed to get chainId correctly from AidaDB; Is: %v; Should be: %v", extractedChainId, chainId)
	}
}

// TestUtilsConfig_parseCmdArgsBlockRange tests correct parsing of cli arguments for block range
func TestUtilsConfig_parseCmdArgsBlockRange(t *testing.T) {
	// prepare components
	var (
		mode     = BlockRangeArgs
		firstArg = "15"
		lastArg  = "30"
	)

	// parse cli arguments slice
	first, last, _, err := parseCmdArgs([]string{firstArg, lastArg}, mode)
	if err != nil {
		t.Fatalf("cannot parse the cli arguments; %v", err)
	}

	// check if the arguments were parsed correctly
	if parsedFirst, _ := strconv.ParseUint(firstArg, 10, 64); parsedFirst != first {
		t.Fatalf("failed to get first argument correctly; Is: %d; Should be: %s", parsedFirst, firstArg)
	}

	if parsedLast, _ := strconv.ParseUint(lastArg, 10, 64); parsedLast != last {
		t.Fatalf("failed to get last argument correctly; Is: %d; Should be: %s", parsedLast, lastArg)
	}
}

// TestUtilsConfig_parseCmdArgsBlockRangeInvalid tests parsing of invalid cli arguments length for block range
func TestUtilsConfig_parseCmdArgsBlockRangeInvalid(t *testing.T) {
	// prepare components
	var (
		mode = BlockRangeArgs
	)

	// parse cli arguments slice of insufficient length
	_, _, _, err := parseCmdArgs([]string{"test"}, mode)
	if err == nil {
		t.Fatalf("failed to throw an error")
	}
}

// TestUtilsConfig_parseCmdArgsBlockRangeProfileDb tests correct parsing of cli arguments for block range
// and profiling DB
func TestUtilsConfig_parseCmdArgsBlockRangeProfileDb(t *testing.T) {
	// prepare components
	var (
		mode         = BlockRangeArgsProfileDB
		firstArg     = "15"
		lastArg      = "30"
		profileDbArg = "./test.db"
	)

	// parse cli arguments slice
	first, last, profileDb, err := parseCmdArgs([]string{firstArg, lastArg, profileDbArg}, mode)
	if err != nil {
		t.Fatalf("cannot parse the cli arguments; %v", err)
	}

	// check if the arguments were parsed correctly
	if parsedFirst, _ := strconv.ParseUint(firstArg, 10, 64); parsedFirst != first {
		t.Fatalf("failed to get first argument correctly; Is: %d; Should be: %s", parsedFirst, firstArg)
	}

	if parsedLast, _ := strconv.ParseUint(lastArg, 10, 64); parsedLast != last {
		t.Fatalf("failed to get last argument correctly; Is: %d; Should be: %s", parsedLast, lastArg)
	}

	if profileDbArg != profileDb {
		t.Fatalf("failed to get last argument correctly; Is: %s; Should be: %s", profileDb, profileDbArg)
	}
}

// TestUtilsConfig_parseCmdArgsBlockRangeProfileDbInvalid tests parsing of invalid cli arguments length for block range
// and profiling DB
func TestUtilsConfig_parseCmdArgsBlockRangeProfileDbInvalid(t *testing.T) {
	// prepare components
	var (
		mode = BlockRangeArgsProfileDB
	)

	// parse cli arguments slice of insufficient length
	_, _, _, err := parseCmdArgs([]string{"test"}, mode)
	if err == nil {
		t.Fatalf("failed to throw an error")
	}

	// second try with length bigger than 3
	_, _, _, err = parseCmdArgs([]string{"test", "test", "test", "test"}, mode)
	if err == nil {
		t.Fatalf("failed to throw an error")
	}
}

// TestUtilsConfig_parseCmdArgsLastBlock tests correct parsing of cli argument for last block number
func TestUtilsConfig_parseCmdArgsLastBlock(t *testing.T) {
	// prepare components
	var (
		mode    = LastBlockArg
		lastArg = "30"
	)

	// parse cli arguments slice
	_, last, _, err := parseCmdArgs([]string{lastArg}, mode)
	if err != nil {
		t.Fatalf("cannot parse the cli arguments; %v", err)
	}

	// check if the arguments were parsed correctly
	if parsedLast, _ := strconv.ParseUint(lastArg, 10, 64); parsedLast != last {
		t.Fatalf("failed to get last argument correctly; Is: %d; Should be: %s", parsedLast, lastArg)
	}
}

// TestUtilsConfig_parseCmdArgsLastBlockInvalid tests parsing of invalid cli arguments length for last block number
func TestUtilsConfig_parseCmdArgsLastBlockInvalid(t *testing.T) {
	// prepare components
	var (
		mode = LastBlockArg
	)

	// parse cli arguments slice of insufficient length
	_, _, _, err := parseCmdArgs([]string{"test"}, mode)
	if err == nil {
		t.Fatalf("failed to throw an error")
	}
}

// TestUtilsConfig_parseCmdArgsOneToNInvalid tests parsing of invalid cli arguments length for last block number
func TestUtilsConfig_parseCmdArgsOneToNInvalid(t *testing.T) {
	// prepare components
	var (
		mode = OneToNArgs
	)

	// parse cli arguments slice of insufficient length
	_, _, _, err := parseCmdArgs([]string{}, mode)
	if err == nil {
		t.Fatalf("failed to throw an error")
	}
}
