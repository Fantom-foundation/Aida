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
