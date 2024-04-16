// Copyright 2024 Fantom Foundation
// This file is part of Aida Testing Infrastructure for Sonic
//
// Aida is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Aida is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with Aida. If not, see <http://www.gnu.org/licenses/>.

package main

import (
	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/executor/extension/profiler"
	"github.com/Fantom-foundation/Aida/executor/extension/statedb"
	"github.com/Fantom-foundation/Aida/executor/extension/tracker"
	"github.com/Fantom-foundation/Aida/executor/extension/validator"
	log "github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/txcontext"
	"github.com/Fantom-foundation/Aida/utils"
	substate "github.com/Fantom-foundation/Substate"
	"github.com/urfave/cli/v2"
)

// RecordCommand data structure for the record app
var RecordCommand = cli.Command{
	Action:    RecordStateDbTrace,
	Name:      "record",
	Usage:     "captures and records StateDB operations while processing blocks",
	ArgsUsage: "<blockNumFirst> <blockNumLast>",
	Flags: []cli.Flag{
		&utils.UpdateBufferSizeFlag,
		&utils.CpuProfileFlag,
		&utils.SyncPeriodLengthFlag,
		&substate.WorkersFlag,
		&utils.ChainIDFlag,
		&utils.TraceFileFlag,
		&utils.TraceDebugFlag,
		&utils.DebugFromFlag,
		&utils.AidaDbFlag,
		&log.LogLevelFlag,
	},
	Description: `
The trace record command requires two arguments:
<blockNumFirst> <blockNumLast>
<blockNumFirst> and <blockNumLast> are the first and
last block of the inclusive range of blocks to trace transactions.`,
}

func RecordStateDbTrace(ctx *cli.Context) error {
	cfg, err := utils.NewConfig(ctx, utils.BlockRangeArgs)
	if err != nil {
		return err
	}

	// force enable transaction validation
	cfg.ValidateTxState = true

	substate.RecordReplay = true
	substateDb, err := executor.OpenSubstateDb(cfg, ctx)
	if err != nil {
		return err
	}
	defer substateDb.Close()

	return record(cfg, substateDb, executor.MakeLiveDbTxProcessor(cfg), nil)
}

func record(
	cfg *utils.Config,
	provider executor.Provider[txcontext.TxContext],
	processor executor.Processor[txcontext.TxContext],
	extra []executor.Extension[txcontext.TxContext],
) error {
	var extensions = []executor.Extension[txcontext.TxContext]{
		profiler.MakeCpuProfiler[txcontext.TxContext](cfg),
		tracker.MakeBlockProgressTracker(cfg, 0),
		statedb.MakeTemporaryStatePrepper(cfg),
		statedb.MakeProxyRecorderPrepper[txcontext.TxContext](cfg),
		validator.MakeLiveDbValidator(cfg, validator.ValidateTxTarget{WorldState: true, Receipt: true}),
		statedb.MakeTransactionEventEmitter[txcontext.TxContext](),
	}

	extensions = append(extensions, extra...)

	return executor.NewExecutor(provider, cfg.LogLevel).Run(
		executor.Params{
			From:                   int(cfg.First),
			To:                     int(cfg.Last) + 1,
			ParallelismGranularity: executor.TransactionLevel,
		},
		processor,
		extensions,
	)
}
