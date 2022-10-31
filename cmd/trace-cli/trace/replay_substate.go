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

// Trace replay-substate command
var TraceReplaySubstateCommand = cli.Command{
	Action:    traceReplaySubstateAction,
	Name:      "replay-substate",
	Usage:     "executes storage trace using substates",
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
The trace replay-substate command requires two arguments:
<blockNumFirst> <blockNumLast>

<blockNumFirst> and <blockNumLast> are the first and
last block of the inclusive range of blocks to replay storage traces.`,
}

// traceReplaySubstateTask simulates storage operations from storage traces on stateDB.
func traceReplaySubstateTask(first uint64, last uint64, cliCtx *cli.Context) error {
	// load dictionaries & indexes
	dCtx := dict.ReadDictionaryContext()
	iCtx := tracer.ReadIndexContext()

	// iterate substate (for in-membory state)
	stateIter := substate.NewSubstateIterator(first, cliCtx.Int(substate.WorkersFlag.Name))
	defer stateIter.Release()

	// replay storage trace
	traceIter := tracer.NewTraceIterator(iCtx, first, last)
	defer traceIter.Release()

	// Get validation flag
	validation_enabled := cliCtx.Bool(validateEndState.Name)

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

	// Create a directory for the store to place all its files.
	state_directory, err := ioutil.TempDir("", "state_db_*")
	if err != nil {
		return err
	}

	// Instantiate the state DB under testing
	impl := cliCtx.String(stateDbImplementation.Name)
	variant := cliCtx.String(stateDbVariant.Name)
	db, err := makeStateDB(state_directory, impl, variant)
	if err != nil {
		return err
	}

	var (
		start       time.Time
		sec         float64
		lastSec     float64
		lastTxCount uint64
		txCount     uint64
	)
	if enableProgress {
		start = time.Now()
		sec = time.Since(start).Seconds()
		lastSec = time.Since(start).Seconds()
	}
	for stateIter.Next() {
		tx := stateIter.Value()
		if tx.Block > last || !iCtx.ExistsBlock(tx.Block) {
			break
		}
		db.PrepareSubstate(&tx.Substate.InputAlloc)
		for traceIter.Next() {
			op := traceIter.Value()
			// skip execution of sub balance if carmen or geth is used
			if op.GetOpId() == operation.SubBalanceID && impl != "memory" {
				continue
			}
			operation.Execute(op, db, dCtx)
			if traceDebug {
				operation.Debug(dCtx, op)
			}

			// find end of transaction
			if op.GetOpId() == operation.EndTransactionID {
				txCount++
				break
			}
		}

		// Validate stateDB and OuputAlloc
		if validation_enabled {
			traceAlloc := db.GetSubstatePostAlloc()
			recordedAlloc := tx.Substate.OutputAlloc
			err := compareSubstateStorage(recordedAlloc, traceAlloc)
			if err != nil {
				return err
			}
		}
		if enableProgress {
			// report progress
			sec = time.Since(start).Seconds()
			diff := sec - lastSec
			if diff >= 15 {
				numTx := txCount - lastTxCount
				lastTxCount = txCount
				fmt.Printf("trace replay-substate: Elapsed time: %.0f s, at block %v (~%.1f Tx/s)\n", sec, tx.Block, float64(numTx)/diff)
				lastSec = sec
			}
		}
	}

	// replay the last EndBlock()
	hasNext := traceIter.Next()
	op := traceIter.Value()
	if !hasNext || op.GetOpId() != operation.EndBlockID {
		return fmt.Errorf("Last operation isn't an EndBlock")
	} else {
		operation.Execute(op, db, dCtx)
		if traceDebug {
			operation.Debug(dCtx, op)
		}
	}

	if enableProgress {
		sec = time.Since(start).Seconds()
		fmt.Printf("trace replay-substate: Total elapsed time: %.3f s, processed %v blocks (~%.1f Tx/s)\n", sec, last-first+1, float64(txCount)/sec)
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

	if enableProgress {
		fmt.Printf("trace replay-substate: Closing DB took %v\n", time.Since(start))
		fmt.Printf("trace replay-substate: Final disk usage: %v MiB\n", float32(getDirectorySize(state_directory))/float32(1024*1024))
	}

	return nil
}

// Implements trace command for replaying.
func traceReplaySubstateAction(ctx *cli.Context) error {
	var err error

	// process arguments
	if ctx.Args().Len() != 2 {
		return fmt.Errorf("trace replay-substate command requires exactly 2 arguments")
	}
	tracer.TraceDir = ctx.String(traceDirectoryFlag.Name) + "/"
	dict.DictionaryContextDir = ctx.String(traceDirectoryFlag.Name) + "/"
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
	err = traceReplaySubstateTask(first, last, ctx)

	return err
}
