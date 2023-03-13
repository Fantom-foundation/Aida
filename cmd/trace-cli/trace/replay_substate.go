package trace

import (
	"fmt"
	"os"
	"time"

	"github.com/Fantom-foundation/Aida/tracer"
	"github.com/Fantom-foundation/Aida/tracer/dictionary"
	"github.com/Fantom-foundation/Aida/tracer/operation"
	"github.com/Fantom-foundation/Aida/utils"
	substate "github.com/Fantom-foundation/Substate"
	"github.com/urfave/cli/v2"
)

// TraceReplaySubstateCommand data structure for the replay-substate app
var TraceReplaySubstateCommand = cli.Command{
	Action:    traceReplaySubstateAction,
	Name:      "replay-substate",
	Usage:     "executes storage trace using substates",
	ArgsUsage: "<blockNumFirst> <blockNumLast>",
	Flags: []cli.Flag{
		&utils.ChainIDFlag,
		&utils.CpuProfileFlag,
		&utils.DisableProgressFlag,
		&utils.RandomizePrimingFlag,
		&utils.PrimeSeedFlag,
		&utils.PrimeThresholdFlag,
		&utils.ProfileFlag,
		&utils.StateDbImplementationFlag,
		&utils.StateDbVariantFlag,
		&utils.StateDbLoggingFlag,
		&utils.ShadowDbImplementationFlag,
		&utils.ShadowDbVariantFlag,
		&substate.SubstateDirFlag,
		&substate.WorkersFlag,
		&utils.TraceDirectoryFlag,
		&utils.TraceDebugFlag,
		&utils.ValidateFlag,
		&utils.ValidateWorldStateFlag,
	},
	Description: `
The trace replay-substate command requires two arguments:
<blockNumFirst> <blockNumLast>

<blockNumFirst> and <blockNumLast> are the first and
last block of the inclusive range of blocks to replay storage traces.`,
}

// traceReplaySubstateTask simulates storage operations from storage traces on stateDB.
func traceReplaySubstateTask(cfg *utils.Config) error {
	// load dictionaries & indexes
	dCtx := dictionary.ReadContext()

	// iterate substate (for in-membory state)
	stateIter := substate.NewSubstateIterator(cfg.First, cfg.Workers)
	defer stateIter.Release()

	// replay storage trace
	traceIter := tracer.NewTraceIterator(cfg.First, cfg.Last)
	defer traceIter.Release()

	// Create a directory for the store to place all its files.
	db, stateDirectory, _, err := utils.PrepareStateDB(cfg)
	if err != nil {
		return err
	}
	defer os.RemoveAll(stateDirectory)

	var (
		start       time.Time
		sec         float64
		lastSec     float64
		lastTxCount uint64
		txCount     uint64
		firstBlock  = true
	)
	if cfg.EnableProgress {
		start = time.Now()
		sec = time.Since(start).Seconds()
		lastSec = time.Since(start).Seconds()
	}

	// A utility to run operations on the local context.
	run := func(op operation.Operation) {
		operation.Execute(op, db, dCtx)
		if cfg.Debug {
			operation.Debug(dCtx, op)
		}
	}

	for stateIter.Next() {
		tx := stateIter.Value()

		// The first Epoch begin and the final EpochEnd need to be artificially
		// added since the range running on may not match epoch boundaries.
		if firstBlock {
			run(operation.NewBeginEpoch(cfg.First / cfg.EpochLength))
			firstBlock = false
		}

		if tx.Block > cfg.Last {
			break
		}

		if cfg.DbImpl == "memory" {
			db.PrepareSubstate(&tx.Substate.InputAlloc, tx.Block)
		} else {
			utils.PrimeStateDB(tx.Substate.InputAlloc, db, cfg)
		}
		for traceIter.Next() {
			op := traceIter.Value()
			run(op)

			// find end of transaction
			if op.GetId() == operation.EndTransactionID {
				txCount++
				break
			}
		}

		// Validate stateDB and OuputAlloc
		if cfg.ValidateWorldState {
			if err := utils.ValidateStateDB(tx.Substate.OutputAlloc, db, false); err != nil {
				return fmt.Errorf("Validation failed. Block %v Tx %v\n\t%v\n", tx.Block, tx.Transaction, err)
			}
		}
		if cfg.EnableProgress {
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

	// replay the last EndBlock() and EndEpoch()
	hasNext := traceIter.Next()
	op := traceIter.Value()
	if !hasNext || op.GetId() != operation.EndBlockID {
		return fmt.Errorf("Last operation isn't an EndBlock")
	} else {
		run(op) // EndBlock
		run(operation.NewEndEpoch())
	}
	sec = time.Since(start).Seconds()

	// print profile statistics (if enabled)
	if operation.EnableProfiling {
		operation.PrintProfiling()
	}

	// close the DB and print disk usage
	start = time.Now()
	if err := db.Close(); err != nil {
		fmt.Printf("Failed to close database: %v", err)
	}

	if cfg.EnableProgress {
		fmt.Printf("trace replay-substate: Closing DB took %v\n", time.Since(start))
		fmt.Printf("trace replay-substate: Final disk usage: %v MiB\n", float32(utils.GetDirectorySize(stateDirectory))/float32(1024*1024))
		fmt.Printf("trace replay-substate: Total elapsed time: %.3f s, processed %v blocks (~%.1f Tx/s)\n", sec, cfg.Last-cfg.First+1, float64(txCount)/sec)
	}

	return nil
}

// traceReplaySubstateAction implements trace command for replaying.
func traceReplaySubstateAction(ctx *cli.Context) error {
	substate.RecordReplay = true
	cfg, err := utils.NewConfig(ctx, utils.BlockRangeArgs)
	if err != nil {
		return err
	}
	// run storage driver
	substate.SetSubstateFlags(cfg.SubstateDBDir)
	substate.OpenSubstateDBReadOnly()
	defer substate.CloseSubstateDB()

	// Get profiling flag
	operation.EnableProfiling = cfg.Profile
	// Start CPU profiling if requested.
	if err := utils.StartCPUProfile(cfg); err != nil {
		return err
	}
	defer utils.StopCPUProfile(cfg)

	err = traceReplaySubstateTask(cfg)

	return err
}
