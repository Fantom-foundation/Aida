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
	"time"

	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/executor/extension/logger"
	"github.com/Fantom-foundation/Aida/executor/extension/profiler"
	"github.com/Fantom-foundation/Aida/executor/extension/register"
	"github.com/Fantom-foundation/Aida/executor/extension/statedb"
	"github.com/Fantom-foundation/Aida/executor/extension/tracker"
	"github.com/Fantom-foundation/Aida/executor/extension/validator"
	"github.com/Fantom-foundation/Aida/rpc"
	"github.com/Fantom-foundation/Aida/state"
	"github.com/Fantom-foundation/Aida/utils"
	"github.com/urfave/cli/v2"
)

const (
	rpcDefaultProgressReportFrequency = 100_000
)

func RunRpc(ctx *cli.Context) error {
	cfg, err := utils.NewConfig(ctx, utils.BlockRangeArgs)
	if err != nil {
		return err
	}

	cfg.SetStateDbSrcReadOnly()

	rpcSource, err := executor.OpenRpcRecording(ctx.Context, cfg)
	if err != nil {
		return err
	}

	defer rpcSource.Close()

	return run(cfg, rpcSource, nil, makeRpcProcessor(cfg), nil)
}

func makeRpcProcessor(cfg *utils.Config) rpcProcessor {
	return rpcProcessor{
		cfg: cfg,
	}
}

type rpcProcessor struct {
	cfg *utils.Config
}

func (p rpcProcessor) Process(state executor.State[*rpc.RequestAndResults], ctx *executor.Context) error {
	var err error
	ctx.ExecutionResult, err = rpc.Execute(uint64(state.Block), state.Data, ctx.Archive, p.cfg)
	if err != nil {
		return err
	}
	return nil
}

func run(
	cfg *utils.Config,
	provider executor.Provider[*rpc.RequestAndResults],
	stateDb state.StateDB,
	processor executor.Processor[*rpc.RequestAndResults],
	extra []executor.Extension[*rpc.RequestAndResults],

) error {
	var extensionList = []executor.Extension[*rpc.RequestAndResults]{
		// RegisterProgress should be the first on the list = last to receive PostRun.
		// This is because it collects the error and records it externally.
		// If not, error that happen afterwards (e.g. on top of) will not be correctly recorded.
		register.MakeRegisterRequestProgress(cfg,
			rpcDefaultProgressReportFrequency,
			register.OnPreBlock,
		),

		profiler.MakeCpuProfiler[*rpc.RequestAndResults](cfg),
		logger.MakeProgressLogger[*rpc.RequestAndResults](cfg, 15*time.Second),
		logger.MakeErrorLogger[*rpc.RequestAndResults](cfg),
		tracker.MakeRequestProgressTracker(cfg, 100_000),
		statedb.MakeTemporaryArchivePrepper(),
		validator.MakeRpcComparator(cfg),
	}

	// this is for testing purposes so mock statedb and mock extension can be used
	extensionList = append(extensionList, extra...)
	if stateDb == nil {
		extensionList = append(
			extensionList,
			statedb.MakeStateDbManager[*rpc.RequestAndResults](cfg, ""),
			statedb.MakeArchiveBlockChecker[*rpc.RequestAndResults](cfg),
			logger.MakeDbLogger[*rpc.RequestAndResults](cfg),
		)

	}

	return executor.NewExecutor(provider, cfg.LogLevel).Run(
		executor.Params{
			From:                   int(cfg.First),
			To:                     int(cfg.Last) + 1,
			NumWorkers:             cfg.Workers,
			ParallelismGranularity: executor.TransactionLevel,
			State:                  stateDb,
		},
		processor,
		extensionList,
		nil,
	)
}
