package stochastic

import (
	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/stochastic"
	"github.com/Fantom-foundation/Aida/utils"
	"github.com/urfave/cli/v2"
)

// StochasticGenerateCommand data structure for the record app.
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

// stochasticGenerateAction produces an event file with uniform parameters.
func stochasticGenerateAction(ctx *cli.Context) error {
	cfg, err := utils.NewConfig(ctx, utils.NoArgs)
	if err != nil {
		return err
	}
	log := logger.NewLogger(cfg.LogLevel, "StochasticGenerate")

	// create a new uniformly distributed event registry
	log.Info("Produce uniform stochastic events")
	eventRegistry := stochastic.GenerateUniformRegistry(cfg, log)

	// writing event registry in JSON format
	if cfg.Output == "" {
		cfg.Output = "./events.json"
	}
	log.Noticef("Write events file %v", cfg.Output)
	err = eventRegistry.WriteJSON(cfg.Output)
	if err != nil {
		return err
	}

	return nil
}
