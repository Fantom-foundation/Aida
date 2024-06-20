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

package primer

import (
	"github.com/urfave/cli/v2"

	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/executor/extension/logger"
	"github.com/Fantom-foundation/Aida/executor/extension/primer"
	"github.com/Fantom-foundation/Aida/executor/extension/statedb"
	"github.com/Fantom-foundation/Aida/txcontext"
	"github.com/Fantom-foundation/Aida/utils"
)

// RunPrimer performs sequential block processing on a StateDb
func RunPrimer(ctx *cli.Context) error {
	cfg, err := utils.NewConfig(ctx, utils.LastBlockArg)
	if err != nil {
		return err
	}

	cfg.StateValidationMode = utils.SubsetCheck

	// Always keep db
	cfg.KeepDb = true

	// This is necessary to pass the check inside the priming exstension
	cfg.First = cfg.Last

	substateDb, err := executor.OpenSubstateDb(cfg, ctx)
	if err != nil {
		return err
	}
	defer substateDb.Close()

	return runPriming(cfg)
}

func runPriming(
	cfg *utils.Config,
) error {
	var extensionList = []executor.Extension[txcontext.TxContext]{
		logger.MakeDbLogger[txcontext.TxContext](cfg),
		statedb.MakeStateDbManager[txcontext.TxContext](cfg, ""),
	}

	extensionList = append(extensionList, []executor.Extension[txcontext.TxContext]{
		primer.MakeStateDbPrimer[txcontext.TxContext](cfg),
	}...,
	)

	return executor.RunUtilPrimer(
		executor.Params{
			To:                     int(cfg.Last),
			NumWorkers:             1, // vm-sdb can run only with one worker
			ParallelismGranularity: executor.BlockLevel,
		},
		extensionList)
}
