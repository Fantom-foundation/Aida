package trace

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"runtime/pprof"
	"time"

	"github.com/Fantom-foundation/Aida/tracer"
	"github.com/Fantom-foundation/Aida/tracer/dict"
	"github.com/Fantom-foundation/Aida/tracer/operation"
	"github.com/ethereum/go-ethereum/substate"
	"github.com/urfave/cli/v2"
)

// TraceReplayCommand data structure for the replay app
var TraceReplayCommand = cli.Command{
	Action:    traceReplayAction,
	Name:      "replay",
	Usage:     "executes storage trace",
	ArgsUsage: "<blockNumFirst> <blockNumLast>",
	Flags: []cli.Flag{
		&cpuProfileFlag,
		&epochLengthFlag,
		&disableProgressFlag,
		&primeSeedFlag,
		&primeCommitThresholdFlag,
		&profileFlag,
		&randomizePrimingFlag,
		&stateDbImplementation,
		&stateDbVariant,
		&substate.SubstateDirFlag,
		&substate.WorkersFlag,
		&traceDirectoryFlag,
		&traceDebugFlag,
		&updateDBDirFlag,
		&validateEndState,
	},
	Description: `
The trace replay command requires two arguments:
<blockNumFirst> <blockNumLast>

<blockNumFirst> and <blockNumLast> are the first and
last block of the inclusive range of blocks to replay storage traces.`,
}

// readTrace reads operations from trace files and puts them into a channel.
func readTrace(cfg *TraceConfig, ch chan operation.Operation) {
	traceIter := tracer.NewTraceIterator(cfg.first, cfg.last)
	defer traceIter.Release()
	for traceIter.Next() {
		op := traceIter.Value()
		ch <- op
	}
	close(ch)
}

// traceReplayTask simulates storage operations from storage traces on stateDB.
func traceReplayTask(cfg *TraceConfig) error {

	// starting reading in parallel
	log.Printf("Start reading operations in parallel")
	opChannel := make(chan operation.Operation, 100000)
	go readTrace(cfg, opChannel)

	// load dictionaries & indexes
	log.Printf("Load dictionaries")
	dCtx := dict.ReadDictionaryContext()

	// create a directory for the store to place all its files, and
	// instantiate the state DB under testing.
	log.Printf("Create stateDB database")
	stateDirectory, err := ioutil.TempDir("", "state_db_*")
	if err != nil {
		return err
	}
	db, err := makeStateDB(stateDirectory, cfg.impl, cfg.variant)
	if err != nil {
		return err
	}

	// intialize the world state and advance it to the first block
	log.Printf("Load and advance worldstate to block %v", cfg.first-1)
	ws, err := generateWorldStateFromUpdateDB(cfg.updateDBDir, cfg.first-1, cfg.workers)
	if err != nil {
		return err
	}

	// prime stateDB
	primeStateDB(ws, db, cfg.primeRandom, cfg.primeSeed, cfg.primeThreshold)

	log.Printf("Replay storage operations on StateDB database")

	// progress message setup
	var (
		start      time.Time
		sec        float64
		lastSec    float64
		firstBlock = true
	)
	if cfg.enableProgress {
		start = time.Now()
		sec = time.Since(start).Seconds()
		lastSec = time.Since(start).Seconds()
	}

	// A utility to run operations on the local context.
	run := func(op operation.Operation) {
		operation.Execute(op, db, dCtx)
		if cfg.debug {
			operation.Debug(dCtx, op)
		}
	}

	// replay storage trace
	for op := range opChannel {
		if beginBlock, ok := op.(*operation.BeginBlock); ok {
			block := beginBlock.BlockNumber

			// The first Epoch begin and the final EpochEnd need to be artificially
			// added since the range running on may not match epoch boundaries.
			if firstBlock {
				run(operation.NewBeginEpoch())
				firstBlock = false
			}

			if block > cfg.last {
				run(operation.NewEndEpoch(cfg.last / cfg.epochLength))
				break
			}
			if cfg.enableProgress {
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
	if cfg.enableValidation {
		log.Printf("Validate final state")
		// advance the world state from the first block to the last block
		advanceWorldState(ws, cfg.first, cfg.last, cfg.workers)
		if err := validateStateDB(ws, db); err != nil {
			return fmt.Errorf("Validation failed. %v\n", err)
		}
	}

	// print profile statistics (if enabled)
	if operation.EnableProfiling {
		operation.PrintProfiling()
	}

	// close the DB and print disk usage
	log.Printf("Close StateDB database")
	start = time.Now()
	if err := db.Close(); err != nil {
		log.Printf("Failed to close database: %v", err)
	}

	// print progress summary
	if cfg.enableProgress {
		log.Printf("trace replay: Total elapsed time: %.3f s, processed %v blocks\n", sec, cfg.last-cfg.first+1)
		log.Printf("trace replay: Closing DB took %v\n", time.Since(start))
		log.Printf("trace replay: Final disk usage: %v MiB\n", float32(getDirectorySize(stateDirectory))/float32(1024*1024))
	}

	return nil
}

// traceReplayAction implements trace command for replaying.
func traceReplayAction(ctx *cli.Context) error {
	var err error
	cfg, err := NewTraceConfig(ctx)
	if err != nil {
		return err
	}
	if cfg.impl == "memory" {
		return fmt.Errorf("db-impl memory is not supported")
	}

	operation.EnableProfiling = cfg.profile
	// set trace directory
	tracer.TraceDir = ctx.String(traceDirectoryFlag.Name) + "/"
	dict.DictionaryContextDir = ctx.String(traceDirectoryFlag.Name) + "/"

	// start CPU profiling if requested.
	if profileFileName := ctx.String(cpuProfileFlag.Name); profileFileName != "" {
		f, err := os.Create(profileFileName)
		if err != nil {
			return err
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	// run storage driver
	substate.SetSubstateFlags(ctx)
	substate.OpenSubstateDBReadOnly()
	defer substate.CloseSubstateDB()
	err = traceReplayTask(cfg)

	return err
}
