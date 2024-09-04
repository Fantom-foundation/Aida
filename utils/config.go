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
	"errors"
	"fmt"
	"math"
	"math/big"
	"math/rand"
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Substate/db"
	_ "github.com/Fantom-foundation/Tosca/go/geth_adapter"
	_ "github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
	"github.com/urfave/cli/v2"
)

type ArgumentMode int
type ChainID int
type ChainIDs map[ChainID]string
type EthTestType int

const (
	Unknown EthTestType = iota
	StateTests
	BlockTests
)

// An enums of argument modes used by trace subcommands
const (
	BlockRangeArgs ArgumentMode = iota // requires 2 arguments: first block and last block
	LastBlockArg                       // requires 1 argument: last block
	NoArgs                             // requires no arguments
	OneToNArgs                         // requires at least one argument, but accepts up to N
	PathArg                            // requires 1 argument: path to file
)

const (
	UnknownChainID  ChainID = 0
	EthereumChainID ChainID = 1
	MainnetChainID  ChainID = 250
	TestnetChainID  ChainID = 4002
	// EthTestsChainID is a mock ChainID which is necessary for setting
	// the chain rules to allow any block number for any fork.
	EthTestsChainID ChainID = 1337
)

var RealChainIDs = ChainIDs{MainnetChainID: "mainnet", TestnetChainID: "testnet", EthereumChainID: "ethereum"}
var AllowedChainIDs = ChainIDs{MainnetChainID: "mainnet", TestnetChainID: "testnet", EthereumChainID: "ethereum", EthTestsChainID: "eth-tests"}

const (
	AidaDbRepositoryMainnetUrl  = "https://storage.googleapis.com/aida-repository-public/mainnet/aida-patches"
	AidaDbRepositoryTestnetUrl  = "https://storage.googleapis.com/aida-repository-public/testnet/aida-patches"
	AidaDbRepositoryEthereumUrl = "https://storage.googleapis.com/aida-repository-public/ethereum/aida-patches"
)

const maxLastBlock = math.MaxUint64 - 1 // we decrease the value by one because params are always +1

var (
	FirstOperaBlock     uint64 // id of the first block in substate
	AidaDbRepositoryUrl string // url of the Aida DB repository
)

// Type of validation performs on stateDB during Tx processing.
type ValidationMode int

const (
	SubsetCheck   ValidationMode = iota // confirms whether a substate is contained in stateDB.
	EqualityCheck                       // confirms whether a substate and StateDB are identical.
)

const ethTestCmdName = "ethereum-test"

// A map of key blocks on Fantom chain
var KeywordBlocks = map[ChainID]map[string]uint64{
	MainnetChainID: {
		"zero":        0,
		"opera":       4_564_026,
		"istanbul":    0, // todo istanbul block for mainnet?
		"muirglacier": 0, // todo muirglacier block for mainnet?
		"berlin":      37_455_223,
		"london":      37_534_833,
		"first":       0,
		"last":        maxLastBlock,
		"lastpatch":   0,
	},
	TestnetChainID: {
		"zero":        0,
		"opera":       479_327,
		"istanbul":    0, // todo istanbul block for testnet?
		"muirglacier": 0, // todo muirglacier block for testnet?
		"berlin":      1_559_470,
		"london":      7_513_335,
		"first":       0,
		"last":        maxLastBlock,
		"lastpatch":   0,
	},
	// ethereum fork blocks are not stored in this structure as ethereum has already prepared config
	// at params.MainnetChainConfig and it has bigger amount of forks than Fantom chain
	EthereumChainID: {
		"zero":        0,
		"opera":       0,
		"istanbul":    9_069_000,
		"muirglacier": 9_200_000,
		"berlin":      12_244_000,
		"london":      12_965_000,
		"first":       0,
		"last":        maxLastBlock,
		"lastpatch":   0,
	},

	// EthTest must always set its fork blocks to 0 because each test has random block number
	// and if that block number is not greater than the config, the test won't get executed
	EthTestsChainID: {},
}

// special transaction number for pseudo transactions
const PseudoTx = 99999

// GitCommit represents the GitHub commit hash the app was built from.
var GitCommit = "0000000000000000000000000000000000000000"

