package trace

import (
	"fmt"
	"strconv"

	"github.com/ethereum/go-ethereum/substate"
	"github.com/urfave/cli/v2"
)

// genUpdateSetCommand data structure for the record app
var GenUpdateSetCommand = cli.Command{
	Action:    genUpdateSet,
	Name:      "gen-update-set",
	Usage:     "generate update set database",
	ArgsUsage: "<blockNumFirst> <blockNumLast> <blockRange>",
	Flags: []cli.Flag{
		&substate.WorkersFlag,
		&substate.SubstateDirFlag,
		&updateDirectoryFlag,
		&validateEndState,
	},
	Description: `
The trace gen-update-set command requires two arguments:
<blockNumFirst> <blockNumLast> <blockRange>

<blockNumFirst> and <blockNumLast> are the first and
last block of the inclusive range of blocks to trace transactions.

<blockRange> is the interval of writing update set to updateDB.`,
}

// genUpdateSet implements trace command for executing VM on a chosen storage system.
func genUpdateSet(ctx *cli.Context) error {
	var err error
	// process arguments and flags
	if ctx.Args().Len() != 3 {
		return fmt.Errorf("trace command requires exactly 2 arguments")
	}
	first, last, argErr := SetBlockRange(ctx.Args().Get(0), ctx.Args().Get(1))
	if argErr != nil {
		return argErr
	}
	interval, ferr := strconv.ParseUint(ctx.Args().Get(2), 10, 64)
	if ferr != nil {
		return ferr
	}
	workers := ctx.Int(substate.WorkersFlag.Name)
	validate := ctx.Bool(validateEndState.Name)
	updateDir := ctx.String(updateDirectoryFlag.Name)

	// initialize updateDB
	db := substate.OpenUpdateDB(updateDir)
	defer db.Close()
	update := make(substate.SubstateAlloc)

	// iterate through subsets in sequence
	substate.SetSubstateFlags(ctx)
	substate.OpenSubstateDBReadOnly()
	defer substate.CloseSubstateDB()

	iter := substate.NewSubstateIterator(first, workers)
	checkPoint := ((first / interval) + 1) * interval
	defer iter.Release()

	txCount := uint64(0)

	for iter.Next() {
		tx := iter.Value()
		// stop when reaching end of block range
		if tx.Block > last {
			break
		}
		update.Merge(tx.Substate.OutputAlloc)
		txCount++

		// write to update set db
		if tx.Block >= checkPoint {
			fmt.Printf("write block %v to updateDB\n", tx.Block)
			fmt.Printf("\tTx: %v, Accounts: %v\n", txCount, len(update))
			db.PutUpdateSet(tx.Block, &update)
			checkPoint += interval
			//validate
			if validate {
				if !db.GetUpdateSet(tx.Block).Equal(update) {
					return fmt.Errorf("validation failed\n")
				}
			}
			update = make(substate.SubstateAlloc)
			txCount = 0
		}
	}
	return err
}
