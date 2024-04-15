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
	"fmt"

	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/stochastic"
	"github.com/Fantom-foundation/Aida/stochastic/visualizer"
	"github.com/Fantom-foundation/Aida/utils"
	"github.com/urfave/cli/v2"
)

// StochasticVisualizeCommand data structure for the visualize app
var StochasticVisualizeCommand = cli.Command{
	Action:    stochasticVisualizeAction,
	Name:      "visualize",
	Usage:     "produces a graphical view of the estimated parameters for various distributions",
	ArgsUsage: "<event-file>",
	Flags: []cli.Flag{
		&utils.PortFlag,
	},
	Description: `
The stochastic visualize command requires one argument:
<events.json>

<events.json> is the event file produced by the stochastic recorder.`,
}

// stochasticVisualizeAction implements the visualize command for computing statistical parameters.
func stochasticVisualizeAction(ctx *cli.Context) error {
	log := logger.NewLogger("INFO", "StochasticVisualize")

	// parse parameters
	if ctx.Args().Len() != 1 {
		return fmt.Errorf("missing event file")
	}
	inputFileName := ctx.Args().Get(0)

	// read events file
	log.Infof("Read event file %v", inputFileName)
	eventRegistry, err := stochastic.ReadEvents(inputFileName)
	if err != nil {
		return err
	}

	// fire-up web-server and visualize events
	port := ctx.String(utils.PortFlag.Name)
	if port == "" {
		port = "8080"
	}
	log.Noticef("Open web browser with http://localhost:" + port)
	log.Notice("Cancel visualize with ^C")
	visualizer.FireUpWeb(eventRegistry, port)

	return nil
}
