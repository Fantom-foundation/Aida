package trace

import (
	"fmt"
	"log"
	"strconv"

	"github.com/ethereum/go-ethereum/substate"
	"github.com/urfave/cli/v2"
)

// genUpdateSetCommand data structure for the record app
var GenUpdateSetCommand = cli.Command{
	Action:    genUpdateSet,
	Name:      "gen-update-set",
	Usage:     "generate update set database",
	ArgsUsage: "<blockNumLast> <blockRange>",
	Flags: []cli.Flag{
		&chainIDFlag,
		&substate.WorkersFlag,
		&substate.SubstateDirFlag,
		&updateDBDirFlag,
		&validateFlag,
		&worldStateDirFlag,
	},
	Description: `
The trace gen-update-set command requires two arguments:
<blockNumLast> <blockRange>

<blockNumLast> is last block of the inclusive range of blocks to generate update set.

<blockRange> is the interval of writing update set to updateDB.`,
}

// genUpdateSet implements trace command for executing VM on a chosen storage system.
func genUpdateSet(ctx *cli.Context) error {
	var err error
	// process arguments and flags
	if ctx.Args().Len() != 2 {
		return fmt.Errorf("trace command requires exactly 2 arguments")
	}
	last, argErr := strconv.ParseUint(ctx.Args().Get(0), 10, 64)
	if argErr != nil {
		return argErr
	}
	interval, ferr := strconv.ParseUint(ctx.Args().Get(1), 10, 64)
	if ferr != nil {
		return ferr
	}
	workers := ctx.Int(substate.WorkersFlag.Name)
	validate := ctx.Bool(validateFlag.Name)
	updateDir := ctx.String(updateDBDirFlag.Name)
	worldStateDir := ctx.String(worldStateDirFlag.Name)
	setFirstBlockFromChainID(ctx.Int(chainIDFlag.Name))
	log.Printf("first block %v\n", FirstSubstateBlock)

	// initialize updateDB
	db := substate.OpenUpdateDB(updateDir)
	defer db.Close()
	update := make(substate.SubstateAlloc)

	// iterate through subsets in sequence
	substate.SetSubstateFlags(ctx)
	substate.OpenSubstateDBReadOnly()
	defer substate.CloseSubstateDB()

	// store world state
	first := FirstSubstateBlock
	log.Printf("Load initial worldstate and store its substateAlloc\n")
	ws, err := generateWorldState(worldStateDir, first-1, workers)
	if err != nil {
		return err
	}
	log.Printf("write block %v to updateDB\n", first-1)
	db.PutUpdateSet(first-1, &ws)
	log.Printf("\tAccounts: %v\n", len(ws))

	iter := substate.NewSubstateIterator(first, workers)
	defer iter.Release()

	txCount := uint64(0)
	oldBlock := uint64(0)
	isFirst := true
	var checkPoint uint64

	for iter.Next() {
		tx := iter.Value()
		if isFirst {
			checkPoint = (((tx.Block/interval)+1)*interval - 1)
			isFirst = false
		}
		// new block
		if oldBlock != tx.Block {
			// write an update-set until prev block to update-set db
			if tx.Block > checkPoint {
				log.Printf("write block %v to updateDB\n", oldBlock)
				db.PutUpdateSet(oldBlock, &update)
				log.Printf("\tTx: %v, Accounts: %v\n", txCount, len(update))
				checkPoint += interval

				if validate {
					if !db.GetUpdateSet(oldBlock).Equal(update) {
						return fmt.Errorf("validation failed\n")
					}
				}
				update = make(substate.SubstateAlloc)
				txCount = 0
			}

			// stop when reaching end of block range
			if tx.Block > last {
				break
			}
			oldBlock = tx.Block
		}

		update.Merge(tx.Substate.OutputAlloc)
		txCount++
	}
	return err
}
