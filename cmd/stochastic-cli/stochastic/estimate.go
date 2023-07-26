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