// Config represents execution configuration for Aida tools.
type Config struct {
	AppName     string
	CommandName string

	First uint64 // first block
	Last  uint64 // last block

	AidaDb                   string  // directory to profiling database containing substate, update, delete accounts data
	ArchiveMaxQueryAge       int     // the maximum age for archive queries (in blocks)
	ArchiveMode              bool    // enable archive mode
	ArchiveQueryRate         int     // the queries per second send to the archive
	ArchiveVariant           string  // selects the implementation variant of the archive
	ArgPath                  string  // path to file or directory given as argument
	BalanceRange             int64   // balance range for stochastic simulation/replay
	BasicBlockProfiling      bool    // enable profiling of basic block
	BlockLength              uint64  // length of a block in number of transactions
	CPUProfile               string  // pprof cpu profile output file name
	CPUProfilePerInterval    bool    // a different CPU profile is taken per 100k block interval
	Cache                    int     // Cache for StateDb or Priming
	CarmenCheckpointInterval int     // how often (in blocks) will Carmen create checkpoints
	CarmenCheckpointPeriod   int     // how often (in minutes) will Carmen create checkpoints
	CarmenSchema             int     // the current DB schema ID to use in Carmen
	CarmenStateCacheSize     int     // the number of values cached in the Carmen StateDB (0 for default value)
	CarmenNodeCacheSize      int     // the size of the in-memory cache to be used by a Carmen LiveDB in byte (0 for default value)
	ChainID                  ChainID // Blockchain ID (mainnet: 250/testnet: 4002)
	ChainCfg                 *params.ChainConfig
	ChannelBufferSize        int    // set a buffer size for profiling channel
	CompactDb                bool   // compact database after merging
	ContinueOnFailure        bool   // continue validation when an error detected
	ContractNumber           int64  // number of contracts to create
	CustomDbName             string // name of state-db directory
	DbComponent              string // options for util-db info are 'all', 'substate', 'delete', 'update', 'state-hash'
	DbImpl                   string // storage implementation
	DbLogging                string // set to true if all DB operations should be logged
	DbTmp                    string // path to temporary database
	DbVariant                string // database variant
	Debug                    bool   // enable trace debug flag
	DebugFrom                uint64 // the first block to print trace debug
	DeleteSourceDbs          bool   // delete source databases
	DeletionDb               string // directory of deleted account database
	DiagnosticServer         int64  // if not zero, the port used for hosting a HTTP server for performance diagnostics
	ErrorLogging             string // if defined, error logging to file is enabled
	EvmImpl                  string
	Genesis                  string         // genesis file
	EthTestType              EthTestType    // which geth test are we running
	IncludeStorage           bool           // represents a flag for contract storage inclusion in an operation
	IsExistingStateDb        bool           // this is true if we are using an existing StateDb
	KeepDb                   bool           // set to true if db is kept after run
	KeysNumber               int64          // number of keys to generate
	LogLevel                 string         // level of the logging of the app action
	MaxNumErrors             int            // maximum number of errors when ContinueOnFailure is enabled
	MaxNumTransactions       int            // the maximum number of processed transactions
	MemoryBreakdown          bool           // enable printing of memory breakdown
	MemoryProfile            string         // capture the memory heap profile into the file
	MicroProfiling           bool           // enable micro-profiling of EVM
	NoHeartbeatLogging       bool           // disables heartbeat logging
	NonceRange               int            // nonce range for stochastic simulation/replay
	OnlySuccessful           bool           // only runs transactions that have been successful
	OperaBinary              string         // path to opera binary
	OperaDb                  string         // path to opera database
	Output                   string         // output directory for aida-db patches or path to events.json file in stochastic generation
	OverwriteRunId           string         // when registering runs, use provided id instead of the autogenerated run id
	PathToStateDb            string         // Path to a working state-db directory
	PrimeRandom              bool           // enable randomized priming
	PrimeThreshold           int            // set account threshold before commit
	Profile                  bool           // enable micro profiling
	ProfileBlocks            bool           // enables block profiler extension
	ProfileDB                string         // profile db for parallel transaction execution
	ProfileDepth             int            // 0 = Interval, 1 = Interval+Block, 2 = Interval+Block+Tx
	ProfileEVMCall           bool           // enable profiling for EVM call
	ProfileFile              string         // output file containing profiling result
	ProfileInterval          uint64         // interval of printing profile result
	ProfileSqlite3           string         // output profiling results to sqlite3 DB
	ProfilingDbName          string         // set a database name for storing micro-profiling results
	RandomSeed               int64          // set random seed for stochastic testing
	RegisterRun              string         // register run to the provided connection string
	RpcRecordingPath         string         // path to source file (or dir with files) with recorded RPC requests
	ShadowDb                 bool           // defines we want to open an existing db as shadow
	ShadowImpl               string         // implementation of the shadow DB to use, empty if disabled
	ShadowVariant            string         // database variant of the shadow DB to be used
	SkipMetadata             bool           // skip metadata insert/getting into AidaDb
	SkipPriming              bool           // skip priming of the state DB
	SkipStateHashScrapping   bool           // if enabled, then state-hashes are not loaded from rpc
	SnapshotDepth            int            // depth of snapshot history
	StateDbSrc               string         // directory to load an existing State DB data
	StateDbSrcDirectAccess   bool           // if true, read and write directly from the source database
	StateDbSrcReadOnly       bool           // if true, source database is not modified
	StateValidationMode      ValidationMode // state validation mode
	SubstateDb               string         // substate directory
	SyncPeriodLength         uint64         // length of a sync-period in number of blocks
	TargetDb                 string         // represents the path of a target DB
	TargetEpoch              uint64         // represents the ID of target epoch to be reached by autogen patch generator
	Trace                    bool           // trace flag
	TraceDirectory           string         // name of trace directory
	TraceFile                string         // name of trace file
	TrackProgress            bool           // enables track progress logging
	TrackerGranularity       int            // defines how often will tracker report achieved block
	TransactionLength        uint64         // determines indirectly the length of a transaction
	UpdateBufferSize         uint64         // cache size in Bytes
	UpdateDb                 string         // update-set directory
	UpdateOnFailure          bool           // if enabled and continue-on-failure is also enabled, this updates any error found in StateDb
	UpdateType               string         // download datatype
	Validate                 bool           // validate validate aida-db
	ValidateStateHashes      bool           // if this is true state hash validation is enabled in Executor
	ValidateTxState          bool           // validate stateDB before and after transaction
	ValuesNumber             int64          // number of values to generate
	VmImpl                   string         // vm implementation (geth/lfvm)
	Workers                  int            // number of worker threads
	TxGeneratorType          []string       // type of the application used for transaction generation
	Forks                    []string       // Which forks are going to get executed byz
}

