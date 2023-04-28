package trace

import (
	"fmt"
	"os"
	"time"

	"github.com/Fantom-foundation/Aida/tracer"
	"github.com/Fantom-foundation/Aida/tracer/context"
	"github.com/Fantom-foundation/Aida/tracer/operation"
	"github.com/Fantom-foundation/Aida/utils"
	substate "github.com/Fantom-foundation/Substate"
	"github.com/op/go-logging"
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
		&utils.QuietFlag,
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
		&utils.TraceFileFlag,
		&utils.TraceDebugFlag,
		&utils.DebugFromFlag,
		&utils.ValidateFlag,
		&utils.ValidateWorldStateFlag,
		&utils.AidaDbFlag,
		&utils.LogLevel,
	},
	Description: `
The trace replay-substate command requires two arguments:
<blockNumFirst> <blockNumLast>

<blockNumFirst> and <blockNumLast> are the first and
last block of the inclusive range of blocks to replay storage traces.`,
}

// traceReplaySubstateTask simulates storage operations from storage traces on stateDB.
func traceReplaySubstateTask(cfg *utils.Config, log *logging.Logger) error {
	// load context
	rCtx := context.NewReplay()

	// iterate substate (for in-membory state)
	stateIter := substate.NewSubstateIterator(cfg.First, cfg.Workers)
	defer stateIter.Release()

	// replay storage trace
	traceIter := tracer.NewTraceIterator(cfg.TraceFile, cfg.First, cfg.Last)
	defer traceIter.Release()

	// Create a directory for the store to place all its files.
	db, stateDbDir, err := utils.PrepareStateDB(cfg)
	if err != nil {
		return err
	}
	defer os.RemoveAll(cfg.StateDbSrc)

	var (
		start        time.Time
		sec          float64
		lastSec      float64
		lastTxCount  uint64
		txCount      uint64
		isFirstBlock = true
		debug        bool // if set enable trace debug
	)
	if !cfg.Quiet {
		start = time.Now()
		sec = time.Since(start).Seconds()
		lastSec = time.Since(start).Seconds()
	}

	// A utility to run operations on the local context.
	run := func(op operation.Operation) {
		operation.Execute(op, db, rCtx)
		if debug {
			operation.Debug(&rCtx.Context, op)
		}
	}

	for stateIter.Next() {
		tx := stateIter.Value()
		debug = cfg.Debug && tx.Block >= cfg.DebugFrom
		// The first SyncPeriod begin and the final SyncPeriodEnd need to be artificially
		// added since the range running on may not match sync-period boundaries.
		if isFirstBlock {
			run(operation.NewBeginSyncPeriod(cfg.First / cfg.SyncPeriodLength))
			isFirstBlock = false
		}

		if tx.Block > cfg.Last {
			break
		}

		if cfg.DbImpl == "memory" {
			db.PrepareSubstate(&tx.Substate.InputAlloc, tx.Block)
		} else {
			utils.PrimeStateDB(tx.Substate.InputAlloc, db, cfg, log)
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
		if !cfg.Quiet {
			// report progress
			sec = time.Since(start).Seconds()
			diff := sec - lastSec
			if diff >= 15 {
				numTx := txCount - lastTxCount
				lastTxCount = txCount
				hours, minutes, seconds := utils.ParseTime(time.Since(start))
				log.Infof("Elapsed time: %vh, %vm %vs, at block %v (~%.0f Tx/s)\n", hours, minutes, seconds, tx.Block, float64(numTx)/diff)
				lastSec = sec
			}
		}
	}

	// replay the last EndBlock() and EndSyncPeriod()
	hasNext := traceIter.Next()
	op := traceIter.Value()
	if !hasNext || op.GetId() != operation.EndBlockID {
		return fmt.Errorf("Last operation isn't an EndBlock")
	} else {
		run(op) // EndBlock
		run(operation.NewEndSyncPeriod())
	}
	sec = time.Since(start).Seconds()

	// print profile statistics (if enabled)
	if operation.EnableProfiling {
		operation.PrintProfiling()
	}

	// close the DB and print disk usage
	start = time.Now()
	if err := db.Close(); err != nil {
		log.Errorf("Failed to close database; %v", err)
	}

	if !cfg.Quiet {
		log.Infof("Closing DB took %v", time.Since(start))
		log.Infof("Final disk usage: %v MiB", float32(utils.GetDirectorySize(stateDbDir))/float32(1024*1024))
		log.Infof("Total elapsed time: %.3f s, processed %v blocks (~%.1f Tx/s)", sec, cfg.Last-cfg.First+1, float64(txCount)/sec)
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
	substate.SetSubstateDirectory(cfg.SubstateDb)
	substate.OpenSubstateDBReadOnly()
	defer substate.CloseSubstateDB()

	// Get profiling flag
	operation.EnableProfiling = cfg.Profile
	// Start CPU profiling if requested.
	if err := utils.StartCPUProfile(cfg); err != nil {
		return err
	}
	defer utils.StopCPUProfile(cfg)

	log := utils.NewLogger(ctx.String(utils.LogLevel.Name), "Trace Replay Substate Action")
	err = traceReplaySubstateTask(cfg, log)

	return err
}
