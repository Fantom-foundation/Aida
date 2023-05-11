// Package state implements executable entry points to the world state generator app.
package state

import (
	"context"
	"time"

	"github.com/Fantom-foundation/Aida/utils"
	"github.com/Fantom-foundation/Aida/world-state/db/snapshot"
	"github.com/Fantom-foundation/Aida/world-state/types"
	"github.com/urfave/cli/v2"
)

// CmdClone defines a CLI command for cloning world state dump database.
var CmdClone = cli.Command{
	Action:  cloneDB,
	Name:    "clone",
	Aliases: []string{"c"},
	Usage:   `Creates a clone of the world state dump database.`,
	Flags: []cli.Flag{
		&utils.TargetDbFlag,
	},
}

// cloneDB performs the DB cloning.
func cloneDB(ctx *cli.Context) error {
	// make config
	cfg, err := utils.NewConfig(ctx, utils.NoArgs)
	if err != nil {
		return err
	}

	// try to open source DB
	inputDB, err := snapshot.OpenStateDB(cfg.WorldStateDb)
	if err != nil {
		return err
	}
	defer snapshot.MustCloseStateDB(inputDB)

	// try to open source DB
	outputDB, err := snapshot.OpenStateDB(DefaultPath(ctx, &utils.TargetDbFlag, ".aida/clone"))
	if err != nil {
		return err
	}
	defer snapshot.MustCloseStateDB(outputDB)

	// make logger
	log := utils.NewLogger(cfg.LogLevel, "clone")
	logTick := time.NewTicker(2 * time.Second)
	defer logTick.Stop()

	var count int
	err = inputDB.Copy(context.Background(), outputDB, func(_ *types.Account) {
		count++
		select {
		case <-logTick.C:
			log.Infof("transferred %d accounts", count)
		default:
		}
	})

	log.Infof("%d accounts done", count)
	return err
}
