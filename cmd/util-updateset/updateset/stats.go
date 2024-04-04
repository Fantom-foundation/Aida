package updateset

import (
	"fmt"

	"github.com/Fantom-foundation/Aida/utils"
	"github.com/Fantom-foundation/Substate/db"
	"github.com/urfave/cli/v2"
)

var UpdateSetStatsCommand = cli.Command{
	Action:    reportUpdateSetStats,
	Name:      "stats",
	Usage:     "print number of accounts and storage keys in update-set",
	ArgsUsage: "<blockNumFirst> <blockNumLast>",
	Flags: []cli.Flag{
		&utils.UpdateDbFlag,
	},
	Description: `
The stats command requires one arguments: <blockNumLast> -- the last block of update-set.`,
}

// reportUpdateSetStats reports number of accounts and storage keys in an update-set
func reportUpdateSetStats(ctx *cli.Context) error {
	var (
		err error
	)
	cfg, argErr := utils.NewConfig(ctx, utils.BlockRangeArgs)
	if argErr != nil {
		return argErr
	}
	// initialize updateDB
	udb, err := db.NewDefaultUpdateDB(cfg.UpdateDb)
	if err != nil {
		return err
	}
	defer udb.Close()

	iter := udb.NewUpdateSetIterator(cfg.First, cfg.Last)
	defer iter.Release()

	for iter.Next() {
		update := iter.Value()
		state := update.WorldState
		fmt.Printf("%v,%v,", update.Block, len(state))
		storage := 0
		for account := range state {
			storage = storage + len(state[account].Storage)
		}
		fmt.Printf("%v\n", storage)
	}
	return err
}
