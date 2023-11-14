package db

import (
	"github.com/Fantom-foundation/Aida/logger"
	util_db "github.com/Fantom-foundation/Aida/utildb"
	"github.com/Fantom-foundation/Aida/utils"
	"github.com/urfave/cli/v2"
)

// UpdateCommand downloads aida-db and new patches
var UpdateCommand = cli.Command{
	Action: update,
	Name:   "update",
	Usage:  "download aida-db patches",
	Flags: []cli.Flag{
		&utils.AidaDbFlag,
		&utils.ChainIDFlag,
		&logger.LogLevelFlag,
		&utils.CompactDbFlag,
		&utils.DbTmpFlag,
		&utils.ValidateFlag,
		&utils.UpdateTypeFlag,
	},
	Description: ` 
Updates aida-db by downloading patches from aida-db generation server.
`,
}

// update updates aida-db by downloading patches from aida-db generation server.
func update(ctx *cli.Context) error {
	cfg, err := utils.NewConfig(ctx, utils.NoArgs)
	if err != nil {
		return err
	}
	if err = util_db.Update(cfg); err != nil {
		return err
	}

	return util_db.PrintMetadata(cfg.AidaDb)
}
