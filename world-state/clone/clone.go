// Package clone implements a cloning tool for the world state dump database.
package clone

import (
	"github.com/Fantom-foundation/Aida-Testing/world-state/db/snapshot"
	"github.com/Fantom-foundation/Aida-Testing/world-state/dump"
	"github.com/Fantom-foundation/Aida-Testing/world-state/logger"
	"github.com/Fantom-foundation/Aida-Testing/world-state/types"
	"github.com/urfave/cli/v2"
	"time"
)

const (
	flagTarget = "to"
)

// CmdClone defines a CLI command for cloning world state dump database.
var CmdClone = cli.Command{
	Action:  cloneDB,
	Name:    "clone",
	Aliases: []string{"c"},
	Usage:   `Creates a clone of the world state dump database.`,
	Flags: []cli.Flag{
		&cli.PathFlag{
			Name:     flagTarget,
			Usage:    "target folder for the cloned DB",
			Value:    "",
			Required: true,
		},
	},
}

// cloneDB performs the DB cloning.
func cloneDB(ctx *cli.Context) error {
	// try to open source DB
	inputDB, err := snapshot.OpenStateDB(ctx.Path(dump.FlagOutputDBPath))
	if err != nil {
		return err
	}
	defer snapshot.MustCloseStateDB(inputDB)

	// try to open source DB
	outputDB, err := snapshot.OpenStateDB(ctx.Path(flagTarget))
	if err != nil {
		return err
	}
	defer snapshot.MustCloseStateDB(outputDB)

	// make logger
	log := logger.New(ctx.App.Writer, "info")
	logTick := time.NewTicker(2 * time.Second)
	defer logTick.Stop()

	var count int
	err = inputDB.Copy(outputDB, func(_ *types.Account) {
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
