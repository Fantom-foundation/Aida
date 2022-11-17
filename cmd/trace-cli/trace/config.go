// Package trace provides cli for recording and replaying storage traces.
package trace

import (
	"fmt"
	"log"
	"strconv"

	"github.com/ethereum/go-ethereum/substate"
	"github.com/urfave/cli/v2"
)

// chainId for recording either mainnet or testnets.
var chainID int

// traceDebug for enabling/disabling debugging.
var traceDebug bool = false

// Command line options for common flags in record and replay.
var (
	chainIDFlag = cli.IntFlag{
		Name:  "chainid",
		Usage: "ChainID for replayer",
		Value: 250,
	}
	cpuProfileFlag = cli.StringFlag{
		Name:  "cpuprofile",
		Usage: "enables CPU profiling",
	}
	profileFlag = cli.BoolFlag{
		Name:  "profile",
		Usage: "enables profiling",
	}
	disableProgressFlag = cli.BoolFlag{
		Name:  "disable-progress",
		Usage: "disable progress report",
	}
	vmImplementation = cli.StringFlag{
		Name:  "vm-impl",
		Usage: "select VM implementation",
		Value: "geth",
	}
	stateDbImplementation = cli.StringFlag{
		Name:  "db-impl",
		Usage: "select state DB implementation",
		Value: "memory",
	}
	stateDbVariant = cli.StringFlag{
		Name:  "db-variant",
		Usage: "select a state DB variant",
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
	validateEndState = cli.BoolFlag{
		Name:  "validate",
		Usage: "enables end-state validation",
	}
	worldStateDirFlag = cli.PathFlag{
		Name:  "worldstatedir",
		Usage: "world state snapshot database path",
	}
)

// execution configuration for replay command.
type TraceConfig struct {
	first uint64 // first block
	last  uint64 // last block

	debug            bool   // enable trace debug flag
	enableValidation bool   // enable validation flag
	enableProgress   bool   // enable progress report flag
	impl             string // storage implementation
	profile          bool   // enable micro profiling
	updateDBDir      string // update-set directory
	variant          string // database variant
	workers          int    // number of worker threads
}

// NewTraceConfig creates and initializes TraceConfig with commandline arguments.
func NewTraceConfig(ctx *cli.Context) (*TraceConfig, error) {
	// process arguments and flags
	if ctx.Args().Len() != 2 {
		return nil, fmt.Errorf("trace command requires exactly 2 arguments")
	}
	first, last, argErr := SetBlockRange(ctx.Args().Get(0), ctx.Args().Get(1))
	if argErr != nil {
		return nil, argErr
	}

	cfg := &TraceConfig{
		first: first,
		last:  last,

		debug:            ctx.Bool(traceDebugFlag.Name),
		enableValidation: ctx.Bool(validateEndState.Name),
		enableProgress:   !ctx.Bool(disableProgressFlag.Name),
		impl:             ctx.String(stateDbImplementation.Name),
		profile:          ctx.Bool(profileFlag.Name),
		updateDBDir:      ctx.String(updateDBDirFlag.Name),
		variant:          ctx.String(stateDbVariant.Name),
		workers:          ctx.Int(substate.WorkersFlag.Name),
	}

	if cfg.enableProgress {
		log.Printf("Run config:\n")
		log.Printf("\tBlock range: %v to %v\n", cfg.first, cfg.last)
		log.Printf("\tStorage system: %v, DB variant: %v\n", cfg.impl, cfg.variant)
		log.Printf("\tUpdate DB directory: %v\n", cfg.updateDBDir)
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
