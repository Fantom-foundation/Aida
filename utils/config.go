// Package trace provides cli for recording and replaying storage traces.
package utils

import (
	"fmt"
	"log"
	"math/big"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"time"

	substate "github.com/Fantom-foundation/Substate"
	_ "github.com/Fantom-foundation/Tosca/go/vm/lfvm"
	_ "github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
	"github.com/urfave/cli/v2"
)

type ArgumentMode int

// An enums of argument modes used by trace subcommands
const (
	BlockRangeArgs ArgumentMode = iota // requires 2 arguments: first block and last block
	LastBlockArg                       // requires 1 argument: last block
	NoArgs                             // requires no arguments
)

var (
	FirstSubstateBlock uint64 // id of the first block in substate
)

// Type of validation performs on stateDB during Tx processing.
type ValidationMode int

const (
	SubsetCheck   ValidationMode = iota // confirms whether a substate is contained in stateDB.
	EqualityCheck                       // confirms whether a substate and StateDB are identical.
)

// GitCommit represents the GitHub commit hash the app was built from.
var GitCommit = "0000000000000000000000000000000000000000"

// Command line options for common flags in record and replay.
var (
	ArchiveModeFlag = cli.BoolFlag{
		Name:  "archive",
		Usage: "set node type to archival mode. If set, the node keep all the EVM state history; otherwise the state history will be pruned.",
	}
	ArchiveVariantFlag = cli.StringFlag{
		Name:  "archive-variant",
		Usage: "set the archive implementation variant for the selected DB implementation, ignored if not running in archive mode",
	}
	BlockLengthFlag = cli.IntFlag{
		Name:  "block-length",
		Usage: "defines the number of transactions per block",
		Value: 10,
	}
	CarmenSchemaFlag = cli.IntFlag{
		Name:  "carmen-schema",
		Usage: "select the DB schema used by Carmen's current state DB",
		Value: 0,
	}
	ChainIDFlag = cli.IntFlag{
		Name:  "chainid",
		Usage: "ChainID for replayer",
		Value: 250,
	}
	ContinueOnFailureFlag = cli.BoolFlag{
		Name:  "continue-on-failure",
		Usage: "continue execute after validation failure detected",
	}
	CpuProfileFlag = cli.StringFlag{
		Name:  "cpuprofile",
		Usage: "enables CPU profiling",
	}
	DebugFromFlag = cli.Uint64Flag{
		Name:  "debug-from",
		Usage: "sets the first block to print trace debug",
		Value: 0,
	}
	DeletionDirFlag = cli.StringFlag{
		Name:  "deletiondir",
		Usage: "sets the directory containing deleted accounts database",
	}
	KeepStateDBFlag = cli.BoolFlag{
		Name:  "keep-db",
		Usage: "if set, statedb is not deleted after run",
	}
	MemProfileFlag = cli.StringFlag{
		Name:  "memprofile",
		Usage: "enables memory allocation profiling",
	}
	SyncPeriodLengthFlag = cli.IntFlag{
		Name:  "sync-period",
		Usage: "defines the number of blocks per sync-period",
		Value: 300, // ~ 300s = 5 minutes
	}
	MemoryBreakdownFlag = cli.BoolFlag{
		Name:  "memory-breakdown",
		Usage: "enables printing of memory usage breakdown",
	}
	ProfileFlag = cli.BoolFlag{
		Name:  "profile",
		Usage: "enables profiling",
	}
	DisableProgressFlag = cli.BoolFlag{
		Name:  "disable-progress",
		Usage: "disable progress report",
	}
	RandomizePrimingFlag = cli.BoolFlag{
		Name:  "prime-random",
		Usage: "randomize order of accounts in StateDB priming",
	}
	PrimeSeedFlag = cli.Int64Flag{
		Name:  "prime-seed",
		Usage: "set seed for randomizing priming",
		Value: time.Now().UnixNano(),
	}
	PrimeThresholdFlag = cli.IntFlag{
		Name:  "prime-threshold",
		Usage: "set number of accounts written to stateDB before applying pending state updates",
		Value: 0,
	}
	RandomSeedFlag = cli.IntFlag{
		Name:  "random-seed",
		Usage: "Set random seed",
		Value: -1,
	}
	SkipPrimingFlag = cli.BoolFlag{
		Name:  "skip-priming",
		Usage: "if set, DB priming should be skipped; most useful with the 'memory' DB implementation",
	}
	StateDbImplementationFlag = cli.StringFlag{
		Name:  "db-impl",
		Usage: "select state DB implementation",
		Value: "geth",
	}
	StateDbVariantFlag = cli.StringFlag{
		Name:  "db-variant",
		Usage: "select a state DB variant",
		Value: "",
	}
	StateDbSrcDirFlag = cli.StringFlag{
		Name:  "db-src-dir",
		Usage: "sets the directory contains source state DB data",
	}
	StateDbTempDirFlag = cli.StringFlag{
		Name:  "db-tmp-dir",
		Usage: "sets the temporary directory where to place state DB data; uses system default if empty",
	}
	StateDbLoggingFlag = cli.BoolFlag{
		Name:  "db-logging",
		Usage: "enable logging of all DB operations",
	}
	ShadowDbImplementationFlag = cli.StringFlag{
		Name:  "db-shadow-impl",
		Usage: "select state DB implementation to shadow the prime DB implementation",
		Value: "",
	}
	ShadowDbVariantFlag = cli.StringFlag{
		Name:  "db-shadow-variant",
		Usage: "select a state DB variant to shadow the prime DB implementation",
		Value: "",
	}
	TraceFlag = cli.BoolFlag{
		Name:  "trace",
		Usage: "enable tracing",
	}
	TraceDebugFlag = cli.BoolFlag{
		Name:  "trace-debug",
		Usage: "enable debug output for tracing",
	}
	TraceDirectoryFlag = cli.StringFlag{
		Name:  "tracedir",
		Usage: "set storage trace's output directory",
		Value: "./",
	}
	UpdateDBDirFlag = cli.StringFlag{
		Name:  "updatedir",
		Usage: "set update-set database directory",
	}
	ValidateFlag = cli.BoolFlag{
		Name:  "validate",
		Usage: "enables validation",
	}
	ValidateTxStateFlag = cli.BoolFlag{
		Name:  "validate-tx",
		Usage: "enables transaction state validation",
	}
	ValidateWorldStateFlag = cli.BoolFlag{
		Name:  "validate-ws",
		Usage: "enables end-state validation",
	}
	VmImplementation = cli.StringFlag{
		Name:  "vm-impl",
		Usage: "select VM implementation",
		Value: "geth",
	}
	WorldStateDirFlag = cli.PathFlag{
		Name:  "worldstatedir",
		Usage: "world state snapshot database path",
	}
	NumberOfBlocksFlag = cli.IntFlag{
		Name:     "number",
		Aliases:  []string{"n"},
		Usage:    "Number of blocks",
		Required: true,
		Value:    0,
	}
	MaxNumTransactionsFlag = cli.IntFlag{
		Name:  "max-transactions",
		Usage: "limit the maximum number of processed transactions, default: unlimited",
		Value: -1,
	}
	OutputFlag = cli.StringFlag{
		Name:  "output",
		Usage: "output filename",
	}
	PortFlag = cli.StringFlag{
		Name:        "port",
		Aliases:     []string{"v"},
		Usage:       "enable visualization on `PORT`",
		DefaultText: "8080",
	}
	DeleteSourceDBsFlag = cli.BoolFlag{
		Name:  "delete-source-dbs",
		Usage: "delete source databases while merging into one database",
		Value: false,
	}
	AidaDBFlag = cli.StringFlag{
		Name:  "aida-db",
		Usage: "set substate, updateset and deleted accounts directory",
	}
	ContractNumberFlag = cli.IntFlag{
		Name:  "num-contracts",
		Usage: "Number of contracts to create",
		Value: 1_000,
	}
	KeysNumberFlag = cli.IntFlag{
		Name:  "num-keys",
		Usage: "Number of keys to generate",
		Value: 1_000,
	}
	ValuesNumberFLag = cli.IntFlag{
		Name:  "num-values",
		Usage: "Number of values to generate",
		Value: 1_000,
	}
	OperationFrequency = cli.IntFlag{
		Name:  "operation-frequency",
		Usage: "Determines indirectly the length of a transaction",
		Value: 10,
	}
	SnapshotDepthFlag = cli.IntFlag{
		Name:  "snapshot-depth",
		Usage: "Depth of snapshot history",
		Value: 100,
	}
	LogLevel = cli.StringFlag{
		Name:    "log",
		Aliases: []string{"l"},
		Usage:   "Level of the logging of the app action (\"critical\", \"error\", \"warning\", \"notice\", \"info\", \"debug\"; default: INFO)",
		Value:   "info",
	}
)