type configContext struct {
	cfg         *Config       // run configuration
	log         logger.Logger // logger for printing logs in config functions
	hasMetadata bool          // if true, Aida-db has a valid metadata table
	ctx         *cli.Context  // command line context for accessing flags and command line arguments
}

func NewConfigContext(cfg *Config, ctx *cli.Context) *configContext {
	return &configContext{
		log:         logger.NewLogger(cfg.LogLevel, "Config"),
		cfg:         cfg,
		hasMetadata: false,
		ctx:         ctx,
	}
}

// NewTestConfig creates a new config for test purpose
func NewTestConfig(t *testing.T, chainId ChainID, first, last uint64, validate bool) *Config {
	chainCfg, err := GetChainConfig(chainId)
	if err != nil {
		t.Fatalf("cannot get chain cfg: %v", err)
	}
	return &Config{
		ChainID:         chainId,
		First:           first,
		Last:            last,
		ChainCfg:        chainCfg,
		LogLevel:        "Critical",
		SkipPriming:     true,
		Validate:        validate,
		ValidateTxState: validate,
	}
}

// NewConfig creates and initializes Config with commandline arguments.
func NewConfig(ctx *cli.Context, mode ArgumentMode) (*Config, error) {
	var err error

	// create config with user flag values, if not set default values are used
	cfg := createConfigFromFlags(ctx)

	// create config context for sharing common arguments
	cc := NewConfigContext(cfg, ctx)

	// check if chainID is set correctly
	err = cc.setChainId()
	if err != nil {
		return nil, fmt.Errorf("cannot get chain id; %v", err)
	}

	err = cc.setChainConfig()
	if err != nil {
		return nil, fmt.Errorf("cannot set chain id: %w", err)
	}

	// set first Opera block according to chian id
	err = cc.setFirstOperaBlock()
	if err != nil {
		return nil, err
	}

	// set aida db repository url
	err = cc.setAidaDbRepositoryUrl()
	if err != nil {
		return cfg, fmt.Errorf("unable to prepare url from chain id %v; %v", cfg.ChainID, err)
	}

	// set numbers of first block, last block and path to profilingDB
	err = cc.updateConfigBlockRange(ctx.Args().Slice(), mode)
	if err != nil {
		return cfg, fmt.Errorf("unable to parse cli arguments; %v", err)
	}

	err = cc.adjustMissingConfigValues()
	if err != nil {
		return nil, fmt.Errorf("cannot adjust missing config values; %v", err)
	}

	cc.reportNewConfig()

	return cfg, nil
}

