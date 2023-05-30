package stochastic

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

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

	if ctx.Args().Len() != 1 {
		return fmt.Errorf("missing event file")
	}

	log.Info("Visualize statistical events")

	// open and parse event file
	inputFileName := ctx.Args().Get(0)

	log.Info("Read event file %v", inputFileName)

	file, err := os.Open(inputFileName)
	if err != nil {
		return fmt.Errorf("failed opening event file %v; %v", inputFileName, err)
	}

	defer file.Close()

	contents, err := ioutil.ReadAll(file)
	if err != nil {
		return fmt.Errorf("failed reading event file %v; %v", inputFileName, err)
	}

	var eventRegistry stochastic.EventRegistryJSON
	err = json.Unmarshal(contents, &eventRegistry)
	if err != nil {
		return fmt.Errorf("cannot unmarshal event registry; %v", err)
	}

	// fire-up web-server
	port := ctx.String(utils.PortFlag.Name)
	if port == "" {
		port = "8080"
	}

	log.Noticef("Open web browser with http://localhost:" + port)
	log.Notice("Cancel visualize with ^C")

	visualizer.FireUpWeb(&eventRegistry, port)

	return nil
}
