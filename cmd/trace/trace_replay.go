package trace

import (
	"fmt"
	cli "gopkg.in/urfave/cli.v1"

	"github.com/Fantom-foundation/aida/tracer"
	"github.com/Fantom-foundation/aida/tracer/dict"
	"github.com/Fantom-foundation/aida/tracer/operation"
	"github.com/Fantom-foundation/substate-cli/state"
	"github.com/ethereum/go-ethereum/substate"
)

// trace replay command
var TraceReplayCommand = cli.Command{
	Action:    traceReplayAction,
	Name:      "replay",
	Usage:     "executes storage trace",
	ArgsUsage: "<blockNumFirst> <blockNumLast>",
	Flags: []cli.Flag{
		substate.SubstateDirFlag,
		substate.WorkersFlag,
		TraceDirectoryFlag,
		TraceDebugFlag,
	},
	Description: `
The substate-cli trace-replay command requires two arguments:
<blockNumFirst> <blockNumLast>

<blockNumFirst> and <blockNumLast> are the first and
last block of the inclusive range of blocks to replay storage traces.`,
}

// comareStorage compares substae after replay traces to recorded substate
func compareStorage(recordedAlloc substate.SubstateAlloc, traceAlloc substate.SubstateAlloc) error {
	for account, recordAccount := range recordedAlloc {
		// account exists in both substate
		if replayAccout, exist := traceAlloc[account]; exist {
			for k, xv := range recordAccount.Storage {
				// mismatched value or key dones't exist
				if yv, exist := replayAccout.Storage[k]; !exist || xv != yv {
					return fmt.Errorf("Error: mismatched value at storage key %v. want %v have %v\n", k, xv, yv)
				}
			}
			for k, yv := range replayAccout.Storage {
				// key exists when expecting nil
				if xv, exist := recordAccount.Storage[k]; !exist {
					return fmt.Errorf("Error: mismatched value at storage key %v. want %v have %v\n", k, xv, yv)
				}
			}
		} else {
			if len(recordAccount.Storage) > 0 {
				return fmt.Errorf("Error: account %v doesn't exist\n", account)
			}
			//else ignores accounts which has no storage
		}
	}

	// checks for unexpected accounts in replayed substate
	for account := range traceAlloc {
		if _, exist := recordedAlloc[account]; !exist {
			return fmt.Errorf("Error: unexpected account %v\n", account)
		}
	}
	return nil
}

// storageDriver simulates storage operations from storage traces on stateDB
func storageDriver(first uint64, last uint64, cliCtx *cli.Context) error {
	// load dictionaries & indexes
	dCtx := dict.ReadDictionaryContext()
	iCtx := tracer.ReadIndexContext()

	// TODO: 1) compute full-state for "first" block, and
	//       2) transcribe full-state to the StateDB object
	//          under test.

	// iterate substate (for in-membory state)
	// TODO set configurable number of workers
	stateIter := substate.NewSubstateIterator(first, cliCtx.Int(substate.WorkersFlag.Name))
	defer stateIter.Release()

	// replay storage trace
	traceIter := tracer.NewTraceIterator(iCtx, first, last)
	defer traceIter.Release()

	var db state.StateDB

	for stateIter.Next() {
		tx := stateIter.Value()
		if tx.Block > last || !iCtx.ExistsBlock(tx.Block){
			break
		}
		db = state.MakeInMemoryStateDB(&tx.Substate.InputAlloc)
		for traceIter.Next() {
			op := traceIter.Value()
			op.Execute(db, dCtx)
			if traceDebug {
				operation.Debug(dCtx, op)
			}

			// find end of transaction
			if op.GetOpId() == operation.EndTransactionID {
				break
			}
		}

		// Validate stateDB and OuputAlloc
		traceAlloc := db.GetSubstatePostAlloc()
		recordedAlloc := tx.Substate.OutputAlloc
		err := compareStorage(recordedAlloc, traceAlloc)
		if err != nil {
			return err
		}
	}

	// replay the last EndBlock()
	hasNext := traceIter.Next()
	op := traceIter.Value()
	if !hasNext || op.GetOpId() != operation.EndBlockID{
		return fmt.Errorf("Last opertion isn't EndBlock")
	} else {
		op.Execute(db, dCtx)
		if traceDebug {
			operation.Debug(dCtx, op)
		}
	}

	return nil
}

// traceReplayAction configures applications acocrding to cli flags then replays traces
func traceReplayAction(ctx *cli.Context) error {
	var err error

	// process arguments
	if len(ctx.Args()) != 2 {
		return fmt.Errorf("trace replay-trace command requires exactly 2 arguments")
	}
	tracer.TraceDir = ctx.String(TraceDirectoryFlag.Name) + "/"
	dict.DictDir = ctx.String(TraceDirectoryFlag.Name) + "/"
	if ctx.Bool("trace-debug") {
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
	err = storageDriver(first, last, ctx)

	return err
}
