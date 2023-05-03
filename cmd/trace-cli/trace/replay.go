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

const readBufferSize = 100000

// TraceReplayCommand data structure for the replay app
var TraceReplayCommand = cli.Command{
	Action:    traceReplayAction,
	Name:      "replay",
	Usage:     "executes storage trace",
	ArgsUsage: "<blockNumFirst> <blockNumLast>",
	Flags: []cli.Flag{
		&utils.CarmenSchemaFlag,
		&utils.ChainIDFlag,
		&utils.CpuProfileFlag,
		&utils.DeletionDbFlag,
		&utils.QuietFlag,
		&utils.SyncPeriodLengthFlag,
		&utils.KeepStateDbFlag,
		&utils.MemoryBreakdownFlag,
		&utils.MemoryProfileFlag,
		&utils.PrimeSeedFlag,
		&utils.PrimeThresholdFlag,
		&utils.ProfileFlag,
		&utils.RandomizePrimingFlag,
		&utils.SkipPrimingFlag,
		&utils.StateDbImplementationFlag,
		&utils.StateDbVariantFlag,
		&utils.StateDbSrcFlag,
		&utils.DbTmpFlag,
		&utils.StateDbLoggingFlag,
		&utils.ShadowDbImplementationFlag,
		&utils.ShadowDbVariantFlag,
		&substate.SubstateDirFlag,
		&substate.WorkersFlag,
		&utils.TraceFileFlag,
		&utils.TraceDebugFlag,
		&utils.DebugFromFlag,
		&utils.UpdateDbFlag,
		&utils.ValidateFlag,
		&utils.ValidateWorldStateFlag,
		&utils.AidaDbFlag,
		&utils.LogLevelFlag,
	},
	Description: `
The trace replay command requires two arguments:
<blockNumFirst> <blockNumLast>

<blockNumFirst> and <blockNumLast> are the first and
last block of the inclusive range of blocks to replay storage traces.`,
}

// readTrace reads operations from trace files and puts them into a channel.
func readTrace(cfg *utils.Config, ch chan operation.Operation) {
	traceIter := tracer.NewTraceIterator(cfg.TraceFile, cfg.First, cfg.Last)
	defer traceIter.Release()
	for traceIter.Next() {
		op := traceIter.Value()
		ch <- op
	}
	close(ch)
}

