package stochastic

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"

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
	if ctx.Args().Len() != 1 {
		return fmt.Errorf("missing event file")
	}

	log.Println("visualize statistical events")

	// open and parse event file
	inputFileName := ctx.Args().Get(0)
	log.Printf("read event file %v\n", inputFileName)
	file, err := os.Open(inputFileName)
	if err != nil {
		return fmt.Errorf("failed opening event file %v", inputFileName)
	}
	defer file.Close()
	contents, err := ioutil.ReadAll(file)
	if err != nil {
		return fmt.Errorf("failed reading event file %v", inputFileName)
	}
	var eventRegistry stochastic.EventRegistryJSON
	json.Unmarshal(contents, &eventRegistry)

	// fire-up web-server
	addr := ctx.String(utils.PortFlag.Name)
	if addr == "" {
		addr = "8080"
	}
	log.Println("Open web browser with http://localhost:" + addr)
	log.Println("Cancel visualize with ^C")
	visualizer.FireUpWeb(&eventRegistry, addr)

	return nil
}
