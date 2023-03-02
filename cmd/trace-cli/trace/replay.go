package trace

import (
	"fmt"
	"log"
	"os"
	"runtime"
	"runtime/pprof"
	"time"

	"github.com/Fantom-foundation/Aida/tracer"
	"github.com/Fantom-foundation/Aida/tracer/dictionary"
	"github.com/Fantom-foundation/Aida/tracer/operation"
	"github.com/Fantom-foundation/Aida/utils"
	substate "github.com/Fantom-foundation/Substate"
	"github.com/urfave/cli/v2"
)

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
		&utils.DeletedAccountDirFlag,
		&utils.DisableProgressFlag,
		&utils.EpochLengthFlag,
		&utils.KeepStateDBFlag,
		&utils.MemoryBreakdownFlag,
		&utils.MemProfileFlag,
		&utils.PrimeSeedFlag,
		&utils.PrimeThresholdFlag,
		&utils.ProfileFlag,
		&utils.RandomizePrimingFlag,
		&utils.SkipPrimingFlag,
		&utils.StateDbImplementationFlag,
		&utils.StateDbVariantFlag,
		&utils.StateDbSrcDirFlag,
		&utils.StateDbTempDirFlag,
		&utils.StateDbLoggingFlag,
		&utils.ShadowDbImplementationFlag,
		&utils.ShadowDbVariantFlag,
		&substate.SubstateDirFlag,
		&substate.WorkersFlag,
		&utils.TraceDirectoryFlag,
		&utils.TraceDebugFlag,
		&utils.UpdateDBDirFlag,
		&utils.ValidateFlag,
		&utils.ValidateWorldStateFlag,
	},
	Description: `
The trace replay command requires two arguments:
<blockNumFirst> <blockNumLast>

<blockNumFirst> and <blockNumLast> are the first and
last block of the inclusive range of blocks to replay storage traces.`,
}

// readTrace reads operations from trace files and puts them into a channel.
func readTrace(cfg *utils.Config, ch chan operation.Operation) {
	traceIter := tracer.NewTraceIterator(cfg.First, cfg.Last)
	defer traceIter.Release()
	for traceIter.Next() {
		op := traceIter.Value()
		ch <- op
	}
	close(ch)
}

