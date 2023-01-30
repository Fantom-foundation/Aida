package stochastic

import (
	"fmt"
	"encoding/json"
	"os"
	"io/ioutil"

	"github.com/Fantom-foundation/Aida/stochastic"
	"github.com/urfave/cli/v2"
)

// StochasticEstimatorCommand data structure for the estimator app
var StochasticEstimatorCommand = cli.Command{
	Action:    stochasticEstimatorAction,
	Name:      "estimator",
	Usage:     "captures and estimators StateDB operations while processing blocks",
	ArgsUsage: "<blockNumFirst> <blockNumLast>",
	Flags: []cli.Flag{
	},
	Description: `
The stochastic estimator command requires one argument:
<EventDistribution.json>

<EventDistribution.json> is the event distribution produced by 
the recorder.`,
}


// stochasticEstimatorAction implements estimator command for deriving 
// statistical parameter
func stochasticEstimatorAction(ctx *cli.Context) error {
	if ctx.Args().Len() != 1 {
		return fmt.Errorf("missing distribution file")
	}

	// open file
	file, err := os.Open(ctx.Args().Get(0))	
	if err != nil {
		return fmt.Errorf("failed opening distribution file")
	}
	defer file.Close()

	// read file
	contents, err := ioutil.ReadAll(file)
	if err != nil {
		return fmt.Errorf("failed reading distribution file")
	}

	var distribution stochastic.EventDistribution 
	json.Unmarshal(contents, &distribution)

	// prepare web-data for rendering 
	stochastic.PrepareWebData(distribution)

	// render web-page
	fmt.Println("Open web browser with http://localhost:8080")
	fmt.Println("Cancel estimator with ^C")
	stochastic.FireUpWeb()

	return nil
}
