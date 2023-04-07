package trace

import (
	"fmt"
	"math"
	"time"

	"github.com/Fantom-foundation/Aida/state"
	"github.com/Fantom-foundation/Aida/tracer"
	"github.com/Fantom-foundation/Aida/tracer/context"
	"github.com/Fantom-foundation/Aida/tracer/operation"
	"github.com/Fantom-foundation/Aida/utils"
	substate "github.com/Fantom-foundation/Substate"
	"github.com/urfave/cli/v2"
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
		&utils.TraceFileFlag,
		&utils.TraceDebugFlag,
		&utils.DebugFromFlag,
		&utils.DBFlag,
	},
	Description: `
The trace record command requires two arguments:
<blockNumFirst> <blockNumLast>
<blockNumFirst> and <blockNumLast> are the first and
last block of the inclusive range of blocks to trace transactions.`,
}

// traceRecordAction implements trace command for recording.
func traceRecordAction(ctx *cli.Context) error {
	substate.RecordReplay = true

	cfg, err := utils.NewConfig(ctx, utils.BlockRangeArgs)
	if err != nil {
		return err
	}
	// force enable transaction validation
	cfg.ValidateTxState = true

	// start CPU profiling if enabled.
	if err := utils.StartCPUProfile(cfg); err != nil {
		return err
	}
	defer utils.StopCPUProfile(cfg)

	// create record context
	rCtx := context.NewRecord(cfg.TraceFile)
	defer rCtx.Close()

	// open SubstateDB and create an substate iterator
	substate.SetSubstateDirectory(cfg.SubstateDBDir)
	substate.OpenSubstateDBReadOnly()
	defer substate.CloseSubstateDB()
	iter := substate.NewSubstateIterator(cfg.First, cfg.Workers)
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
	rCtx.Debug = cfg.Debug && (cfg.First >= cfg.DebugFrom)
	operation.WriteOp(rCtx, operation.NewBeginEpoch(curEpoch))
	for iter.Next() {
		tx := iter.Value()
		if !rCtx.Debug {
			rCtx.Debug = cfg.Debug && (tx.Block >= cfg.DebugFrom)
		}
		// close off old block with an end-block operation
		if curBlock != tx.Block {
			if tx.Block > cfg.Last {
				break
			}
			if curBlock != math.MaxUint64 {
				operation.WriteOp(rCtx, operation.NewEndBlock())
				// Record epoch changes.
				newEpoch := tx.Block / cfg.EpochLength
				for curEpoch < newEpoch {
					operation.WriteOp(rCtx, operation.NewEndEpoch())
					curEpoch++
					operation.WriteOp(rCtx, operation.NewBeginEpoch(curEpoch))
				}
			}
			curBlock = tx.Block
			// open new block with a begin-block operation and clear index cache
			operation.WriteOp(rCtx, operation.NewBeginBlock(tx.Block))
		}
		operation.WriteOp(rCtx, operation.NewBeginTransaction(uint32(tx.Transaction)))
		var statedb state.StateDB
		statedb = state.MakeGethInMemoryStateDB(&tx.Substate.InputAlloc, tx.Block)
		statedb = tracer.NewProxyRecorder(statedb, rCtx)

		if err := utils.ProcessTx(statedb, cfg, tx.Block, tx.Transaction, tx.Substate); err != nil {
			return fmt.Errorf("Failed to process block %v tx %v. %v", tx.Block, tx.Transaction, err)
		}
		operation.WriteOp(rCtx, operation.NewEndTransaction())
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
		operation.WriteOp(rCtx, operation.NewEndBlock())
	}
	operation.WriteOp(rCtx, operation.NewEndEpoch())

	if cfg.EnableProgress {
		sec = time.Since(start).Seconds()
		fmt.Printf("trace record: Total elapsed time: %.3f s, processed %v blocks\n", sec, cfg.Last-cfg.First+1)
	}

	return nil
}
