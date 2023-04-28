package updateset

import (
	"fmt"

	"github.com/Fantom-foundation/Aida/utils"
	substate "github.com/Fantom-foundation/Substate"
	"github.com/urfave/cli/v2"
)

var UpdateSetStatsCommand = cli.Command{
	Action:    reportUpdateSetStats,
	Name:      "stats",
	Usage:     "print number of accounts and storage keys in update-set",
	ArgsUsage: "<blockNumLast>",
	Flags: []cli.Flag{
		&utils.UpdateDbFlag,
		&utils.AidaDbFlag,
	},
	Description: `
The stats command requires one arguments: <blockNumLast> -- the last block of update-set.`,
}

// reportUpdateSetStats reports number of accounts and storage keys in an update-set
func reportUpdateSetStats(ctx *cli.Context) error {
	var (
		err error
	)
	// process arguments and flags
	if ctx.Args().Len() != 1 {
		return fmt.Errorf("stats command requires exactly 1 arguments")
	}
	cfg, argErr := utils.NewConfig(ctx, utils.LastBlockArg)
	if argErr != nil {
		return argErr
	}
	// initialize updateDB
	db := substate.OpenUpdateDBReadOnly(cfg.UpdateDb)
	defer db.Close()

	iter := substate.NewUpdateSetIterator(db, 0, cfg.Last)
	defer iter.Release()

	for iter.Next() {
		update := iter.Value()
		state := *update.UpdateSet
		fmt.Printf("%v,%v,", update.Block, len(state))
		storage := 0
		for account := range state {
			storage = storage + len(state[account].Storage)
		}
		fmt.Printf("%v\n", storage)
	}
	return err
}