// traceReplayTask simulates storage operations from storage traces on stateDB.
func traceReplayTask(cfg *utils.Config, log *logging.Logger) error {

	// starting reading in parallel
	log.Notice("Start reading operations in parallel")
	opChannel := make(chan operation.Operation, readBufferSize)
	go readTrace(cfg, opChannel)

	// create a directory for the store to place all its files, and
	// instantiate the state DB under testing.
	log.Notice("Create stateDB database")
	db, stateDirectory, loadedExistingDB, err := utils.PrepareStateDB(cfg)
	if err != nil {
		return err
	}
	if !cfg.KeepStateDb {
		log.Warningf("Directory %v with DB will be removed at the end of this run.", stateDirectory)
		defer os.RemoveAll(stateDirectory)
	}

	if cfg.SkipPriming || loadedExistingDB {
		log.Warning("Skipping DB priming.")
	} else {
		// intialize the world state and advance it to the first block
		log.Noticef("Load and advance worldstate to block %v", cfg.First-1)
		ws, err := utils.GenerateWorldStateFromUpdateDB(cfg, cfg.First-1)
		if err != nil {
			return err
		}

		// prime stateDB
		log.Notice("Prime StateDB")
		utils.PrimeStateDB(ws, db, 0, cfg, log)

		// print memory usage after priming
		if cfg.MemoryBreakdown {
			if usage := db.GetMemoryUsage(); usage != nil {
				log.Noticef("State DB memory usage: %d byte\n%s", usage.UsedBytes, usage.Breakdown)
			} else {
				log.Noticef("Utilized storage solution does not support memory breakdowns.")
			}
		}

		// delete destroyed accounts from stateDB
		log.Noticef("Delete destroyed accounts")
		// remove destroyed accounts until one block before the first block
		if err = utils.DeleteDestroyedAccountsFromStateDB(db, cfg, cfg.First-1); err != nil {
			return err
		}
	}

	log.Noticef("Replay storage operations on StateDB database")

	// load context
	dCtx := context.NewReplay()

	// progress message setup
	var (
		start      time.Time
		sec        float64
		lastSec    float64
		firstBlock = true
		lastBlock  uint64
		debug      bool
	)
	if !cfg.Quiet {
		start = time.Now()
		sec = time.Since(start).Seconds()
		lastSec = time.Since(start).Seconds()
	}

	// A utility to run operations on the local context.
	run := func(op operation.Operation) {
		operation.Execute(op, db, dCtx)
		if debug {
			operation.Debug(&dCtx.Context, op)
		}
	}

	// replay storage trace
	for op := range opChannel {
		var block uint64
		if beginBlock, ok := op.(*operation.BeginBlock); ok {
			block = beginBlock.BlockNumber
			debug = cfg.Debug && block >= cfg.DebugFrom
			// The first SyncPeriod begin and the final SyncPeriodEnd need to be artificially
			// added since the range running on may not match sync-period boundaries.
			if firstBlock {
				run(operation.NewBeginSyncPeriod(cfg.First / cfg.SyncPeriodLength))
				firstBlock = false
			}

			if block > cfg.Last {
				run(operation.NewEndSyncPeriod())
				break
			}
			lastBlock = block // track the last processed block
			if !cfg.Quiet {
				// report progress
				hours, minutes, seconds := utils.ParseTime(time.Since(start))
				if sec-lastSec >= 15 {
					log.Infof("Elapsed time: %vh %vm %vs, at block %v\n", hours, minutes, seconds, block)
					lastSec = sec
				}
			}
		}
		run(op)
	}

	sec = time.Since(start).Seconds()

	log.Notice("Finished replaying storage operations on StateDB.")

	// destroy context to make space
	dCtx = nil

	// validate stateDB
	if cfg.ValidateWorldState {
		log.Notice("Validate final state")
		ws, err := utils.GenerateWorldStateFromUpdateDB(cfg, cfg.Last)
		if err = utils.DeleteDestroyedAccountsFromWorldState(ws, cfg, cfg.Last); err != nil {
			return fmt.Errorf("Failed to remove detroyed accounts. %v\n", err)
		}
		if err := utils.ValidateStateDB(ws, db, false); err != nil {
			return fmt.Errorf("Validation failed. %v\n", err)
		}
	}

	if cfg.MemoryBreakdown {
		if usage := db.GetMemoryUsage(); usage != nil {
			log.Notice("State DB memory usage: %d byte\n%s", usage.UsedBytes, usage.Breakdown)
		} else {
			log.Notice("Utilized storage solution does not support memory breakdowns.")
		}
	}

	// write memory profile if requested
	if err := utils.StartMemoryProfile(cfg); err != nil {
		return err
	}

	// print profile statistics (if enabled)
	if operation.EnableProfiling {
		operation.PrintProfiling()
	}

	if cfg.KeepStateDb {
		rootHash, _ := db.Commit(true)
		if err := utils.WriteStateDbInfo(stateDirectory, cfg, lastBlock, rootHash); err != nil {
			log.Error(err)
		}
		//rename directory after closing db.
		defer utils.RenameTempStateDBDirectory(cfg, stateDirectory, lastBlock)
	}

	// close the DB and print disk usage
	log.Notice("Close StateDB database")
	start = time.Now()
	if err := db.Close(); err != nil {
		log.Errorf("Failed to close database: %v", err)
	}

	// print progress summary
	if !cfg.Quiet {
		log.Noticef("Total elapsed time: %.3f s, processed %v blocks\n", sec, cfg.Last-cfg.First+1)
		log.Noticef("Closing DB took %v\n", time.Since(start))
		log.Noticef("Final disk usage: %v MiB\n", float32(utils.GetDirectorySize(stateDirectory))/float32(1024*1024))
	}

	return nil
}

// traceReplayAction implements trace command for replaying.
func traceReplayAction(ctx *cli.Context) error {
	var err error
	cfg, err := utils.NewConfig(ctx, utils.BlockRangeArgs)
	if err != nil {
		return err
	}
	if cfg.DbImpl == "memory" {
		return fmt.Errorf("db-impl memory is not supported")
	}

	operation.EnableProfiling = cfg.Profile

	// start CPU profiling if requested.
	if err := utils.StartCPUProfile(cfg); err != nil {
		return err
	}
	defer utils.StopCPUProfile(cfg)

	// run storage driver
	substate.SetSubstateDirectory(cfg.SubstateDb)
	substate.OpenSubstateDBReadOnly()
	defer substate.CloseSubstateDB()

	log := utils.NewLogger(cfg.LogLevel, "Trace Replay Action")
	err = traceReplayTask(cfg, log)

	return err
}