func (cc *configContext) setFirstOperaBlock() error {
	if _, ok := AllowedChainIDs[cc.cfg.ChainID]; !ok {
		return fmt.Errorf("unknown chain id %v", cc.cfg.ChainID)
	}
	FirstOperaBlock = KeywordBlocks[cc.cfg.ChainID]["opera"]
	return nil
}

// setAidaDbRepositoryUrl based on chain id selects correct aida-db repository url
func (cc *configContext) setAidaDbRepositoryUrl() error {
	switch cc.cfg.ChainID {
	case MainnetChainID:
		AidaDbRepositoryUrl = AidaDbRepositoryMainnetUrl
	case TestnetChainID:
		AidaDbRepositoryUrl = AidaDbRepositoryTestnetUrl
	case EthereumChainID:
		AidaDbRepositoryUrl = AidaDbRepositoryEthereumUrl
	default:
		cc.log.Warningf("%v chain-id does not have aida-db repository url set - setting to mainnet", cc.cfg)
		AidaDbRepositoryUrl = AidaDbRepositoryMainnetUrl
	}

	return nil
}

func (cfg *Config) SetStateDbSrcReadOnly() {
	cfg.StateDbSrcDirectAccess = true
	cfg.StateDbSrcReadOnly = true
}

// GetChainConfig returns chain configuration of either mainnet or testnets.
func GetChainConfig(chainId ChainID) (*params.ChainConfig, error) {
	if _, ok := AllowedChainIDs[chainId]; !ok {
		return nil, fmt.Errorf("unknown chain id %v\nallowed chain-ids: %v", chainId, AllowedChainIDs)
	}
	switch chainId {
	case EthereumChainID:
		chainConfig := params.MainnetChainConfig
		chainConfig.DAOForkSupport = false
		return chainConfig, nil
	case EthTestsChainID:
		return params.AllDevChainProtocolChanges, nil
	default:
		// Make a copy of the basic config before modifying it to avoid
		// unexpected side-effects and synchronization issues in parallel runs.
		chainConfig := *params.AllEthashProtocolChanges
		chainConfig.ChainID = big.NewInt(int64(chainId))

		chainConfig.BerlinBlock = new(big.Int).SetUint64(KeywordBlocks[chainId]["berlin"])
		chainConfig.LondonBlock = new(big.Int).SetUint64(KeywordBlocks[chainId]["london"])
		return &chainConfig, nil
	}
}

// directoryExists returns true if a directory exists
func directoryExists(path string) bool {
	if _, err := os.Stat(path); err != nil {
		return false
	}
	return true

}

// SetBlockRange checks the validity of a block range and return the first and last block as numbers.
func SetBlockRange(firstArg string, lastArg string, chainId ChainID) (uint64, uint64, error) {
	var err error = nil
	first, ferr := strconv.ParseUint(firstArg, 10, 64)
	last, lerr := strconv.ParseUint(lastArg, 10, 64)

	if ferr != nil {
		first, err = setBlockNumber(firstArg, chainId)
		if err != nil {
			return 0, 0, err
		}
	}

	if lerr != nil {
		last, err = setBlockNumber(lastArg, chainId)
		if err != nil {
			return 0, 0, err
		}
	}

	if first > last {
		return 0, 0, fmt.Errorf("first block %v has larger number than last block %v", first, last)
	}

	return first, last, err
}

