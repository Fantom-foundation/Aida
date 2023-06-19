package stochastic

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/Fantom-foundation/Aida/logger"
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
	log := logger.NewLogger("INFO", "StochasticEstimate")

	if ctx.Args().Len() != 1 {
		return fmt.Errorf("missing event file")
	}

	log.Info("Produce a simulation file from an event file")

	// open and parse event file
	inputFileName := ctx.Args().Get(0)

	log.Infof("Read event file %v", inputFileName)

	file, err := os.Open(inputFileName)
	if err != nil {
		return fmt.Errorf("failed opening event file %v; %v", inputFileName, err)
	}
	defer file.Close()

	contents, err := ioutil.ReadAll(file)
	if err != nil {
		return fmt.Errorf("failed reading event file; %v", err)
	}

	var eventRegistry stochastic.EventRegistryJSON

	err = json.Unmarshal(contents, &eventRegistry)
	if err != nil {
		return fmt.Errorf("cannot unmarshal event registry; %v", err)
	}

	// estimate parameters
	log.Info("Estimate parameters")
	estimationModel := stochastic.NewEstimationModelJSON(&eventRegistry)

	// write event file
	outputFileName := ctx.String(utils.OutputFlag.Name)
	if outputFileName == "" {
		outputFileName = "./simulation.json"
	}

	log.Noticef("Write event filename %v", outputFileName)

	err = WriteSimulation(&estimationModel, outputFileName)
	if err != nil {
		return err
	}

	return nil
}

// WriteSimulation writes event file in JSON format.
func WriteSimulation(m *stochastic.EstimationModelJSON, filename string) error {
	f, fErr := os.Create(filename)
	if fErr != nil {
		return fmt.Errorf("cannot open JSON file; %v", fErr)
	}
	defer f.Close()

	jOut, jErr := json.MarshalIndent(m, "", "    ")
	if jErr != nil {
		return fmt.Errorf("failed to convert JSON file; %v", jErr)
	}

	_, pErr := fmt.Fprintln(f, string(jOut))
	if pErr != nil {
		return fmt.Errorf("failed to convert JSON file; %v", pErr)
	}

	return nil
}
