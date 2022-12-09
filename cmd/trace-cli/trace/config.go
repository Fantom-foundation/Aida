// Package trace provides cli for recording and replaying storage traces.
package trace

import (
	"fmt"
	"log"
	"math/big"
	"strconv"
	"time"

	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/substate"
	"github.com/urfave/cli/v2"
)

var (
	FirstSubstateBlock uint64         // id of the first block in substate
	traceDebug         bool   = false // traceDebug for enabling/disabling debugging.
)

// Command line options for common flags in record and replay.
var (
	chainIDFlag = cli.IntFlag{
		Name:  "chainid",
		Usage: "ChainID for replayer",
		Value: 250,
	}
	continueOnFailureFlag = cli.BoolFlag{
		Name:  "continue-on-failure",
		Usage: "continue execute after validation failure detected",
	}
	cpuProfileFlag = cli.StringFlag{
		Name:  "cpuprofile",
		Usage: "enables CPU profiling",
	}
	memProfileFlag = cli.StringFlag{
		Name:  "memprofile",
		Usage: "enables memory allocation profiling",
	}
	epochLengthFlag = cli.IntFlag{
		Name:  "epochlength",
		Usage: "defines the number of blocks per epoch",
		Value: 300, // ~ 300s = 5 minutes
	}
	memoryBreakdownFlag = cli.BoolFlag{
		Name:  "memory-breakdown",
		Usage: "enables printing of memory usage breakdown",
	}
	profileFlag = cli.BoolFlag{
		Name:  "profile",
		Usage: "enables profiling",
	}
	disableProgressFlag = cli.BoolFlag{
		Name:  "disable-progress",
		Usage: "disable progress report",
	}
	randomizePrimingFlag = cli.BoolFlag{
		Name:  "prime-random",
		Usage: "randomize order of accounts in StateDB priming",
	}
	primeSeedFlag = cli.Int64Flag{
		Name:  "prime-seed",
		Usage: "set seed for randomizing priming",
		Value: time.Now().UnixNano(),
	}
	primeThresholdFlag = cli.IntFlag{
		Name:  "prime-threshold",
		Usage: "set number of accounts written to stateDB before applying pending state updates",
		Value: 0,
	}
	stateDbImplementationFlag = cli.StringFlag{
		Name:  "db-impl",
		Usage: "select state DB implementation",
		Value: "geth",
	}
	stateDbVariantFlag = cli.StringFlag{
		Name:  "db-variant",
		Usage: "select a state DB variant",
		Value: "",
	}
	stateDbTempDirFlag = cli.StringFlag{
		Name:  "db-tmp-dir",
		Usage: "sets the temporary directory where to place state DB data; uses system default if empty",
	}
	stateDbLoggingFlag = cli.BoolFlag{
		Name:  "db-logging",
		Usage: "enable logging of all DB operations",
	}
	shadowDbImplementationFlag = cli.StringFlag{
		Name:  "db-shadow-impl",
		Usage: "select state DB implementation to shadow the prime DB implementation",
		Value: "",
	}
	shadowDbVariantFlag = cli.StringFlag{
		Name:  "db-shadow-variant",
		Usage: "select a state DB variant to shadow the prime DB implementation",
		Value: "",
	}
	traceDebugFlag = cli.BoolFlag{
		Name:  "trace-debug",
		Usage: "enable debug output for tracing",
	}
	traceDirectoryFlag = cli.StringFlag{
		Name:  "tracedir",
		Usage: "set storage trace's output directory",
		Value: "./",
	}
	updateDBDirFlag = cli.StringFlag{
		Name:  "updatedir",
		Usage: "set update-set database directory",
		Value: "./updatedb",
	}
	validateFlag = cli.BoolFlag{
		Name:  "validate",
		Usage: "enables validation",
	}
	validateTxStateFlag = cli.BoolFlag{
		Name:  "validate-tx",
		Usage: "enables transaction state validation",
	}
	validateWorldStateFlag = cli.BoolFlag{
		Name:  "validate-ws",
		Usage: "enables end-state validation",
	}
	vmImplementation = cli.StringFlag{
		Name:  "vm-impl",
		Usage: "select VM implementation",
		Value: "geth",
	}
	worldStateDirFlag = cli.PathFlag{
		Name:  "worldstatedir",
		Usage: "world state snapshot database path",
	}
	numberOfBlocksFlag = cli.IntFlag{
		Name:     "number",
		Aliases:  []string{"n"},
		Usage:    "Number of blocks",
		Required: true,
		Value:    0,
	}
	stochasticMatrixFlag = cli.StringFlag{
		Name:  "stochastic-matrix",
		Usage: "set stochastic matrix file",
		Value: "stochastic-matrix.csv",
	}
	stochasticMatrixFormatFlag = cli.StringFlag{
		Name:  "stochastic-matrix-format",
		Usage: "type of the output matrix file (\"dot\" or \"csv\")",
		Value: "csv",
	}
	stochasticSeedFlag = cli.Int64Flag{
		Name:  "seed",
		Usage: "seed for pseudorandom number generator",
		Value: -1,
	}
)

