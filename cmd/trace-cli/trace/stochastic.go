package trace

import (
	"fmt"
	"github.com/Fantom-foundation/Aida/tracer"
	"github.com/Fantom-foundation/Aida/tracer/dict"
	"github.com/Fantom-foundation/Aida/tracer/operation"
	"github.com/ethereum/go-ethereum/substate"
	"github.com/urfave/cli/v2"
	"io/ioutil"
	"log"
	"os"
	"runtime/pprof"
)

// StochasticCommand record frequencies of operations and distribution of their contract, storage, value accesses
var StochasticCommand = cli.Command{
	Action:    traceStochasticAction,
	Name:      "stochastic",
	Usage:     "executes stochastic distribution generation",
	ArgsUsage: "<blockNumFirst> <blockNumLast>",
	Flags: []cli.Flag{
		&disableProgressFlag,
		&substate.SubstateDirFlag,
		&substate.WorkersFlag,
		&traceDirectoryFlag,
		&traceDebugFlag,
		&stateDbImplementation,
		&stochasticMatrixFlag,
		&stochasticMatrixFormatFlag,
	},
	Description: `
The stochastic command requires two arguments:
<blockNumFirst> <blockNumLast>

<blockNumFirst> and <blockNumLast> are the first and
last block of the inclusive range of blocks to calculate distributions.`,
}

// traceStochasticTask simulates storage operations from storage traces and calculates distributions of operations.
func traceStochasticTask(cfg *TraceConfig) error {
	// load dictionaries & indexes
	dCtx := dict.ReadDictionaryContext()

	// intialize the world state and advance it to the first block
	log.Printf("Load and advance worldstate to block %v", cfg.first-1)
	ws, err := generateWorldState(cfg.worldStateDir, cfg.first-1, cfg.workers)
	if err != nil {
		return err
	}

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

	// prime stateDB
	log.Printf("Prime StateDB database with world-state")
	if cfg.impl == "memory" {
		db.PrepareSubstate(&ws)
	} else {
		primeStateDB(ws, db)
	}

	// initialize trace interator
	traceIter := tracer.NewTraceIterator(cfg.first, cfg.last)
	defer traceIter.Release()

	const firstOperation = 255
	prevOpId := byte(firstOperation)
	tFreq := map[[2]byte]uint64{}

	// replace storage trace
	fmt.Printf("trace replay: Replay storage operations\n")
	for traceIter.Next() {
		op := traceIter.Value()

		if op.GetId() == operation.BeginBlockID {
			block := op.(*operation.BeginBlock).BlockNumber
			if block > cfg.last {
				break
			}
		}

		if prevOpId != firstOperation {
			opId := op.GetId()
			tFreq[[2]byte{prevOpId, opId}]++
			prevOpId = opId
		} else {
			prevOpId = op.GetId()
		}

		operation.Execute(op, db, dCtx)
		if cfg.debug {
			operation.Debug(dCtx, op)
		}

	}

	dCtx.WriteDistributions()

	// print profile statistics (if enabled)
	if operation.EnableProfiling {
		operation.PrintProfiling()
	}

	// close the DB and print disk usage
	log.Printf("Close StateDB database")
	if err := db.Close(); err != nil {
		log.Printf("Failed to close database: %v", err)
	}

	// write stochastic matrix
	writeStochasticMatrix(stochasticMatrixFlag.Value, stochasticMatrixFormatFlag.Value, tFreq)

	fmt.Printf("trace stochastic: Done\n")
	return nil
}

func writeStochasticMatrix(smFile string, f string, tFreq map[[2]byte]uint64) {
	// write stochastic-matrix
	if f == "csv" {
		writeStochasticMatrixCsv(smFile, tFreq)
	} else {
		writeStochasticMatrixDot(smFile, tFreq)
	}
}

// writeStochasticMatrixCsv writes frequencies of operation chaining in csv format
func writeStochasticMatrixCsv(smFile string, tFreq map[[2]byte]uint64) {
	file, err := os.Create(smFile)
	if err != nil {
		log.Fatalf("Cannot open stochastic matrix file. Error: %v", err)
	}
	defer func() {
		err := file.Close()
		if err != nil {
			log.Fatalf("Cannot close stochastic matrix file. Error: %v", err)
		}
	}()

	for i := byte(0); i < operation.NumProfiledOperations; i++ {
		total := uint64(0)
		for j := byte(0); j < operation.NumProfiledOperations; j++ {
			total += tFreq[[2]byte{i, j}]
		}
		maxFreq := uint64(0)
		for j := byte(0); j < operation.NumProfiledOperations; j++ {
			if tFreq[[2]byte{i, j}] > maxFreq {
				maxFreq = tFreq[[2]byte{i, j}]
			}
		}
		for j := byte(0); j < operation.NumProfiledOperations; j++ {
			//fmt.Fprintf(file, "\t%v -> %v [%v],",
			//operation.GetLabel(i),
			//	operation.GetLabel(j),
			//	float64(tFreq[[2]byte{i, j}])/float64(total))

			var n float64
			if total == 0 {
				n = 0
			} else {
				n = float64(tFreq[[2]byte{i, j}]) / float64(total)
			}

			fmt.Fprintf(file, "%v", n)

			if j != operation.NumProfiledOperations-1 {
				fmt.Fprint(file, ",")
			}
		}
		fmt.Fprintf(file, "\n")
	}
}

// writeStochasticMatrixDot writes frequencies of operation chaining in dot format
func writeStochasticMatrixDot(smFile string, tFreq map[[2]byte]uint64) {
	file, err := os.Create(smFile)
	if err != nil {
		log.Fatalf("Cannot open stochastic matrix file. Error: %v", err)
	}
	defer func() {
		err := file.Close()
		if err != nil {
			log.Fatalf("Cannot close stochastic matrix file. Error: %v", err)
		}
	}()
	fmt.Fprintf(file, "digraph C {\n")
	for i := byte(0); i < operation.NumProfiledOperations; i++ {
		total := uint64(0)
		for j := byte(0); j < operation.NumProfiledOperations; j++ {
			total += tFreq[[2]byte{i, j}]
		}
		maxFreq := uint64(0)
		for j := byte(0); j < operation.NumProfiledOperations; j++ {
			if tFreq[[2]byte{i, j}] > maxFreq {
				maxFreq = tFreq[[2]byte{i, j}]
			}
		}
		for j := byte(0); j < operation.NumProfiledOperations; j++ {
			if tFreq[[2]byte{i, j}] != 0 {
				if tFreq[[2]byte{i, j}] != maxFreq {
					fmt.Fprintf(file, "\t%v -> %v [label=\"%v\"]\n",
						operation.GetLabel(i),
						operation.GetLabel(j),
						float64(tFreq[[2]byte{i, j}])/float64(total))
				} else {
					fmt.Fprintf(file, "\t%v -> %v [label=\"%v\", color=red]\n",
						operation.GetLabel(i),
						operation.GetLabel(j),
						float64(tFreq[[2]byte{i, j}])/float64(total))
				}
			}
		}
	}
	fmt.Fprintf(file, "}\n")
}

// traceReplayAction implements trace command for replaying.
func traceStochasticAction(ctx *cli.Context) error {
	var err error
	cfg, err := NewTraceConfig(ctx)
	if err != nil {
		return err
	}

	operation.EnableProfiling = ctx.Bool(profileFlag.Name)
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
	err = traceStochasticTask(cfg)

	return err
}