// traceReplayTask simulates storage operations from storage traces on stateDB.
func traceReplayTask(cfg *utils.Config) error {

	// starting reading in parallel
	log.Printf("Start reading operations in parallel")
	opChannel := make(chan operation.Operation, 100000)
	go readTrace(cfg, opChannel)

	// load dictionaries & indexes
	log.Printf("Load dictionaries")
	dCtx := dictionary.ReadContext()

	// create a directory for the store to place all its files, and
	// instantiate the state DB under testing.
	log.Printf("Create stateDB database")
	db, stateDirectory, loadedExistingDB, err := utils.PrepareStateDB(cfg)
	if err != nil {
		return err
	}
	if !cfg.KeepStateDB {
		log.Printf("WARNING: directory %v will be removed at the end of this run.\n", stateDirectory)
		defer os.RemoveAll(stateDirectory)
	}

	if cfg.SkipPriming || loadedExistingDB {
		log.Printf("Skipping DB priming.\n")
	} else {
		// intialize the world state and advance it to the first block
		log.Printf("Load and advance worldstate to block %v", cfg.First-1)
		ws, err := utils.GenerateWorldStateFromUpdateDB(cfg, cfg.First-1)
		if err != nil {
			return err
		}

		// prime stateDB
		log.Printf("Prime stateDB \n")
		utils.PrimeStateDB(ws, db, cfg)

		// print memory usage after priming
		if cfg.MemoryBreakdown {
			if usage := db.GetMemoryUsage(); usage != nil {
				log.Printf("State DB memory usage: %d byte\n%s\n", usage.UsedBytes, usage.Breakdown)
			} else {
				log.Printf("Utilized storage solution does not support memory breakdowns.\n")
			}
		}

		// delete destroyed accounts from stateDB
		log.Printf("Delete destroyed accounts \n")
		// remove destroyed accounts until one block before the first block
		if err = utils.DeleteDestroyedAccountsFromStateDB(db, cfg, cfg.First-1); err != nil {
			return err
		}
	}

	log.Printf("Replay storage operations on StateDB database")

	// progress message setup
	var (
		start      time.Time
		sec        float64
		lastSec    float64
		firstBlock = true
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

	var lastBlock uint64
	// replay storage trace
	for op := range opChannel {
		if beginBlock, ok := op.(*operation.BeginBlock); ok {
			block := beginBlock.BlockNumber

			// The first Epoch begin and the final EpochEnd need to be artificially
			// added since the range running on may not match epoch boundaries.
			if firstBlock {
				run(operation.NewBeginEpoch(cfg.First / cfg.EpochLength))
				firstBlock = false
			}

			if block > cfg.Last {
				run(operation.NewEndEpoch())
				break
			}
			lastBlock = block // track the last processed block
			if cfg.EnableProgress {
				// report progress
				sec = time.Since(start).Seconds()
				if sec-lastSec >= 15 {
					log.Printf("Elapsed time: %.0f s, at block %v\n", sec, block)
					lastSec = sec
				}
			}
		}
		run(op)
	}

	sec = time.Since(start).Seconds()

	log.Printf("Finished replaying storage operations on StateDB database")

	// validate stateDB
	if cfg.ValidateWorldState {
		log.Printf("Validate final state")
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
			log.Printf("State DB memory usage: %d byte\n%s\n", usage.UsedBytes, usage.Breakdown)
		} else {
			log.Printf("Utilized storage solution does not support memory breakdowns.\n")
		}
	}

	// write memory profile if requested
	if cfg.MemoryProfile != "" {
		f, err := os.Create(cfg.MemoryProfile)
		if err != nil {
			return fmt.Errorf("could not create memory profile: %s", err)
		}
		runtime.GC() // get up-to-date statistics
		if err := pprof.WriteHeapProfile(f); err != nil {
			return fmt.Errorf("could not write memory profile: %s", err)
		}
	}

	// print profile statistics (if enabled)
	if operation.EnableProfiling {
		operation.PrintProfiling()
	}

	if cfg.KeepStateDB {
		rootHash, _ := db.Commit(true)
		if err := utils.WriteStateDbInfo(stateDirectory, cfg, lastBlock, rootHash); err != nil {
			log.Println(err)
		}
		//rename directory after closing db.
		defer utils.RenameTempStateDBDirectory(cfg, stateDirectory, lastBlock)
	}

	// close the DB and print disk usage
	log.Printf("Close StateDB database")
	start = time.Now()
	if err := db.Close(); err != nil {
		log.Printf("Failed to close database: %v", err)
	}

	// print progress summary
	if cfg.EnableProgress {
		log.Printf("trace replay: Total elapsed time: %.3f s, processed %v blocks\n", sec, cfg.Last-cfg.First+1)
		log.Printf("trace replay: Closing DB took %v\n", time.Since(start))
		log.Printf("trace replay: Final disk usage: %v MiB\n", float32(utils.GetDirectorySize(stateDirectory))/float32(1024*1024))
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
	// set trace directory
	tracer.TraceDir = ctx.String(utils.TraceDirectoryFlag.Name) + "/"
	dictionary.ContextDir = ctx.String(utils.TraceDirectoryFlag.Name) + "/"

	// start CPU profiling if requested.
	if profileFileName := ctx.String(utils.CpuProfileFlag.Name); profileFileName != "" {
		f, err := os.Create(profileFileName)
		if err != nil {
			return fmt.Errorf("could not create CPU profile: %s", err)
		}
		if err := pprof.StartCPUProfile(f); err != nil {
			return fmt.Errorf("could not start CPU profile: %s", err)
		}
		defer pprof.StopCPUProfile()
	}

	// run storage driver
	substate.SetSubstateFlags(ctx)
	substate.OpenSubstateDBReadOnly()
	defer substate.CloseSubstateDB()
	err = traceReplayTask(cfg)

	return err
}
