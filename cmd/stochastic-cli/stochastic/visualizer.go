package stochastic

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/Fantom-foundation/Aida/stochastic"
	"github.com/Fantom-foundation/Aida/stochastic/visualizer"
	"github.com/Fantom-foundation/Aida/utils"
	"github.com/urfave/cli/v2"
)

// StochasticVisualizerCommand data structure for the visualizer app
var StochasticVisualizerCommand = cli.Command{
	Action:    stochasticVisualizerAction,
	Name:      "visualize",
	Usage:     "produces a graphical view of the estimated parameters for various distributions",
	ArgsUsage: "<event-file>",
	Flags: []cli.Flag{
		&utils.PortFlag,
	},
	Description: `
The stochastic visualizer command requires one argument:
<events.json>

<events.json> is the event file produced by the stochastic recorder.`,
}

// stochasticVisualizerAction implements visualizer command for computing statistical parameters.
func stochasticVisualizerAction(ctx *cli.Context) error {
	if ctx.Args().Len() != 1 {
		return fmt.Errorf("missing event file")
	}

	// open file
	file, err := os.Open(ctx.Args().Get(0))
	if err != nil {
		return fmt.Errorf("failed opening event file")
	}
	defer file.Close()

	// read file
	contents, err := ioutil.ReadAll(file)
	if err != nil {
		return fmt.Errorf("failed reading event file")
	}

	var eventRegistry stochastic.EventRegistryJSON
	json.Unmarshal(contents, &eventRegistry)

	// visualize visualizer results
	addr := ctx.String(utils.PortFlag.Name)
	if addr == "" {
		addr = "8080"
	}

	// fire-up web-server
	fmt.Println("Open web browser with http://localhost:" + addr)
	fmt.Println("Cancel visualizer with ^C")
	visualizer.FireUpWeb(&eventRegistry, addr)

	return nil
}
