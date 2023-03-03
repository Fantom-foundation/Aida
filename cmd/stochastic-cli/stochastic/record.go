package stochastic

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"os"
	"time"

	"github.com/Fantom-foundation/Aida/state"
	"github.com/Fantom-foundation/Aida/stochastic"
	"github.com/Fantom-foundation/Aida/utils"
	substate "github.com/Fantom-foundation/Substate"
	"github.com/urfave/cli/v2"
)

// StochasticRecordCommand data structure for the record app
var StochasticRecordCommand = cli.Command{
	Action:    stochasticRecordAction,
	Name:      "record",
	Usage:     "record StateDB events while processing blocks",
	ArgsUsage: "<blockNumFirst> <blockNumLast>",
	Flags: []cli.Flag{
		&utils.CpuProfileFlag,
		&utils.DisableProgressFlag,
		&utils.EpochLengthFlag,
		&utils.OutputFlag,
		&substate.WorkersFlag,
		&substate.SubstateDirFlag,
		&utils.ChainIDFlag,
	},
	Description: `
The stochastic record command requires two arguments:
<blockNumFirst> <blockNumLast>

<blockNumFirst> and <blockNumLast> are the first and
last block for recording events.`,
}

// stochasticRecordAction implements recording of events.
func stochasticRecordAction(ctx *cli.Context) error {
	substate.RecordReplay = true
	var err error

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

	if ctx.Bool(utils.TraceDebugFlag.Name) {
		utils.TraceDebug = true
	}

	// iterate through subsets in sequence
	substate.SetSubstateFlags(ctx)
	substate.OpenSubstateDBReadOnly()
	defer substate.CloseSubstateDB()
	iter := substate.NewSubstateIterator(cfg.First, ctx.Int(substate.WorkersFlag.Name))
	defer iter.Release()
	oldBlock := uint64(math.MaxUint64) // set to an infeasible block
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

	// create a new event registry
	eventRegistry := stochastic.NewEventRegistry()

	curEpoch := cfg.First / cfg.EpochLength
	eventRegistry.RegisterOp(stochastic.BeginEpochID)

	// iterate over all substates in order
	for iter.Next() {
		tx := iter.Value()
		// close off old block with an end-block operation
		if oldBlock != tx.Block {
			if tx.Block > cfg.Last {
				break
			}
			if oldBlock != math.MaxUint64 {
				eventRegistry.RegisterOp(stochastic.EndBlockID)
				newEpoch := tx.Block / cfg.EpochLength
				for curEpoch < newEpoch {
					eventRegistry.RegisterOp(stochastic.EndEpochID)
					curEpoch++
					eventRegistry.RegisterOp(stochastic.BeginEpochID)
				}
			}
			// open new block with a begin-block operation and clear index cache
			eventRegistry.RegisterOp(stochastic.BeginBlockID)
			oldBlock = tx.Block
		}
		eventRegistry.RegisterOp(stochastic.BeginTransactionID)

		var statedb state.StateDB
		statedb = state.MakeGethInMemoryStateDB(&tx.Substate.InputAlloc, tx.Block)
		statedb = stochastic.NewEventProxy(statedb, &eventRegistry)
		if err := utils.ProcessTx(statedb, cfg, tx.Block, tx.Transaction, tx.Substate); err != nil {
			return err
		}

		eventRegistry.RegisterOp(stochastic.EndTransactionID)
		if cfg.EnableProgress {
			// report progress
			sec = time.Since(start).Seconds()
			if sec-lastSec >= 15 {
				fmt.Printf("stochastic record: Elapsed time: %.0f s, at block %v\n", sec, oldBlock)
				lastSec = sec
			}
		}
	}
	// end last block
	if oldBlock != math.MaxUint64 {
		eventRegistry.RegisterOp(stochastic.EndBlockID)
	}
	eventRegistry.RegisterOp(stochastic.EndEpochID)

	if cfg.EnableProgress {
		sec = time.Since(start).Seconds()
		fmt.Printf("stochastic record: Total elapsed time: %.3f s, processed %v blocks\n", sec, cfg.Last-cfg.First+1)
	}

	// writing event registry
	fmt.Printf("stochastic record: write events file ...\n")
	outputFileName := ctx.String(utils.OutputFlag.Name)
	if outputFileName == "" {
		outputFileName = "./events.json"
	}
	WriteEvents(&eventRegistry, outputFileName)

	return err
}

// WriteEvent writes event file in JSON format.
func WriteEvents(r *stochastic.EventRegistry, filename string) {
	f, fErr := os.Create(filename)
	if fErr != nil {
		log.Fatalf("cannot open JSON file. Error: %v", fErr)
	}
	defer f.Close()

	jOut, jErr := json.MarshalIndent(r.NewEventRegistryJSON(), "", "    ")
	if jErr != nil {
		log.Fatalf("failed to convert JSON file. Error: %v", jErr)
	}

	_, pErr := fmt.Fprintln(f, string(jOut))
	if pErr != nil {
		log.Fatalf("failed to convert JSON file. Error: %v", pErr)
	}
}
