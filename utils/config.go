// Package trace provides cli for recording and replaying storage traces.
package utils

import (
	"fmt"
	"log"
	"math/big"
	"os"
	"strconv"
	"time"

	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/substate"
	"github.com/urfave/cli/v2"
)

type ArgumentMode int

// An enums of argument modes used by trace subcommands
const (
	BlockRangeArgs ArgumentMode = iota // requires 2 arguments: first block and last block
	LastBlockArg                       // requires 1 argument: last block
)

var (
	FirstSubstateBlock uint64         // id of the first block in substate
	TraceDebug         bool   = false // TraceDebug for enabling/disabling debugging.
)

// Command line options for common flags in record and replay.
var (
	ArchiveModeFlag = cli.BoolFlag{
		Name:  "archive",
		Usage: "set node type to archival mode. If set, the node keep all the EVM state history; otherwise the state history will be pruned.",
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
	DeletedAccountDirFlag = cli.StringFlag{
		Name:  "deleted-account-dir",
		Usage: "sets the directory containing deleted accounts database",
	}
	MemProfileFlag = cli.StringFlag{
		Name:  "memprofile",
		Usage: "enables memory allocation profiling",
	}
	EpochLengthFlag = cli.IntFlag{
		Name:  "epochlength",
		Usage: "defines the number of blocks per epoch",
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
		Value: "./updatedb",
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
)

// execution configuration for replay command.
type Config struct {
	First uint64 // first block
	Last  uint64 // last block

	Debug              bool   // enable trace debug flag
	ContinueOnFailure  bool   // continue validation when an error detected
	ChainID            int    // Blockchain ID (mainnet: 250/testnet: 4002)
	DbImpl             string // storage implementation
	DbVariant          string // database variant
	DbLogging          bool   // set to true if all DB operations should be logged
	DeletedAccountDir  string // directory of deleted account database
	EnableProgress     bool   // enable progress report flag
	EpochLength        uint64 // length of an epoch in number of blocks
	ArchiveMode        bool   // enable archive mode
	HasDeletedAccounts bool   // true if deletedAccountDir is not empty; otherwise false
	MaxNumTransactions int    // the maximum number of processed transactions
	MemoryBreakdown    bool   // enable printing of memory breakdown
	MemoryProfile      string // capture the memory heap profile into the file
	PrimeRandom        bool   // enable randomized priming
	PrimeSeed          int64  // set random seed
	PrimeThreshold     int    // set account threshold before commit
	Profile            bool   // enable micro profiling
	SkipPriming        bool   // skip priming of the state DB
	ShadowImpl         string // implementation of the shadow DB to use, empty if disabled
	ShadowVariant      string // database variant of the shadow DB to be used
	StateDbDir         string // directory to store State DB data
	UpdateDBDir        string // update-set directory
	ValidateTxState    bool   // validate stateDB before and after transaction
	ValidateWorldState bool   // validate stateDB before and after replay block range
	VmImpl             string // vm implementation (geth/lfvm)
	Workers            int    // number of worker threads
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
		ArchiveMode:        ctx.Bool(ArchiveModeFlag.Name),
		Debug:              ctx.Bool(TraceDebugFlag.Name),
		ChainID:            ctx.Int(ChainIDFlag.Name),
		ContinueOnFailure:  ctx.Bool(ContinueOnFailureFlag.Name),
		EnableProgress:     !ctx.Bool(DisableProgressFlag.Name),
		EpochLength:        ctx.Uint64(EpochLengthFlag.Name),
		First:              first,
		DbImpl:             ctx.String(StateDbImplementationFlag.Name),
		DbVariant:          ctx.String(StateDbVariantFlag.Name),
		DbLogging:          ctx.Bool(StateDbLoggingFlag.Name),
		DeletedAccountDir:  ctx.String(DeletedAccountDirFlag.Name),
		HasDeletedAccounts: true,
		Last:               last,
		MaxNumTransactions: ctx.Int(MaxNumTransactionsFlag.Name),
		MemoryBreakdown:    ctx.Bool(MemoryBreakdownFlag.Name),
		PrimeRandom:        ctx.Bool(RandomizePrimingFlag.Name),
		PrimeSeed:          ctx.Int64(PrimeSeedFlag.Name),
		PrimeThreshold:     ctx.Int(PrimeThresholdFlag.Name),
		Profile:            ctx.Bool(ProfileFlag.Name),
		SkipPriming:        ctx.Bool(SkipPrimingFlag.Name),
		ShadowImpl:         ctx.String(ShadowDbImplementationFlag.Name),
		ShadowVariant:      ctx.String(ShadowDbVariantFlag.Name),
		StateDbDir:         ctx.String(StateDbTempDirFlag.Name),
		UpdateDBDir:        ctx.String(UpdateDBDirFlag.Name),
		ValidateTxState:    validateTxState,
		ValidateWorldState: validateWorldState,
		VmImpl:             ctx.String(VmImplementation.Name),
		Workers:            ctx.Int(substate.WorkersFlag.Name),
		MemoryProfile:      ctx.String(MemProfileFlag.Name),
	}
	setFirstBlockFromChainID(cfg.ChainID)
	if cfg.EpochLength <= 0 {
		cfg.EpochLength = 300
	}

	if cfg.EnableProgress {
		log.Printf("Run config:\n")
		log.Printf("\tBlock range: %v to %v\n", cfg.First, cfg.Last)
		if cfg.MaxNumTransactions >= 0 {
			log.Printf("\tTransaction limit: %d\n", cfg.MaxNumTransactions)
		}
		log.Printf("\tChain id: %v (record & run-vm only)\n", cfg.ChainID)
		log.Printf("\tEpoch length: %v\n", cfg.EpochLength)
		if cfg.ShadowImpl == "" {
			log.Printf("\tStorage system: %v, DB variant: %v\n", cfg.DbImpl, cfg.DbVariant)
		} else {
			log.Printf("\tPrime storage system: %v, DB variant: %v\n", cfg.DbImpl, cfg.DbVariant)
			log.Printf("\tShadow storage system: %v, DB variant: %v\n", cfg.ShadowImpl, cfg.ShadowVariant)
		}
		log.Printf("\tStorage parent directory: %v\n", cfg.StateDbDir)
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
	} else if first < 0 || last < 0 {
		err = fmt.Errorf("error: block number must be greater than 0")
	} else if first > last {
		err = fmt.Errorf("error: first block has larger number than last block")
	}
	return first, last, err
}