// execution configuration for replay command.
type TraceConfig struct {
	first uint64 // first block
	last  uint64 // last block

	debug              bool   // enable trace debug flag
	continueOnFailure  bool   // continue validation when an error detected
	chainID            int    // Blockchain ID (mainnet: 250/testnet: 4002)
	dbImpl             string // storage implementation
	dbVariant          string // database variant
	dbLogging          bool   // set to true if all DB operations should be logged
	enableProgress     bool   // enable progress report flag
	epochLength        uint64 // length of an epoch in number of blocks
	memoryBreakdown    bool   // enable printing of memory breakdown
	primeRandom        bool   // enable randomized priming
	primeSeed          int64  // set random seed
	primeThreshold     int    // set account threshold before commit
	profile            bool   // enable micro profiling
	shadowImpl         string // implementation of the shadow DB to use, empty if disabled
	shadowVariant      string // database variant of the shadow DB to be used
	stateDbDir         string // directory to store State DB data
	updateDBDir        string // update-set directory
	validateTxState    bool   // validate stateDB before and after transaction
	validateWorldState bool   // validate stateDB before and after replay block range
	vmImpl             string // vm implementation (geth/lfvm)
	workers            int    // number of worker threads
}

// getChainConnfig returns chain configuration of either mainnet or testnets.
func getChainConfig(chainID int) *params.ChainConfig {
	var chainConfig *params.ChainConfig
	chainConfig = params.AllEthashProtocolChanges
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

// NewTraceConfig creates and initializes TraceConfig with commandline arguments.
func NewTraceConfig(ctx *cli.Context) (*TraceConfig, error) {
	// number of blocks to be generated by Stochastic
	n := ctx.Uint64(numberOfBlocksFlag.Name)

	var first, last uint64
	if n != 0 {
		first = 1
		last = n
	} else {
		// process arguments and flags
		if ctx.Args().Len() != 2 {
			return nil, fmt.Errorf("trace command requires exactly 2 arguments")
		}
		var argErr error
		first, last, argErr = SetBlockRange(ctx.Args().Get(0), ctx.Args().Get(1))
		if argErr != nil {
			return nil, argErr
		}
	}

	// --continue-on-failure implicitly enables transaction state validation
	validateTxState := ctx.Bool(validateFlag.Name) ||
		ctx.Bool(validateTxStateFlag.Name) ||
		ctx.Bool(continueOnFailureFlag.Name)
	validateWorldState := ctx.Bool(validateFlag.Name) ||
		ctx.Bool(validateWorldStateFlag.Name)

	cfg := &TraceConfig{
		first: first,
		last:  last,

		debug:              ctx.Bool(traceDebugFlag.Name),
		chainID:            ctx.Int(chainIDFlag.Name),
		continueOnFailure:  ctx.Bool(continueOnFailureFlag.Name),
		enableProgress:     !ctx.Bool(disableProgressFlag.Name),
		epochLength:        ctx.Uint64(epochLengthFlag.Name),
		dbImpl:             ctx.String(stateDbImplementationFlag.Name),
		dbVariant:          ctx.String(stateDbVariantFlag.Name),
		dbLogging:          ctx.Bool(stateDbLoggingFlag.Name),
		memoryBreakdown:    ctx.Bool(memoryBreakdownFlag.Name),
		primeRandom:        ctx.Bool(randomizePrimingFlag.Name),
		primeSeed:          ctx.Int64(primeSeedFlag.Name),
		primeThreshold:     ctx.Int(primeThresholdFlag.Name),
		profile:            ctx.Bool(profileFlag.Name),
		shadowImpl:         ctx.String(shadowDbImplementationFlag.Name),
		shadowVariant:      ctx.String(shadowDbVariantFlag.Name),
		stateDbDir:         ctx.String(stateDbTempDirFlag.Name),
		updateDBDir:        ctx.String(updateDBDirFlag.Name),
		validateTxState:    validateTxState,
		validateWorldState: validateWorldState,
		vmImpl:             ctx.String(vmImplementation.Name),
		workers:            ctx.Int(substate.WorkersFlag.Name),
	}
	setFirstBlockFromChainID(cfg.chainID)
	if cfg.epochLength <= 0 {
		cfg.epochLength = 300
	}

	if cfg.enableProgress {
		log.Printf("Run config:\n")
		log.Printf("\tBlock range: %v to %v\n", cfg.first, cfg.last)
		log.Printf("\tChain id: %v (record & run-vm only)\n", cfg.chainID)
		log.Printf("\tEpoch length: %v\n", cfg.epochLength)
		if cfg.shadowImpl == "" {
			log.Printf("\tStorage system: %v, DB variant: %v\n", cfg.dbImpl, cfg.dbVariant)
		} else {
			log.Printf("\tPrime storage system: %v, DB variant: %v\n", cfg.dbImpl, cfg.dbVariant)
			log.Printf("\tShadow storage system: %v, DB variant: %v\n", cfg.shadowImpl, cfg.shadowVariant)
		}
		log.Printf("\tStorage parent directory: %v\n", cfg.stateDbDir)
		log.Printf("\tUsed VM implementation: %v\n", cfg.vmImpl)
		log.Printf("\tUpdate DB directory: %v\n", cfg.updateDBDir)
		log.Printf("\tRandomized Priming: %v\n", cfg.primeRandom)
		if cfg.primeRandom {
			log.Printf("\t\tSeed: %v, threshold: %v\n", cfg.primeSeed, cfg.primeThreshold)
		}
		log.Printf("\tValidate world state: %v, validate tx state: %v\n", cfg.validateWorldState, cfg.validateTxState)
	}

	if cfg.validateTxState {
		log.Printf("WARNING: validation enabled, reducing Tx throughput\n")
	}
	if cfg.shadowImpl != "" {
		log.Printf("WARNING: DB shadowing enabled, reducing Tx throughput and increasing memory and storage usage\n")
	}
	if cfg.dbLogging {
		log.Printf("WARNING: DB logging enabled, reducing Tx throughput\n")
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
