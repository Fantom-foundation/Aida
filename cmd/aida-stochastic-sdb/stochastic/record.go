// Copyright 2024 Fantom Foundation
// This file is part of Aida Testing Infrastructure for Sonic
//
// Aida is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Aida is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with Aida. If not, see <http://www.gnu.org/licenses/>.

package stochastic

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"time"

	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/state"
	"github.com/Fantom-foundation/Aida/stochastic"
	substatecontext "github.com/Fantom-foundation/Aida/txcontext/substate"
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
		&utils.SyncPeriodLengthFlag,
		&utils.OutputFlag,
		&substate.WorkersFlag,
		&utils.ChainIDFlag,
		&utils.AidaDbFlag,
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
	// force enable transaction validation
	cfg.ValidateTxState = true

	// start CPU profiling if enabled.
	if err := utils.StartCPUProfile(cfg); err != nil {
		return err
	}
	defer utils.StopCPUProfile(cfg)

	processor := executor.MakeLiveDbTxProcessor(cfg)

	// iterate through subsets in sequence
	substate.SetSubstateDb(cfg.AidaDb)
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
	start = time.Now()
	sec = time.Since(start).Seconds()
	lastSec = time.Since(start).Seconds()

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
		statedb = state.MakeInMemoryStateDB(substatecontext.NewWorldState(tx.Substate.InputAlloc), tx.Block)
		statedb = stochastic.NewEventProxy(statedb, &eventRegistry)
		if _, err = processor.ProcessTransaction(statedb, int(tx.Block), tx.Transaction, substatecontext.NewTxContext(tx.Substate)); err != nil {
			return err
		}

		// report progress
		sec = time.Since(start).Seconds()
		if sec-lastSec >= 15 {
			fmt.Printf("stochastic record: Elapsed time: %.0f s, at block %v\n", sec, oldBlock)
			lastSec = sec
		}
	}
	// end last block
	if oldBlock != math.MaxUint64 {
		eventRegistry.RegisterOp(stochastic.EndBlockID)
	}
	eventRegistry.RegisterOp(stochastic.EndSyncPeriodID)

	sec = time.Since(start).Seconds()
	fmt.Printf("stochastic record: Total elapsed time: %.3f s, processed %v blocks\n", sec, cfg.Last-cfg.First+1)

	// writing event registry
	fmt.Printf("stochastic record: write events file ...\n")
	if cfg.Output == "" {
		cfg.Output = "./events.json"
	}
	err = WriteEvents(&eventRegistry, cfg.Output)
	if err != nil {
		return err
	}

	return nil
}

// WriteEvents writes event file in JSON format.
func WriteEvents(r *stochastic.EventRegistry, filename string) error {
	f, fErr := os.Create(filename)
	if fErr != nil {
		return fmt.Errorf("cannot open JSON file; %v", fErr)
	}
	defer f.Close()

	jOut, jErr := json.MarshalIndent(r.NewEventRegistryJSON(), "", "    ")
	if jErr != nil {
		return fmt.Errorf("failed to convert JSON file; %v", jErr)
	}

	_, pErr := fmt.Fprintln(f, string(jOut))
	if pErr != nil {
		return fmt.Errorf("failed to convert JSON file; %v", pErr)
	}

	return nil
}