// setBlockNumber parse the command line argument (number, hardfork keyword or keyword with offset)
// returns calculated block number
func setBlockNumber(arg string, chainId ChainID) (uint64, error) {
	var blkNum uint64
	var hasOffset bool
	var keyword string
	var symbol string
	var offset uint64

	// check if keyword has an offset and extract the keyword, offset direction (arithmetical symbol) and offset value
	re := regexp.MustCompile(`^[a-zA-Z]+\w*[+-]\d+$`)
	if hasOffset = re.MatchString(arg); hasOffset {
		var err error
		if keyword, symbol, offset, err = parseOffset(arg); err != nil {
			return 0, err
		}
	} else {
		keyword = strings.ToLower(arg)
	}
	// find base block number from keyword
	if val, ok := KeywordBlocks[chainId][keyword]; ok {
		blkNum = val
	} else {
		return 0, fmt.Errorf("block number not a valid keyword or integer")
	}

	// shift base block number by the offset
	if hasOffset {
		blkNum = offsetBlockNum(blkNum, symbol, offset)
	}

	return blkNum, nil
}

// parseOffset parse the hardfork keyword, offset value and a direction of the offset
func parseOffset(arg string) (string, string, uint64, error) {
	if strings.Contains(arg, "+") {
		if keyword, offset, ok := splitKeywordOffset(arg, "+"); ok {
			return strings.ToLower(keyword), "+", offset, nil
		}

		return "", "", 0, fmt.Errorf("block number not a valid keyword with offset")
	} else if strings.Contains(arg, "-") {
		if keyword, offset, ok := splitKeywordOffset(arg, "-"); ok {
			return strings.ToLower(keyword), "-", offset, nil
		}

		return "", "", 0, fmt.Errorf("block number not a valid keyword with offset")
	}

	return "", "", 0, fmt.Errorf("block number has invalid arithmetical sign")
}

// splitKeywordOffset split the hardfork keyword and the arithmetical sign determining the direction of the offset
func splitKeywordOffset(arg string, symbol string) (string, uint64, bool) {
	res := strings.Split(arg, symbol)

	// if the keyword doesn't exist, return.
	if _, ok := KeywordBlocks[MainnetChainID][strings.ToLower(res[0])]; !ok {
		return "", 0, false
	}

	offset, err := strconv.ParseUint(res[1], 10, 64)
	if err != nil {
		return "", 0, false
	}

	return res[0], offset, true
}

// offsetBlockNum adds/subtracts the offset to/from block number
func offsetBlockNum(blkNum uint64, symbol string, offset uint64) uint64 {
	res := uint64(0)
	if symbol == "+" {
		res = blkNum + offset
	} else if symbol == "-" {
		res = blkNum - offset
	}

	return res
}

// getMdBlockRange gets block range from aidaDB metadata
func (cc *configContext) getMdBlockRange() (uint64, uint64, uint64, error) {
	defaultFirst := KeywordBlocks[cc.cfg.ChainID]["first"]
	defaultLast := KeywordBlocks[cc.cfg.ChainID]["last"]
	defaultLastPatch := KeywordBlocks[cc.cfg.ChainID]["lastpatch"]

	if !directoryExists(cc.cfg.AidaDb) {
		cc.log.Warningf("Unable to open Aida-db in %s", cc.cfg.AidaDb)
		return defaultFirst, defaultLast, defaultLastPatch, nil
	}

	// read meta data
	aidaDb, err := db.NewReadOnlyBaseDB(cc.cfg.AidaDb)
	if err != nil {
		cc.log.Warningf("Cannot open AidaDB; %v", err)
		return defaultFirst, defaultLast, defaultLastPatch, nil
	}
	md := NewAidaDbMetadata(aidaDb, cc.cfg.LogLevel)
	err = md.getBlockRange()
	if err != nil {
		cc.log.Warning(err)
		return defaultFirst, defaultLast, defaultLastPatch, nil
	}
	cc.hasMetadata = true
	lastPatchBlock, err := getPatchFirstBlock(md.LastBlock)
	if err != nil {
		cc.log.Warningf("Cannot get first block of the last patch of given AidaDB; %v", err)
	}

	err = aidaDb.Close()
	if err != nil {
		return defaultFirst, defaultLast, defaultLastPatch, fmt.Errorf("cannot close db; %v", err)
	}

	return md.FirstBlock, md.LastBlock, lastPatchBlock, nil
}

