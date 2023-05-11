package state

import (
	"github.com/Fantom-foundation/Aida/utils"
	"github.com/Fantom-foundation/Aida/world-state/db/snapshot"
	"github.com/urfave/cli/v2"
)

// CmdInfo retrieves basic info about snapshot database
var CmdInfo = cli.Command{
	Action:      info,
	Name:        "info",
	Aliases:     []string{"i"},
	Usage:       "Retrieves basic info about snapshot database.",
	Description: `Looks up current block number of database.`,
	ArgsUsage:   "",
	Flags:       []cli.Flag{},
}

// root retrieves root hash of given block number
func info(ctx *cli.Context) error {
	// make config
	cfg, err := utils.NewConfig(ctx, utils.NoArgs)
	if err != nil {
		return err
	}

	// try to open state DB
	stateDB, err := snapshot.OpenStateDB(cfg.WorldStateDb)
	if err != nil {
		return err
	}
	defer snapshot.MustCloseStateDB(stateDB)

	// make logger
	log := utils.NewLogger(cfg.LogLevel, "info")

	blk, err := stateDB.GetBlockNumber()
	if err != nil {
		return err
	}

	log.Infof("database is currently at block %d", blk)
	log.Info("done")
	return nil
}
