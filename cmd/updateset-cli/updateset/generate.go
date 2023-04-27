package updateset

import (
	"errors"
	"fmt"
	"os"
	"strconv"

	"github.com/Fantom-foundation/Aida/utils"
	substate "github.com/Fantom-foundation/Substate"
	"github.com/ethereum/go-ethereum/common"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/urfave/cli/v2"
)

var GenUpdateSetCommand = cli.Command{
	Action:    generateUpdateSet,
	Name:      "generate",
	Usage:     "generate update-set from substate",
	ArgsUsage: "<blockNumLast> <interval>",
	Flags: []cli.Flag{
		&utils.ChainIDFlag,
		&utils.DeletionDbFlag,
		&substate.SubstateDirFlag,
		&utils.UpdateDbFlag,
		&utils.ValidateFlag,
		&utils.WorldStateFlag,
		&utils.LogLevel,
	},
	Description: `
The gen-update-set command requires two arguments: <blockNumLast> <interval>

<blockNumLast> is last block of the inclusive range of blocks to generate update set.

<interval> is the block interval of writing update set to updateDB.`,
}

// generateUpdateSet command generates a series of update sets from substate db.
func generateUpdateSet(ctx *cli.Context) error {
	var (
		err               error
		destroyedAccounts []common.Address
		log               = utils.NewLogger(ctx.String(utils.LogLevel.Name), "Generate Update Set")
	)
	// process arguments and flags
	if ctx.Args().Len() != 2 {
		return fmt.Errorf("gen-update-set command requires exactly 2 arguments")
	}
	cfg, argErr := utils.NewConfig(ctx, utils.LastBlockArg)
	if argErr != nil {
		return argErr
	}
	interval, ferr := strconv.ParseUint(ctx.Args().Get(1), 10, 64)
	if ferr != nil {
		return ferr
	}

	// initialize updateDB
	db := substate.OpenUpdateDB(cfg.UpdateDb)
	defer db.Close()
	update := make(substate.SubstateAlloc)

	// iterate through subsets in sequence
	substate.SetSubstateDirectory(cfg.SubstateDb)
	substate.OpenSubstateDBReadOnly()
	defer substate.CloseSubstateDB()

	// set first block
	cfg.First = db.GetLastKey() + 1

	// store world state if applicable
	skipWorldState := cfg.First > utils.FirstSubstateBlock
	worldState := ctx.String(utils.WorldStateFlag.Name)
	if _, err := os.Stat(worldState); os.IsNotExist(err) {
		skipWorldState = true
	}

	if !skipWorldState {
		cfg.First = utils.FirstSubstateBlock
		log.Notice("Load initial worldstate and store its substateAlloc")
		ws, err := utils.GenerateWorldState(worldState, cfg.First-1, cfg)
		if err != nil {
			return err
		}
		log.Infof("Write block %v to updateDB", cfg.First-1)
		db.PutUpdateSet(cfg.First-1, &ws, destroyedAccounts)
		log.Infof("\tAccounts: %v", len(ws))
	}

	iter := substate.NewSubstateIterator(cfg.First, cfg.Workers)
	defer iter.Release()
	deletedAccountDB := substate.OpenDestroyedAccountDBReadOnly(cfg.DeletionDb)
	defer deletedAccountDB.Close()

	var (
		txCount       uint64              // transaction counter
		curBlock      uint64              // current block
		checkPoint    uint64              // block number of the next interval
		isFirst       = true              // first block
		estimatedSize uint64              // estimated size of current update set
		maxSize       uint64 = 1000000000 // 1GB
	)

	log.Noticef("Generate update sets from block %v to block %v.\n", cfg.First, cfg.Last)
	for iter.Next() {
		tx := iter.Value()
		// if first block, calculate next change point
		if isFirst {
			checkPoint = ((tx.Block/interval)+1)*interval - 1
			isFirst = false
		}
		// new block
		if curBlock != tx.Block {
			// write an update-set to updatedb if 1) interval condition is met or 2) estimated size > max size
			if tx.Block > checkPoint || estimatedSize > maxSize {
				log.Infof("Write block %v to updateDB", curBlock)
				db.PutUpdateSet(curBlock, &update, destroyedAccounts)
				log.Infof("\tTx: %v, Accounts: %v, Suicided: %v, Size: %v",
					txCount, len(update), len(destroyedAccounts), estimatedSize)
				if cfg.ValidateTxState {
					if !db.GetUpdateSet(curBlock).Equal(update) {
						return fmt.Errorf("validation failed\n")
					}
				}

				// reset update set & counters
				if estimatedSize > maxSize {
					estimatedSize = 0
				} else {
					checkPoint += interval
				}
				destroyedAccounts = nil
				update = make(substate.SubstateAlloc)
				txCount = 0
			}

			// stop when reaching end of block range
			if tx.Block > cfg.Last {
				break
			}
			curBlock = tx.Block
		}

		// clear storage of destroyed and resurrected accounts in
		// the current transaction before merging its substate
		destroyed, resurrected, err := deletedAccountDB.GetDestroyedAccounts(curBlock, tx.Transaction)
		if !(err == nil || errors.Is(err, leveldb.ErrNotFound)) {
			return err
		}
		utils.ClearAccountStorage(update, destroyed)
		utils.ClearAccountStorage(update, resurrected)
		destroyedAccounts = append(destroyedAccounts, destroyed...)
		destroyedAccounts = append(destroyedAccounts, resurrected...)

		// perform substate merge
		update.Merge(tx.Substate.OutputAlloc)
		// estimate update-set size after merge
		estimatedSize += update.EstimateIncrementalSize(tx.Substate.OutputAlloc)
		txCount++
	}
	return err
}
