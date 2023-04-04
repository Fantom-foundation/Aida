package stochastic

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/Fantom-foundation/Aida/stochastic"
	"github.com/Fantom-foundation/Aida/utils"
	"github.com/urfave/cli/v2"
)

// StochasticEstimateCommand data structure for the estimator app
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

	// writing event registry
	fmt.Printf("stochastic estimate: write simulation file ...\n")
	estimationModel := stochastic.NewEstimationModelJSON(&eventRegistry)
	outputFileName := ctx.String(utils.OutputFlag.Name)
	if outputFileName == "" {
		outputFileName = "./simulation.json"
	}
	WriteSimulation(&estimationModel, outputFileName)

	return nil
}

// WriteSimulation writes event file in JSON format.
func WriteSimulation(m *stochastic.EstimationModelJSON, filename string) {
	f, fErr := os.Create(filename)
	if fErr != nil {
		log.Fatalf("cannot open JSON file. Error: %v", fErr)
	}
	defer f.Close()

	jOut, jErr := json.MarshalIndent(m, "", "    ")
	if jErr != nil {
		log.Fatalf("failed to convert JSON file. Error: %v", jErr)
	}

	_, pErr := fmt.Fprintln(f, string(jOut))
	if pErr != nil {
		log.Fatalf("failed to convert JSON file. Error: %v", pErr)
	}
}