// adjustBlockRange finds overlap between metadata block range and block range specified by user in command line
func (cc *configContext) adjustBlockRange(firstArg, lastArg uint64) (uint64, uint64, error) {
	var first, last, firstMd, lastMd uint64
	firstMd = KeywordBlocks[cc.cfg.ChainID]["first"]
	lastMd = KeywordBlocks[cc.cfg.ChainID]["last"]

	if lastArg >= firstMd && lastMd >= firstArg {
		// get first block number
		if firstArg >= firstMd {
			first = firstArg
		} else {
			first = firstMd
			cc.log.Warningf("First block arg (%v) is out of range of AidaDb - adjusted to the first block of AidaDb (%v)", firstArg, firstMd)
		}

		// get last block number
		if lastArg <= lastMd {
			last = lastArg
		} else {
			last = lastMd
			cc.log.Warningf("Last block arg (%v) is out of range of AidaDb - adjusted to the last block of AidaDb (%v)", lastArg, lastMd)
		}

		return first, last, nil
	} else {
		return 0, 0, fmt.Errorf("block range of your AidaDb (%v-%v) cannot execute given block range %v-%v", firstMd, lastMd, firstArg, lastArg)
	}
}

// setChainId set config chainID to the default (mainnet) or user specified chainID
// if the chainID is unknown type, it'll be loaded from aidaDB
func (cc *configContext) setChainId() error {
	// first look for chainId since we need it for verbal block indication
	if cc.cfg.ChainID == UnknownChainID {
		cc.log.Warningf("ChainID (--%v) was not set; looking for it in AidaDb", ChainIDFlag.Name)

		// we check if AidaDb was set with err == nil
		if aidaDb, err := db.OpenBaseDB(cc.cfg.AidaDb); err == nil {
			md := NewAidaDbMetadata(aidaDb, cc.cfg.LogLevel)

			cc.cfg.ChainID = md.GetChainID()

			if err = aidaDb.Close(); err != nil {
				return fmt.Errorf("cannot close db; %v", err)
			}
		}

		if cc.cfg.ChainID == 0 {
			if strings.EqualFold(cc.ctx.Command.Name, ethTestCmdName) {
				cc.cfg.ChainID = EthTestsChainID
				return nil
			}
			cc.log.Warningf("ChainID was neither specified with flag (--%v) nor was found in AidaDb (%v); setting default value for mainnet", ChainIDFlag.Name, cc.cfg.AidaDb)
			cc.cfg.ChainID = MainnetChainID
		} else {
			cc.log.Noticef("Found chainId (%v) in AidaDb", cc.cfg.ChainID)
		}
	}
	return nil
}

// updateConfigBlockRange parse the command line arguments according to the mode in which selected tool runs
// and store them into the config
func (cc *configContext) updateConfigBlockRange(args []string, mode ArgumentMode) error {
	var (
		first uint64
		last  uint64
	)

	switch mode {
	case BlockRangeArgs:
		// process arguments and flags
		if len(args) >= 2 {
			// try to extract block range from db metadata
			firstMd, lastMd, lastPatchMd, err := cc.getMdBlockRange()
			if err != nil {
				return err
			}
			KeywordBlocks[cc.cfg.ChainID]["first"] = firstMd
			KeywordBlocks[cc.cfg.ChainID]["last"] = lastMd
			KeywordBlocks[cc.cfg.ChainID]["lastpatch"] = lastPatchMd

			// try to parse and check block range
			firstArg, lastArg, argErr := SetBlockRange(args[0], args[1], cc.cfg.ChainID)
			if argErr != nil {
				return argErr
			}

			if !cc.hasMetadata {
				first = firstArg
				last = lastArg
				break
			}

			// find if values overlap
			first, last, err = cc.adjustBlockRange(firstArg, lastArg)
			if err != nil {
				return err
			}
		} else {
			return fmt.Errorf("command requires 2 arguments")
		}
	case LastBlockArg:
		var err error

		last, err = strconv.ParseUint(args[0], 10, 64)
		if err != nil {
			return err
		}
	case OneToNArgs:
		if len(args) < 1 {
			return errors.New("this command requires at least 1 argument")
		}
	case NoArgs:
	case PathArg:
		if len(args) != 1 {
			return fmt.Errorf("path argument (%v) is required to run this command", args[0])
		}

		_, err := os.Stat(args[0])
		if err != nil {
			if os.IsNotExist(err) {
				return fmt.Errorf("given path (%v) argument does not exist", args[0])
			}
			return fmt.Errorf("cannot read argument path (%v)", err)
		}

		cc.cfg.ArgPath = args[0]
	default:
		return errors.New("unknown mode; unable to process commandline arguments")
	}

	cc.cfg.First = first
	cc.cfg.Last = last
	return nil
}