// Config represents execution configuration for replay command.
type Config struct {
	AppName     string
	CommandName string

	First uint64 // first block
	Last  uint64 // last block

	ArchiveMode         bool           // enable archive mode
	ArchiveVariant      string         // selects the implementation variant of the archive
	BlockLength         uint64         // length of a block in number of transactions
	CarmenSchema        int            // the current DB schema ID to use in Carmen
	ChainID             int            // Blockchain ID (mainnet: 250/testnet: 4002)
	ContinueOnFailure   bool           // continue validation when an error detected
	ContractNumber      int64          // number of contracts to create
	CPUProfile          string         // pprof cpu profile output file name
	DbImpl              string         // storage implementation
	DbVariant           string         // database variant
	DbLogging           bool           // set to true if all DB operations should be logged
	Debug               bool           // enable trace debug flag
	DebugFrom           uint64         // the first block to print trace debug
	DeletedAccountDir   string         // directory of deleted account database
	EnableProgress      bool           // enable progress report flag
	SyncPeriodLength    uint64         // length of a sync-period in number of blocks
	HasDeletedAccounts  bool           // true if deletedAccountDir is not empty; otherwise false
	KeepStateDB         bool           // set to true if stateDB is kept after run
	KeysNumber          int64          // number of keys to generate
	MaxNumTransactions  int            // the maximum number of processed transactions
	MemoryBreakdown     bool           // enable printing of memory breakdown
	MemoryProfile       string         // capture the memory heap profile into the file
	OperationFrequency  uint64         // determines indirectly the length of a transaction
	PrimeRandom         bool           // enable randomized priming
	PrimeSeed           int64          // set random seed
	PrimeThreshold      int            // set account threshold before commit
	Profile             bool           // enable micro profiling
	RandomSeed          int64          // set random seed for stochastic testing (TODO: Perhaps combine with PrimeSeed??)
	SkipPriming         bool           // skip priming of the state DB
	ShadowImpl          string         // implementation of the shadow DB to use, empty if disabled
	ShadowVariant       string         // database variant of the shadow DB to be used
	StateDbSrcDir       string         // directory to load an existing State DB data
	DBDir               string         // directory to profiling database containing substate, update, delete accounts data
	StateDbTempDir      string         // directory to store a working copy of State DB data
	StateValidationMode ValidationMode // state validation mode
	UpdateDBDir         string         // update-set directory
	SnapshotDepth       int            // depth of snapshot history
	SubstateDBDir       string         // substate directory
	ValidateTxState     bool           // validate stateDB before and after transaction
	ValidateWorldState  bool           // validate stateDB before and after replay block range
	ValuesNumber        int64          // number of values to generate
	VmImpl              string         // vm implementation (geth/lfvm)
	Workers             int            // number of worker threads
	TraceFile           string         // name of trace file
	Trace               bool           // trace flag
}

