package db

import (
	"fmt"

	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/utildb"
	"github.com/Fantom-foundation/Aida/utils"
	"github.com/urfave/cli/v2"
)

// GenerateCommand data structure for the replay app
var GenerateCommand = cli.Command{
	Action: generate,
	Name:   "generate",
	Usage:  "generates full aida-db from substatedb - generates deletiondb and updatesetdb, merges them into aida-db and then creates a patch",
	Flags: []cli.Flag{
		&utils.AidaDbFlag,
		&utils.ChainIDFlag,
		&utils.OperaDbFlag,
		&utils.OperaBinaryFlag,
		&utils.OutputFlag,
		&utils.DbTmpFlag,
		&utils.UpdateBufferSizeFlag,
		&utils.SkipStateHashScrappingFlag,
		&utils.WorkersFlag,
		&logger.LogLevelFlag,
	},
	Description: `
The db generate command requires 4 arguments:
<firstBlock> <lastBlock> <firstEpoch> <lastEpoch>
This command is designed for manual generation of deletion, updateset and patch just from substates in aidadb.
`,
}

// generate AidaDb
func generate(ctx *cli.Context) error {
	cfg, err := utils.NewConfig(ctx, utils.NoArgs)
	if err != nil {
		return fmt.Errorf("cannot create config %v", err)
	}

	g, err := utildb.NewGenerator(ctx, cfg)
	if err != nil {
		return err
	}

	err = g.PrepareManualGenerate(ctx, cfg)
	if err != nil {
		return fmt.Errorf("prepareManualGenerate: %v; make sure you have correct substate range already recorded in aidadb", err)
	}

	if err = g.Generate(); err != nil {
		return err
	}

	utildb.MustCloseDB(g.AidaDb)

	return utildb.PrintMetadata(g.Cfg.AidaDb)
}