// adjustMissingConfigValues fill the missing values in the config
func (cc *configContext) adjustMissingConfigValues() error {
	cfg := cc.cfg
	log := cc.log

	// set default db variant if not provided.
	if cfg.DbImpl == "carmen" && cfg.DbVariant == "" {
		cfg.DbVariant = "go-file"
		log.Info("set a DB variant to go-file")
	}

	// if ErrorLogging is set we expect we want to catch all processing errors hence we enable ContinueOnFailure
	if cfg.ErrorLogging != "" {
		cfg.ContinueOnFailure = true
		log.Warning("Enable continue-on-failure mode because error logging is used.")
	}

	// --continue-on-failure implicitly enables transaction validation
	cfg.ValidateTxState = cfg.Validate || cfg.ValidateTxState || cfg.ContinueOnFailure
	cfg.ValidateStateHashes = cfg.Validate || cfg.ValidateStateHashes

	if cfg.RandomSeed < 0 {
		cfg.RandomSeed = int64(rand.Uint32())
	}

	// if AidaDB path is given, redirect source path to AidaDB.
	if found := directoryExists(cfg.AidaDb); found {
		OverwriteDbPathsByAidaDb(cfg)
	}

	// in-memory StateDB cannot be kept after run.
	if cfg.KeepDb && strings.Contains(cfg.DbVariant, "memory") {
		cfg.KeepDb = false
		log.Warning("Keep DB feature is disabled because in-memory storage is used.")
	}

	// if path doesn't exist, use system temp directory.
	if found := directoryExists(cfg.DbTmp); !found {
		cc.log.Warningf("Temporary directory %v is not found. Use the system default %v.", cfg.DbTmp, os.TempDir())
		cfg.DbTmp = os.TempDir()
	}
	return nil
}

// OverwriteDbPathsByAidaDb overwrites the paths of the DBs by the AidaDb path
func OverwriteDbPathsByAidaDb(cfg *Config) {
	cfg.UpdateDb = cfg.AidaDb
	cfg.DeletionDb = cfg.AidaDb
	cfg.SubstateDb = cfg.AidaDb
}

// reportNewConfig logs out the state of config in current run
func (cc *configContext) reportNewConfig() {
	cfg := cc.cfg
	log := cc.log

	log.Noticef("Run config:")
	log.Infof("Block range: %v to %v", cfg.First, cfg.Last)
	if cfg.MaxNumTransactions >= 0 {
		log.Noticef("Transaction limit: %d", cfg.MaxNumTransactions)
	}
	log.Infof("Chain id: %v (record & run-vm only)", cfg.ChainID)
	log.Infof("SyncPeriod length: %v", cfg.SyncPeriodLength)
	log.Noticef("Used EVM implementation: %v", cfg.VmImpl)
	log.Noticef("Used VM implementation: %v", cfg.VmImpl)
	log.Infof("Aida DB directory: %v", cfg.AidaDb)

	// todo move to tx validator once finished
	log.Infof("validate tx state: %v", cfg.ValidateTxState)
	if cfg.Profile {
		log.Infof("Profiling enabled - at depth: %d", cfg.ProfileDepth)
		if cfg.ProfileFile != "" {
			log.Infof("  Profiling results output file path: %s", cfg.ProfileFile)
		}
		if cfg.ProfileSqlite3 != "" {
			log.Infof("  Profiling results output to sqlite3: %s", cfg.ProfileSqlite3)
		}
	}

	if cfg.RegisterRun != "" {
		log.Infof("Register Run to: %v", cfg.RegisterRun)
	}

	if cfg.ShadowDb {
		log.Warning("DB shadowing enabled, reducing Tx throughput and increasing memory and storage usage")
	}
	if cfg.DbLogging != "" {
		log.Warning("Db logging enabled, reducing Tx throughput")
	}
}

func (cc *configContext) setChainConfig() (err error) {
	cc.cfg.ChainCfg, err = GetChainConfig(cc.cfg.ChainID)
	return err
}
