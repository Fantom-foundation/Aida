package compare

import (
	"context"
	"fmt"
	"github.com/Fantom-foundation/Aida-Testing/world-state/db/snapshot"
	"github.com/Fantom-foundation/Aida-Testing/world-state/logger"
	"github.com/urfave/cli/v2"
)

const (
	flagStateDBPath = "db"
	flagRefDBPath   = "to"
)

// CmdCompareState compares states of two databases whether they are identical
var CmdCompareState = cli.Command{
	Action:      compareState,
	Name:        "compare",
	Aliases:     []string{"cmp"},
	Usage:       "Compare whether states of two databases are identical",
	Description: `Compares given snapshot database against target snapshot database.`,
	ArgsUsage:   "<to>",
	Flags: []cli.Flag{
		&cli.PathFlag{
			Name:     flagRefDBPath,
			Usage:    "Path to target snapshot database",
			Required: true,
		},
	},
}

// compareState compares state of two databases
func compareState(ctx *cli.Context) error {
	// try to open state DB
	stateDB, err := snapshot.OpenStateDB(ctx.Path(flagStateDBPath))
	if err != nil {
		return err
	}
	defer snapshot.MustCloseStateDB(stateDB)

	// try to open target state DB
	stateRefDB, err := snapshot.OpenStateDB(ctx.Path(flagRefDBPath))
	if err != nil {
		return err
	}
	defer snapshot.MustCloseStateDB(stateRefDB)

	// make logger
	log := logger.New(ctx.App.Writer, "info")

	log.Infof("Comparing %s against %s", ctx.Path(flagStateDBPath), ctx.Path(flagRefDBPath))

	// call CompareTo against target database
	err = stateDB.CompareTo(context.Background(), stateRefDB)
	if err != nil {
		err = fmt.Errorf("while comparing %s against %s ; %s", ctx.Path(flagStateDBPath), ctx.Path(flagRefDBPath), err.Error())
		return err
	}
	log.Info("Databases are identical")
	log.Info("done")
	return nil
}
