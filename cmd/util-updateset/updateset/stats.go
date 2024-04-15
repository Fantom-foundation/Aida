// Copyright 2024 Fantom Foundation
// This file is part of Aida Testing Infrastructure for Sonic
//
// Aida is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Aida is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with Aida. If not, see <http://www.gnu.org/licenses/>.

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
	db, err := substate.OpenUpdateDBReadOnly(cfg.UpdateDb)
	if err != nil {
		return err
	}
	defer db.Close()

	iter := substate.NewUpdateSetIterator(db, cfg.First, cfg.Last)
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
