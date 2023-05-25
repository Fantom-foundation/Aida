package stochastic

import (
	"github.com/Fantom-foundation/Aida/logger"
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
		&logger.LogLevelFlag,
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

	cfg, err := utils.NewConfig(ctx, utils.NoArgs)
	if err != nil {
		return err
	}

	log := logger.NewLogger(cfg.LogLevel, "StochasticGenerate")

	log.Info("Produce uniform stochastic event file")

	// create a new uniformly distributed event registry
	eventRegistry := stochastic.GenerateUniformRegistry(cfg, log)

	// writing event registry
	if cfg.Output == "" {
		cfg.Output = "./events.json"
	}
	log.Noticef("Write event file to %v", cfg.Output)
	err = WriteEvents(eventRegistry, cfg.Output)
	if err != nil {
		return err
	}

	return nil
}