// getChainConnfig returns chain configuration of either mainnet or testnets.
func GetChainConfig(chainID int) *params.ChainConfig {
	chainConfig := params.AllEthashProtocolChanges
	chainConfig.ChainID = big.NewInt(int64(chainID))
	if chainID == 250 {
		// mainnet chainID 250
		chainConfig.BerlinBlock = new(big.Int).SetUint64(37455223)
		chainConfig.LondonBlock = new(big.Int).SetUint64(37534833)
	} else if chainID == 4002 {
		// testnet chainID 4002
		chainConfig.BerlinBlock = new(big.Int).SetUint64(1559470)
		chainConfig.LondonBlock = new(big.Int).SetUint64(7513335)
	} else {
		log.Fatalf("unknown chain id %v", chainID)
	}
	return chainConfig
}

func setFirstBlockFromChainID(chainID int) {
	if chainID == 250 {
		FirstSubstateBlock = 4564026
	} else if chainID == 4002 {
		FirstSubstateBlock = 479327
	} else {
		log.Fatalf("unknown chain id %v", chainID)
	}
}

// NewConfig creates and initializes Config with commandline arguments.
func NewConfig(ctx *cli.Context, mode ArgumentMode) (*Config, error) {
	// number of blocks to be generated by Stochastic
	n := ctx.Uint64(NumberOfBlocksFlag.Name)

	var first, last uint64
	if n != 0 {
		first = 1
		last = n
	} else {
		var argErr error
		switch mode {
		case BlockRangeArgs:
			// process arguments and flags
			if ctx.Args().Len() != 2 {
				return nil, fmt.Errorf("trace command requires exactly 2 arguments")
			}
			first, last, argErr = SetBlockRange(ctx.Args().Get(0), ctx.Args().Get(1))
			if argErr != nil {
				return nil, argErr
			}
		case LastBlockArg:
			last, argErr = strconv.ParseUint(ctx.Args().Get(0), 10, 64)
			if argErr != nil {
				return nil, argErr
			}
		case NoArgs:
		default:
			return nil, fmt.Errorf("Unknown mode. Unable to process commandline arguments.")
		}
	}

	// --continue-on-failure implicitly enables transaction state validation
	validateTxState := ctx.Bool(ValidateFlag.Name) ||
		ctx.Bool(ValidateTxStateFlag.Name) ||
		ctx.Bool(ContinueOnFailureFlag.Name)
	validateWorldState := ctx.Bool(ValidateFlag.Name) ||
		ctx.Bool(ValidateWorldStateFlag.Name)

	cfg := &Config{
		AppName:     ctx.App.HelpName,
		CommandName: ctx.Command.Name,

		ArchiveMode:         ctx.Bool(ArchiveModeFlag.Name),
		ArchiveVariant:      ctx.String(ArchiveVariantFlag.Name),
		BlockLength:         ctx.Uint64(BlockLengthFlag.Name),
		CarmenSchema:        ctx.Int(CarmenSchemaFlag.Name),
		ChainID:             ctx.Int(ChainIDFlag.Name),
		ContractNumber:      ctx.Int64(ContractNumberFlag.Name),
		ContinueOnFailure:   ctx.Bool(ContinueOnFailureFlag.Name),
		CPUProfile:          ctx.String(CpuProfileFlag.Name),
		Debug:               ctx.Bool(TraceDebugFlag.Name),
		DebugFrom:           ctx.Uint64(DebugFromFlag.Name),
		EnableProgress:      !ctx.Bool(DisableProgressFlag.Name),
		SyncPeriodLength:    ctx.Uint64(SyncPeriodLengthFlag.Name),
		First:               first,
		DbImpl:              ctx.String(StateDbImplementationFlag.Name),
		DbVariant:           ctx.String(StateDbVariantFlag.Name),
		DbLogging:           ctx.Bool(StateDbLoggingFlag.Name),
		DeletedAccountDir:   ctx.String(DeletionDirFlag.Name),
		HasDeletedAccounts:  true,
		KeepStateDB:         ctx.Bool(KeepStateDBFlag.Name),
		KeysNumber:          ctx.Int64(KeysNumberFlag.Name),
		Last:                last,
		MaxNumTransactions:  ctx.Int(MaxNumTransactionsFlag.Name),
		MemoryBreakdown:     ctx.Bool(MemoryBreakdownFlag.Name),
		MemoryProfile:       ctx.String(MemProfileFlag.Name),
		OperationFrequency:  ctx.Uint64(OperationFrequency.Name),
		PrimeRandom:         ctx.Bool(RandomizePrimingFlag.Name),
		PrimeSeed:           ctx.Int64(PrimeSeedFlag.Name),
		RandomSeed:          ctx.Int64(RandomSeedFlag.Name),
		PrimeThreshold:      ctx.Int(PrimeThresholdFlag.Name),
		Profile:             ctx.Bool(ProfileFlag.Name),
		SkipPriming:         ctx.Bool(SkipPrimingFlag.Name),
		ShadowImpl:          ctx.String(ShadowDbImplementationFlag.Name),
		ShadowVariant:       ctx.String(ShadowDbVariantFlag.Name),
		SnapshotDepth:       ctx.Int(SnapshotDepthFlag.Name),
		StateDbSrcDir:       ctx.String(StateDbSrcDirFlag.Name),
		DBDir:               ctx.String(AidaDBFlag.Name),
		StateDbTempDir:      ctx.String(StateDbTempDirFlag.Name),
		StateValidationMode: EqualityCheck,
		UpdateDBDir:         ctx.String(UpdateDBDirFlag.Name),
		SubstateDBDir:       ctx.String(substate.SubstateDirFlag.Name),
		ValuesNumber:        ctx.Int64(ValuesNumberFLag.Name),
		ValidateTxState:     validateTxState,
		ValidateWorldState:  validateWorldState,
		VmImpl:              ctx.String(VmImplementation.Name),
		Workers:             ctx.Int(substate.WorkersFlag.Name),
		TraceFile:           ctx.String(TraceDirectoryFlag.Name) + "/trace.dat",
		Trace:               ctx.Bool(TraceFlag.Name),
	}
	if cfg.ChainID == 0 {
		cfg.ChainID = ChainIDFlag.Value
	}
	setFirstBlockFromChainID(cfg.ChainID)
	if cfg.SyncPeriodLength <= 0 {
		cfg.SyncPeriodLength = 300
	}
	if cfg.RandomSeed < 0 {
		cfg.RandomSeed = int64(rand.Uint32())
	}

	if mode == NoArgs {
		return cfg, nil
	}

	if _, err := os.Stat(cfg.DBDir); !os.IsNotExist(err) {
		log.Printf("Found merged DB: %s redirecting UpdateDB, DeletedAccountDB, SubstateDB paths to it\n", cfg.DBDir)
		cfg.UpdateDBDir = cfg.DBDir
		cfg.DeletedAccountDir = cfg.DBDir
		cfg.SubstateDBDir = cfg.DBDir
	}

	if cfg.EnableProgress {
		log.Printf("Run config:\n")
		log.Printf("\tBlock range: %v to %v\n", cfg.First, cfg.Last)
		if cfg.MaxNumTransactions >= 0 {
			log.Printf("\tTransaction limit: %d\n", cfg.MaxNumTransactions)
		}
		log.Printf("\tChain id: %v (record & run-vm only)\n", cfg.ChainID)
		log.Printf("\tSyncPeriod length: %v\n", cfg.SyncPeriodLength)

		logDbMode := func(prefix, impl, variant string) {
			if cfg.DbImpl == "carmen" {
				log.Printf("\t%s: %v, DB variant: %v, DB schema: %d\n", prefix, impl, variant, cfg.CarmenSchema)
			} else {
				log.Printf("\t%s: %v, DB variant: %v\n", prefix, impl, variant)
			}
		}
		if cfg.ShadowImpl == "" {
			logDbMode("Storage system", cfg.DbImpl, cfg.DbVariant)
		} else {
			logDbMode("Prime storage system", cfg.DbImpl, cfg.DbVariant)
			logDbMode("Shadow storage system", cfg.ShadowImpl, cfg.ShadowVariant)
		}
		log.Printf("\tSource storage directory (empty if new): %v\n", cfg.StateDbSrcDir)
		log.Printf("\tWorking storage directory: %v\n", cfg.StateDbTempDir)
		if cfg.ArchiveMode {
			log.Printf("\tArchive mode: enabled\n")
			if cfg.ArchiveVariant == "" {
				log.Printf("\tArchive variant: <implementation-default>\n")
			} else {
				log.Printf("\tArchive variant: %s\n", cfg.ArchiveVariant)
			}
		} else {
			log.Printf("\tArchive mode: disabled\n")
		}
		log.Printf("\tUsed VM implementation: %v\n", cfg.VmImpl)
		log.Printf("\tUpdate DB directory: %v\n", cfg.UpdateDBDir)
		if cfg.SkipPriming {
			log.Printf("\tPriming: Skipped\n")
		} else {
			log.Printf("\tRandomized Priming: %v\n", cfg.PrimeRandom)
			if cfg.PrimeRandom {
				log.Printf("\t\tSeed: %v, threshold: %v\n", cfg.PrimeSeed, cfg.PrimeThreshold)
			}
		}
		log.Printf("\tValidate world state: %v, validate tx state: %v\n", cfg.ValidateWorldState, cfg.ValidateTxState)
	}

	// TODO: enrich warning with colored text
	if cfg.ValidateTxState {
		log.Printf("WARNING: validation enabled, reducing Tx throughput\n")
	}
	if cfg.ShadowImpl != "" {
		log.Printf("WARNING: DB shadowing enabled, reducing Tx throughput and increasing memory and storage usage\n")
	}
	if cfg.DbLogging {
		log.Printf("WARNING: DB logging enabled, reducing Tx throughput\n")
	}
	if _, err := os.Stat(cfg.DeletedAccountDir); os.IsNotExist(err) {
		log.Printf("WARNING: deleted-account-dir is not provided or does not exist")
		cfg.HasDeletedAccounts = false
	}
	if cfg.KeepStateDB && cfg.ShadowImpl != "" {
		log.Printf("WARNING: keeping persistent stateDB with a shadow db is not supported yet")
		cfg.KeepStateDB = false
	}
	if cfg.KeepStateDB && strings.Contains(cfg.DbVariant, "memory") {
		log.Printf("WARNING: Unable to keep in-memory stateDB")
		cfg.KeepStateDB = false
	}
	if cfg.SkipPriming && cfg.ValidateWorldState {
		log.Printf("ERROR: skipPriming and validation of world state can not be enabled at the same time\n")
		return cfg, fmt.Errorf("skipPriming and world-state validation can not be enabled at the same time")
	}

	return cfg, nil
}

// SetBlockRange checks the validity of a block range and return the first and last block as numbers.
func SetBlockRange(firstArg string, lastArg string) (uint64, uint64, error) {
	var err error = nil
	first, ferr := strconv.ParseUint(firstArg, 10, 64)
	last, lerr := strconv.ParseUint(lastArg, 10, 64)
	if ferr != nil || lerr != nil {
		err = fmt.Errorf("error: block number not an integer")
	} else if first > last {
		err = fmt.Errorf("error: first block has larger number than last block")
	}
	return first, last, err
}
