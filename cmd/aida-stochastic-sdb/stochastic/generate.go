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
	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/stochastic"
	"github.com/Fantom-foundation/Aida/utils"
	"github.com/urfave/cli/v2"
)

// StochasticGenerateCommand data structure for the record app.
var StochasticGenerateCommand = cli.Command{
	Action:    stochasticGenerateAction,
	Name:      "generate",
	Usage:     "generate uniform events file",
	ArgsUsage: "",
	Flags: []cli.Flag{
		&logger.LogLevelFlag,
		&utils.BlockLengthFlag,
		&utils.SyncPeriodLengthFlag,
		&utils.TransactionLengthFlag,
		&utils.ContractNumberFlag,
		&utils.KeysNumberFlag,
		&utils.ValuesNumberFlag,
		&utils.SnapshotDepthFlag,
	},
	Description: "The stochastic produces an events.json file with uniform parameters",
}

// stochasticGenerateAction produces an event file with uniform parameters.
func stochasticGenerateAction(ctx *cli.Context) error {
	cfg, err := utils.NewConfig(ctx, utils.NoArgs)
	if err != nil {
		return err
	}
	log := logger.NewLogger(cfg.LogLevel, "StochasticGenerate")

	// create a new uniformly distributed event registry
	log.Info("Produce uniform stochastic events")
	eventRegistry := stochastic.GenerateUniformRegistry(cfg, log)

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
