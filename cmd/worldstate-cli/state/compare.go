// Package state implements executable entry points to the world state generator app.
package state

import (
	"context"
	"fmt"

	"github.com/Fantom-foundation/Aida/cmd/worldstate-cli/flags"
	"github.com/Fantom-foundation/Aida/utils"
	"github.com/Fantom-foundation/Aida/world-state/db/snapshot"
	"github.com/urfave/cli/v2"
)

// CmdCompareState compares states of two databases whether they are identical
var CmdCompareState = cli.Command{
	Action:      compareDb,
	Name:        "compare",
	Aliases:     []string{"cmp"},
	Usage:       "Compare whether states of two databases are identical.",
	Description: `Compares given snapshot database against target snapshot database.`,
	ArgsUsage:   "<to>",
	Flags: []cli.Flag{
		&flags.TargetDBPath,
	},
}

// compareDb compares world state stored inside source and destination databases.
func compareDb(ctx *cli.Context) error {
	// try to open state DB
	stateDB, err := snapshot.OpenStateDB(ctx.Path(flags.StateDBPath.Name))
	if err != nil {
		return err
	}
	defer snapshot.MustCloseStateDB(stateDB)

	// try to open target state DB
	stateRefDB, err := snapshot.OpenStateDB(DefaultPath(ctx, &flags.TargetDBPath, "clone"))
	if err != nil {
		return err
	}
	defer snapshot.MustCloseStateDB(stateRefDB)

	// make logger
	log := utils.NewLogger(ctx, "cmp")
	log.Infof("comparing %s against %s", ctx.Path(flags.StateDBPath.Name), ctx.Path(flags.TargetDBPath.Name))

	// call CompareTo against target database
	err = stateDB.CompareTo(context.Background(), stateRefDB)
	if err != nil {
		return fmt.Errorf("compare failed; %s", err.Error())
	}

	log.Info("databases are identical")
	return nil
}
