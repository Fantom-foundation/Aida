// Package clone implements a cloning tool for the world state dump database.
package clone

import (
	"github.com/Fantom-foundation/Aida-Testing/world-state/db/snapshot"
	"github.com/Fantom-foundation/Aida-Testing/world-state/dump"
	"github.com/Fantom-foundation/Aida-Testing/world-state/types"
	"github.com/urfave/cli/v2"
	"log"
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

	var count int
	err = inputDB.Copy(outputDB, func(_ *types.Account) {
		count++
	})
	log.Printf("processed %d accounts", count)
	return err
}
