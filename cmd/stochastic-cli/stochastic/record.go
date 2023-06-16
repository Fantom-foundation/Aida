package stochastic

import (
	"math"
	"time"

	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/state"
	"github.com/Fantom-foundation/Aida/stochastic"
	"github.com/Fantom-foundation/Aida/utils"
	substate "github.com/Fantom-foundation/Substate"
	"github.com/urfave/cli/v2"
)

// StochasticRecordCommand data structure for the record app.
var StochasticRecordCommand = cli.Command{
	Action:    stochasticRecordAction,
	Name:      "record",
	Usage:     "record StateDB events while processing blocks",
	ArgsUsage: "<blockNumFirst> <blockNumLast>",
	Flags: []cli.Flag{
		&utils.CpuProfileFlag,
		&utils.QuietFlag,
		&utils.SyncPeriodLengthFlag,
		&utils.OutputFlag,
		&substate.WorkersFlag,
		&substate.SubstateDbFlag,
		&utils.ChainIDFlag,
		&utils.AidaDbFlag,
	},
	Description: `
The stochastic record command requires two arguments:
<blockNumFirst> <blockNumLast>

<blockNumFirst> and <blockNumLast> are the first and
last block for recording events.`,
}

// stochasticRecordAction implements recording of events by running the EVM for a given blockrange.
func stochasticRecordAction(ctx *cli.Context) error {
	substate.RecordReplay = true
	var err error

	// process configuration
	cfg, err := utils.NewConfig(ctx, utils.BlockRangeArgs)
	if err != nil {
		return err
	}
	cfg.ValidateTxState = true // force enable transaction validation
	log := logger.NewLogger(cfg.LogLevel, "StochasticRecord")

	// start CPU profiling if enabled.
	if err := utils.StartCPUProfile(cfg); err != nil {
		return err
	}
	defer utils.StopCPUProfile(cfg)

	// iterate through subsets in sequence
	substate.SetSubstateDb(cfg.SubstateDb)
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
	if !cfg.Quiet {
		start = time.Now()
		sec = time.Since(start).Seconds()
		lastSec = time.Since(start).Seconds()
	}

	// create a new event registry
	eventRegistry := stochastic.NewEventRegistry()

	curSyncPeriod := cfg.First / cfg.SyncPeriodLength
	eventRegistry.RegisterOp(stochastic.BeginSyncPeriodID)

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
				newSyncPeriod := tx.Block / cfg.SyncPeriodLength
				for curSyncPeriod < newSyncPeriod {
					eventRegistry.RegisterOp(stochastic.EndSyncPeriodID)
					curSyncPeriod++
					eventRegistry.RegisterOp(stochastic.BeginSyncPeriodID)
				}
			}
			// open new block with a begin-block operation and clear index cache
			eventRegistry.RegisterOp(stochastic.BeginBlockID)
			oldBlock = tx.Block
		}

		var statedb state.StateDB
		statedb = state.MakeGethInMemoryStateDB(&tx.Substate.InputAlloc, tx.Block)
		statedb = stochastic.NewEventProxy(statedb, &eventRegistry)
		if err := utils.ProcessTx(statedb, cfg, tx.Block, tx.Transaction, tx.Substate); err != nil {
			return err
		}

		if !cfg.Quiet {
			// report progress
			sec = time.Since(start).Seconds()
			if sec-lastSec >= 15 {
				log.Infof("Elapsed time: %.0f s, at block %n", sec, oldBlock)
				lastSec = sec
			}
		}
	}
	// end last block
	if oldBlock != math.MaxUint64 {
		eventRegistry.RegisterOp(stochastic.EndBlockID)
	}
	eventRegistry.RegisterOp(stochastic.EndSyncPeriodID)

	if !cfg.Quiet {
		sec = time.Since(start).Seconds()
		log.Noticef("Total elapsed time: %.3f s, processed %v blocks", sec, cfg.Last-cfg.First+1)
	}

	// writing event registry in JSON format
	if cfg.Output == "" {
		cfg.Output = "./events.json"
	}
	log.Noticef("Write events file %v", cfg.Output)
	err = eventRegistry.WriteJSON(cfg.Output)
	if err != nil {
		return err
	}

	return nil
}
