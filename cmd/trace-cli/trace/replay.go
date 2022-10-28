package trace

import (
	"fmt"
	"io/ioutil"
	"os"
	"runtime/pprof"
	"time"

	"github.com/Fantom-foundation/Aida/cmd/gen-world-state/flags"
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
		&flags.StateDBPath,
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

	// get validation flag
	validationEnabled := cliCtx.Bool(validateEndState.Name)
	// get profiling flag
	operation.Profiling = cliCtx.Bool(profileFlag.Name)
	// get progress flag
	enableProgress := !cliCtx.Bool(disableProgressFlag.Name)
	// start CPU profiling if requested.
	if profile_file_name := cliCtx.String(cpuProfileFlag.Name); profile_file_name != "" {
		f, err := os.Create(profile_file_name)
		if err != nil {
			return err
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}
	// get number of workers
	workers := cliCtx.Int(substate.WorkersFlag.Name)

	// intialize the world state and advance it to the first block
	fmt.Printf("trace replay: Load and advance worldstate to block %v\n", first-1)
	ws, err := generateWorldState(cliCtx.String(flags.StateDBPath.Name), first-1, workers)
	if err != nil {
		return err
	}

	// create a directory for the store to place all its files.
	state_directory, err := ioutil.TempDir("", "state_db_*")
	if err != nil {
		return err
	}

	// instantiate the state DB under testing
	impl := cliCtx.String(stateDbImplementation.Name)
	variant := cliCtx.String(stateDbVariant.Name)
	db, err := makeStateDB(state_directory, impl, variant)
	if err != nil {
		return err
	}

	// prime stateDB
	fmt.Printf("trace replay: Prime stateDB\n")
	if cliCtx.String(stateDbImplementation.Name) != "memory" {
		primeStateDB(ws, db)
	} else {
		db.PrepareSubstate(&ws)
	}

	// initialize trace interator
	traceIter := tracer.NewTraceIterator(iCtx, first, last)
	defer traceIter.Release()

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

	// replace storage trace
	fmt.Printf("trace replay: Replay storage operations\n")
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


	// validate stateDB
	if validationEnabled {
		// advance the world state from the first block to the last block
		advanceWorldState(ws, first, last, workers)
		if err := validateStateDB(ws, db); err != nil {
			return fmt.Errorf("Validation failed. %v\n", err)
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

// traceReplayAction implements trace command for replaying.
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
