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

package stochastic

import (
	"fmt"

	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/stochastic"
	"github.com/Fantom-foundation/Aida/utils"
	"github.com/urfave/cli/v2"
)

// StochasticEstimateCommand data structure for the estimator app.
var StochasticEstimateCommand = cli.Command{
	Action:    stochasticEstimateAction,
	Name:      "estimate",
	Usage:     "estimates parameters of access distributions and produces a simulation file",
	ArgsUsage: "<event-file>",
	Description: `
The stochastic estimator command requires one argument:
<events.json>

<events.json> is the event file produced by the stochastic recorder.`,
}

// stochasticEstimateAction implements estimator command for computing statistical parameters.
func stochasticEstimateAction(ctx *cli.Context) error {
	log := logger.NewLogger("INFO", "StochasticEstimate")

	// parse arguments
	if ctx.Args().Len() != 1 {
		return fmt.Errorf("missing events file")
	}
	inputFileName := ctx.Args().Get(0)

	// read event file in JSON format
	log.Infof("Read events file %v", inputFileName)
	eventRegistryJSON, err := stochastic.ReadEvents(inputFileName)
	if err != nil {
		return err
	}

	// estimate parameters
	log.Info("Estimate parameters")
	estimationModel := stochastic.NewEstimationModelJSON(eventRegistryJSON)

	// write simulation file
	outputFileName := ctx.String(utils.OutputFlag.Name)
	if outputFileName == "" {
		outputFileName = "./simulation.json"
	}
	log.Noticef("Write simulation file %v", outputFileName)
	if err := estimationModel.WriteJSON(outputFileName); err != nil {
		return err
	}

	return nil
}
