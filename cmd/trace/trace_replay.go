package trace

import (
	"fmt"
	"io/ioutil"
	"os"
	"runtime/pprof"
	"time"

	"github.com/Fantom-foundation/Aida/tracer"
	"github.com/Fantom-foundation/Aida/tracer/dict"
	"github.com/Fantom-foundation/Aida/tracer/operation"
	"github.com/ethereum/go-ethereum/substate"
	"github.com/urfave/cli/v2"
)

// Trace replay command
var TraceReplayCommand = cli.Command{
	Action:    traceReplayAction,
	Name:      "replay",
	Usage:     "executes storage trace",
	ArgsUsage: "<blockNumFirst> <blockNumLast>",
	Flags: []cli.Flag{
		&cpuProfileFlag,
		&disableProgressFlag,
		&profileFlag,
		&stateDbImplementation,
		&stateDbVariant,
		&substate.SubstateDirFlag,
		&substate.WorkersFlag,
		&traceDirectoryFlag,
		&traceDebugFlag,
		&validateEndState,
	},
	Description: `
The trace replay command requires two arguments:
<blockNumFirst> <blockNumLast>

<blockNumFirst> and <blockNumLast> are the first and
last block of the inclusive range of blocks to replay storage traces.`,
}

// traceReplayTask simulates storage operations from storage traces on stateDB.
func traceReplayTask(first uint64, last uint64, cliCtx *cli.Context) error {
	// load dictionaries & indexes
	dCtx := dict.ReadDictionaryContext()
	iCtx := tracer.ReadIndexContext()

	// Get validation flag
	validationEnabled := cliCtx.Bool(validateEndState.Name)

	// Get profiling flag
	operation.Profiling = cliCtx.Bool(profileFlag.Name)

	// Get progress flag
	enableProgress := !cliCtx.Bool(disableProgressFlag.Name)

	// Start CPU profiling if requested.
	if profile_file_name := cliCtx.String(cpuProfileFlag.Name); profile_file_name != "" {
		f, err := os.Create(profile_file_name)
		if err != nil {
			return err
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	workers := cliCtx.Int(substate.WorkersFlag.Name)

	// intialize the world state of the first block
	ws := generateWorldState(first, workers)

	// replay storage trace
	traceIter := tracer.NewTraceIterator(iCtx, first, last)
	defer traceIter.Release()


	// Create a directory for the store to place all its files.
	state_directory, err := ioutil.TempDir("", "state_db_*")
	if err != nil {
		return err
	}

	// Instantiate the state DB under testing
	db, err := makeStateDB(state_directory, cliCtx)
	if err != nil {
		return err
	}
	if cliCtx.String(stateDbImplementation.Name) != "memory" {
		primeStateDB(ws, db)
	} else {
		db.PrepareSubstate(&ws)
	}

	if err := validateStateDB(ws, db); err != nil {
		return fmt.Errorf("Pre: Validation failed. %v\n", err)
	}
	var (
		start   time.Time
		sec     float64
		lastSec float64
	)
	if enableProgress {
		start = time.Now()
		sec = time.Since(start).Seconds()
		lastSec = time.Since(start).Seconds()
	}

	for traceIter.Next() {
		op := traceIter.Value()
		if op.GetOpId() == operation.BeginBlockID {
			block := op.(*operation.BeginBlock).BlockNumber
			if block > last {
				break
			}
			if enableProgress {
				// report progress
				sec = time.Since(start).Seconds()
				if sec-lastSec >= 15 {
					fmt.Printf("trace replay: Elapsed time: %.0f s, at block %v\n", sec, block)
					lastSec = sec
				}
			}


		}
		operation.Execute(op, db, dCtx)
		if traceDebug {
			operation.Debug(dCtx, op)
		}

	}

	sec = time.Since(start).Seconds()

	advanceWorldState(ws, first, last, workers)
	// Validate stateDB
	if validationEnabled {
		if err := validateStateDB(ws, db); err != nil {
			return fmt.Errorf("Post: Validation failed. %v\n", err)
		}
	}

	// print profile statistics (if enabled)
	if operation.Profiling {
		operation.PrintProfiling()
	}

	// close the DB and print disk usage
	start = time.Now()
	if err := db.Close(); err != nil {
		fmt.Printf("Failed to close database: %v", err)
	}

	// print progress summary
	if enableProgress {
		fmt.Printf("trace replay: Total elapsed time: %.3f s, processed %v blocks\n", sec, last-first+1)
		fmt.Printf("trace replay: Closing DB took %v\n", time.Since(start))
		fmt.Printf("trace replay: Final disk usage: %v MiB\n", float32(getDirectorySize(state_directory))/float32(1024*1024))
	}

	return nil
}

// Implements trace command for replaying.
func traceReplayAction(ctx *cli.Context) error {
	var err error

	// process arguments
	if ctx.Args().Len() != 2 {
		return fmt.Errorf("trace replay command requires exactly 2 arguments")
	}
	tracer.TraceDir = ctx.String(traceDirectoryFlag.Name) + "/"
	dict.DictDir = ctx.String(traceDirectoryFlag.Name) + "/"
	if ctx.Bool(traceDebugFlag.Name) {
		traceDebug = true
	}
	first, last, argErr := SetBlockRange(ctx.Args().Get(0), ctx.Args().Get(1))
	if argErr != nil {
		return argErr
	}

	// run storage driver
	substate.SetSubstateFlags(ctx)
	substate.OpenSubstateDBReadOnly()
	defer substate.CloseSubstateDB()
	err = traceReplayTask(first, last, ctx)

	return err
}
