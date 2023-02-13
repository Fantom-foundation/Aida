package stochastic

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"

	"github.com/Fantom-foundation/Aida/stochastic"
	"github.com/Fantom-foundation/Aida/utils"
	"github.com/urfave/cli/v2"
)

// StochasticReplayCommand data structure for the replay app.
var StochasticReplayCommand = cli.Command{
	Action:    stochasticReplayAction,
	Name:      "replay",
	Usage:     "Simulates StateDB operations using a random generator with realistic distributions",
	ArgsUsage: "<simulation-file>",
	Flags: []cli.Flag{
		&utils.VerboseFlag,
	},
	Description: `
The stochastic replay command requires two argument:
<simulation.json> <simulation-length>

<simulation.json> contains the simulation parameters produced by the stochastic estimator.
<simulation-length> determines the number of issued StateDB operations`,
}

// stochasticReplayAction implements the replay command. The user
// provides simulation file and simulation as arguments.
func stochasticReplayAction(ctx *cli.Context) error {
	if ctx.Args().Len() != 2 {
		return fmt.Errorf("missing simulation file and simulation length as parameter")
	}
	simLength, perr := strconv.ParseInt(ctx.Args().Get(1), 10, 64)
	if perr != nil {
		return fmt.Errorf("error: simulation length is not an integer")
	}

	// open file
	file, err := os.Open(ctx.Args().Get(0))
	if err != nil {
		return fmt.Errorf("failed opening simulation file")
	}
	defer file.Close()

	// read simulation file in JSON format.
	contents, err := ioutil.ReadAll(file)
	if err != nil {
		return fmt.Errorf("failed reading simulation file")
	}
	var simulation stochastic.EstimationModelJSON
	err = json.Unmarshal(contents, &simulation)
	if err != nil {
		return fmt.Errorf("failed loading simulation file")
	}

	// run simulation.
	fmt.Printf("stochastic replay: run simulation ...\n")
	verbose := ctx.Bool(utils.VerboseFlag.Name)
	stochastic.RunStochasticReplay(&simulation, int(simLength), verbose)

	return nil
}
