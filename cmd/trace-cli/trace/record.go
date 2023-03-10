package trace

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"math"
	"os"
	"runtime"
	"time"

	"github.com/Fantom-foundation/Aida/state"
	"github.com/Fantom-foundation/Aida/tracer"
	"github.com/Fantom-foundation/Aida/tracer/dictionary"
	"github.com/Fantom-foundation/Aida/tracer/operation"
	"github.com/Fantom-foundation/Aida/utils"
	substate "github.com/Fantom-foundation/Substate"
	"github.com/dsnet/compress/bzip2"
	"github.com/urfave/cli/v2"
)

const (
	WriteBufferSize  = 1048576 // Size of write buffer for writing trace file.
	WriteChannelSize = 1000    // Max. channel size for serialising StateDB operations.
)

// TraceRecordCommand data structure for the record app
var TraceRecordCommand = cli.Command{
	Action:    traceRecordAction,
	Name:      "record",
	Usage:     "captures and records StateDB operations while processing blocks",
	ArgsUsage: "<blockNumFirst> <blockNumLast>",
	Flags: []cli.Flag{
		&utils.CpuProfileFlag,
		&utils.EpochLengthFlag,
		&utils.DisableProgressFlag,
		&substate.WorkersFlag,
		&substate.SubstateDirFlag,
		&utils.ChainIDFlag,
		&utils.TraceDirectoryFlag,
		&utils.TraceDebugFlag,
		&utils.DBFlag,
	},
	Description: `
The trace record command requires two arguments:
<blockNumFirst> <blockNumLast>
<blockNumFirst> and <blockNumLast> are the first and
last block of the inclusive range of blocks to trace transactions.`,
}

// writeOperation writes operation to file.
func writeOperation(dCtx *dictionary.Context, zFile io.Writer, op operation.Operation) {
	operation.Write(zFile, op)
	if utils.TraceDebug {
		operation.Debug(dCtx, op)
	}
}

// traceRecordAction implements trace command for recording.
func traceRecordAction(ctx *cli.Context) error {
	substate.RecordReplay = true

	cfg, err := utils.NewConfig(ctx, utils.BlockRangeArgs)
	if err != nil {
		return err
	}
	// force enable tracsaction validation
	cfg.ValidateTxState = true

	// start CPU profiling if enabled.
	if err := utils.StartCPUProfile(cfg); err != nil {
		return err
	}
	defer utils.StopCPUProfile(cfg)

	// create dictionary and index contexts
	dCtx := dictionary.NewContext()

	// process arguments
	tracer.TraceDir = ctx.String(utils.TraceDirectoryFlag.Name) + "/"
	dictionary.ContextDir = ctx.String(utils.TraceDirectoryFlag.Name) + "/"
	if ctx.Bool(utils.TraceDebugFlag.Name) {
		utils.TraceDebug = true
	}

	// iterate through subsets in sequence
	substate.SetSubstateFlags(cfg.SubstateDBDir)
	substate.OpenSubstateDBReadOnly()
	// close substate
	defer substate.CloseSubstateDB()

	// open trace file, write buffer, and compressed stream
	file, err := os.OpenFile(tracer.TraceDir+"trace.dat", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		log.Fatalf("Cannot open trace file. Error: %v", err)
	}
	bFile := bufio.NewWriterSize(file, WriteBufferSize)
	zFile, err := bzip2.NewWriter(bFile, &bzip2.WriterConfig{Level: 9})
	if err != nil {
		log.Fatalf("Cannot open bzip2 stream. Error: %v", err)
	}
	defer func() {
		// closing compressed stream, flushing buffer, and closing trace file
		if err := zFile.Close(); err != nil {
			log.Fatalf("Cannot close bzip2 writer. Error: %v", err)
		}
		if err := bFile.Flush(); err != nil {
			log.Fatalf("Cannot flush buffer. Error: %v", err)
		}
		if err := file.Close(); err != nil {
			log.Fatalf("Cannot close trace file. Error: %v", err)
		}
	}()

	iter := substate.NewSubstateIterator(cfg.First, cfg.Workers)

	// release iterator
	defer iter.Release()

	curBlock := uint64(math.MaxUint64) // set to an infeasible block
	var (
		start   time.Time
		sec     float64
		lastSec float64
	)
	if cfg.EnableProgress {
		start = time.Now()
		sec = time.Since(start).Seconds()
		lastSec = time.Since(start).Seconds()
	}

	curEpoch := cfg.First / cfg.EpochLength
	writeOperation(dCtx, zFile, operation.NewBeginEpoch(curEpoch))

	for iter.Next() {
		tx := iter.Value()
		// close off old block with an end-block operation
		if curBlock != tx.Block {
			if tx.Block > cfg.Last {
				break
			}
			if curBlock != math.MaxUint64 {
				writeOperation(dCtx, zFile, operation.NewEndBlock())
				// Record epoch changes.
				newEpoch := tx.Block / cfg.EpochLength
				for curEpoch < newEpoch {
					writeOperation(dCtx, zFile, operation.NewEndEpoch())
					curEpoch++
					writeOperation(dCtx, zFile, operation.NewBeginEpoch(curEpoch))
				}
			}
			curBlock = tx.Block
			// open new block with a begin-block operation and clear index cache
			writeOperation(dCtx, zFile, operation.NewBeginBlock(tx.Block))
		}
		writeOperation(dCtx, zFile, operation.NewBeginTransaction(uint32(tx.Transaction)))
		var statedb state.StateDB
		statedb = state.MakeGethInMemoryStateDB(&tx.Substate.InputAlloc, tx.Block)
		statedb = NewProxyRecorder(statedb, dCtx, zFile, utils.TraceDebug)

		if err := utils.ProcessTx(statedb, cfg, tx.Block, tx.Transaction, tx.Substate); err != nil {
			return fmt.Errorf("Failed to process block %v tx %v. %v", tx.Block, tx.Transaction, err)
		}
		writeOperation(dCtx, zFile, operation.NewEndTransaction())
		if cfg.EnableProgress {
			// report progress
			sec = time.Since(start).Seconds()
			if sec-lastSec >= 15 {
				fmt.Printf("trace record: Elapsed time: %.0f s, at block %v\n", sec, curBlock)
				lastSec = sec
			}
		}

	}

	// end last block
	if curBlock != math.MaxUint64 {
		writeOperation(dCtx, zFile, operation.NewEndBlock())
	}
	writeOperation(dCtx, zFile, operation.NewEndEpoch())

	if cfg.EnableProgress {
		sec = time.Since(start).Seconds()
		fmt.Printf("trace record: Total elapsed time: %.3f s, processed %v blocks\n", sec, cfg.Last-cfg.First+1)
	}

	// write dictionaries and indexes
	dCtx.Write()

	// wait for writer thread
	runtime.Gosched()

	return nil
}
