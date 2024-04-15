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

package main

import (
	"time"

	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/executor/extension/aidadb"
	"github.com/Fantom-foundation/Aida/executor/extension/logger"
	"github.com/Fantom-foundation/Aida/executor/extension/primer"
	"github.com/Fantom-foundation/Aida/executor/extension/profiler"
	"github.com/Fantom-foundation/Aida/executor/extension/register"
	"github.com/Fantom-foundation/Aida/executor/extension/statedb"
	"github.com/Fantom-foundation/Aida/executor/extension/tracker"
	"github.com/Fantom-foundation/Aida/executor/extension/validator"
	"github.com/Fantom-foundation/Aida/state"
	"github.com/Fantom-foundation/Aida/txcontext"
	"github.com/Fantom-foundation/Aida/utils"
	"github.com/urfave/cli/v2"
)

// RunSubstate performs sequential block processing on a StateDb
func RunSubstate(ctx *cli.Context) error {
	cfg, err := utils.NewConfig(ctx, utils.BlockRangeArgs)
	if err != nil {
		return err
	}

	cfg.StateValidationMode = utils.SubsetCheck

	substateDb, err := executor.OpenSubstateDb(cfg, ctx)
	if err != nil {
		return err
	}
	defer substateDb.Close()

	return runSubstates(cfg, substateDb, nil, executor.MakeLiveDbTxProcessor(cfg), nil)
}

func runSubstates(
	cfg *utils.Config,
	provider executor.Provider[txcontext.TxContext],
	stateDb state.StateDB,
	processor executor.Processor[txcontext.TxContext],
	extra []executor.Extension[txcontext.TxContext],
) error {
	// order of extensionList has to be maintained
	var extensionList = []executor.Extension[txcontext.TxContext]{
		profiler.MakeCpuProfiler[txcontext.TxContext](cfg),
		profiler.MakeDiagnosticServer[txcontext.TxContext](cfg),
	}

	if stateDb == nil {
		extensionList = append(
			extensionList,
			statedb.MakeStateDbManager[txcontext.TxContext](cfg, ""),
			statedb.MakeLiveDbBlockChecker[txcontext.TxContext](cfg),
			validator.MakeShadowDbValidator(cfg),
			logger.MakeDbLogger[txcontext.TxContext](cfg),
		)
	}

	extensionList = append(extensionList, extra...)

	extensionList = append(extensionList, []executor.Extension[txcontext.TxContext]{
		register.MakeRegisterProgress(cfg, 100_000),
		// RegisterProgress should be the as top-most as possible on the list
		// In this case, after StateDb is created.
		// Any error that happen in extension above it will not be correctly recorded.
		profiler.MakeThreadLocker[txcontext.TxContext](),
		aidadb.MakeAidaDbManager[txcontext.TxContext](cfg),
		profiler.MakeVirtualMachineStatisticsPrinter[txcontext.TxContext](cfg),
		logger.MakeProgressLogger[txcontext.TxContext](cfg, 15*time.Second),
		logger.MakeErrorLogger[txcontext.TxContext](cfg),
		tracker.MakeBlockProgressTracker(cfg, 100_000),
		primer.MakeStateDbPrimer[txcontext.TxContext](cfg),
		profiler.MakeMemoryUsagePrinter[txcontext.TxContext](cfg),
		profiler.MakeMemoryProfiler[txcontext.TxContext](cfg),
		statedb.MakeStateDbPrepper(),
		statedb.MakeArchiveInquirer(cfg),
		validator.MakeStateHashValidator[txcontext.TxContext](cfg),
		statedb.MakeBlockEventEmitter[txcontext.TxContext](),
		statedb.MakeTransactionEventEmitter[txcontext.TxContext](),
		validator.MakeLiveDbValidator(cfg, validator.ValidateTxTarget{WorldState: true, Receipt: true}),
		profiler.MakeOperationProfiler[txcontext.TxContext](cfg),

		// block profile extension should be always last because:
		// 1) Pre-Func are called forwards so this is called last and
		// 2) Post-Func are called backwards so this is called first
		// that means the gap between time measurements will be as small as possible
		profiler.MakeBlockRuntimeAndGasCollector(cfg),
	}...,
	)

	return executor.NewExecutor(provider, cfg.LogLevel).Run(
		executor.Params{
			From:                   int(cfg.First),
			To:                     int(cfg.Last) + 1,
			NumWorkers:             1, // vm-sdb can run only with one worker
			State:                  stateDb,
			ParallelismGranularity: executor.BlockLevel,
		},
		processor,
		extensionList,
	)
}
