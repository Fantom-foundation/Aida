package stochastic

import (
	"log"

	"github.com/Fantom-foundation/Aida/stochastic"
	"github.com/Fantom-foundation/Aida/utils"
	"github.com/urfave/cli/v2"
)

// StochasticGenerateCommand data structure for the record app
var StochasticGenerateCommand = cli.Command{
	Action:    stochasticGenerateAction,
	Name:      "generate",
	Usage:     "generate uniform events file",
	ArgsUsage: "",
	Flags: []cli.Flag{
		&utils.BlockLengthFlag,
		&utils.SyncPeriodLengthFlag,
		&utils.TransactionLengthFlag,
		&utils.ContractNumberFlag,
		&utils.KeysNumberFlag,
		&utils.ValuesNumberFlag,
		&utils.SnapshotDepthFlag,
	},
	Description: "The stochastic produces an events.json file with uniform parameters",
}

// stochasticGenerateAction generates the uniform simulation data and writes the JSON file.
func stochasticGenerateAction(ctx *cli.Context) error {
	var err error

	cfg, err := utils.NewConfig(ctx, utils.NoArgs)
	if err != nil {
		return err
	}

	log.Println("produce uniform stochastic event file")

	// create a new uniformly distributed event registry
	eventRegistry := stochastic.GenerateUniformRegistry(cfg)

	// writing event registry
	if cfg.Output == "" {
		cfg.Output = "./events.json"
	}
	log.Printf("write event file to %v\n", cfg.Output)
	WriteEvents(eventRegistry, cfg.Output)

	return nil
}
