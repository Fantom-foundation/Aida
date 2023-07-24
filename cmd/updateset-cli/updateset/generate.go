package updateset

import (
	"errors"
	"fmt"
	"os"
	"strconv"

	"github.com/Fantom-foundation/Aida/logger"
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
		&substate.SubstateDbFlag,
		&substate.WorkersFlag,
		&utils.UpdateDbFlag,
		&utils.UpdateBufferSizeFlag,
		&utils.ValidateFlag,
		&utils.WorldStateFlag,
		&logger.LogLevelFlag,
	},
	Description: `
The gen-update-set command requires two arguments: <blockNumLast> <interval>

<blockNumLast> is last block of the inclusive range of blocks to generate update set.

<interval> is the block interval of writing update set to updateDB.`,
}

// generateUpdateSet command generates a series of update sets from substate db.
func generateUpdateSet(ctx *cli.Context) error {
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

	// we need all three db paths to execute this cmd
	if cfg.UpdateDb == "" {
		return fmt.Errorf("you need to specify where you want update-db to save (--update-db)")
	}

	if cfg.DeletionDb == "" {
		return fmt.Errorf("you need to specify path to existing deletion-db (--deletion-db)")
	}

	if cfg.SubstateDb == "" {
		return fmt.Errorf("you need to specify path to existing substate (--substate-db)")
	}

	// retrieve last update set
	db, err := substate.OpenUpdateDB(cfg.UpdateDb)
	if err != nil {
		return err
	}
	// set first block
	if db.GetLastKey() > 0 {
		cfg.First = db.GetLastKey() + 1
	}
	db.Close()

	return GenUpdateSet(cfg, cfg.First, interval)
}

// GenUpdateSet generates a series of update sets from substate db
func GenUpdateSet(cfg *utils.Config, first uint64, interval uint64) error {
	var (
		err               error
		destroyedAccounts []common.Address
		log               = logger.NewLogger(cfg.LogLevel, "Generate Update Set")
	)

	// initialize updateDB
	db, err := substate.OpenUpdateDB(cfg.UpdateDb)
	if err != nil {
		return err
	}
	defer db.Close()

	// start with putting metadata into the db
	if err = db.PutMetadata(interval, cfg.UpdateBufferSize); err != nil {
		return err
	}

	// iterate through subsets in sequence
	substate.SetSubstateDb(cfg.SubstateDb)
	substate.OpenSubstateDBReadOnly()
	defer substate.CloseSubstateDB()

	skipOperaWorldState := true
	// legacy support if a user wants to generate update set from the first opera world state
	// store world state if a path is provided
	worldState := cfg.WorldStateDb
	if _, err := os.Stat(worldState); err == nil {
		skipOperaWorldState = false
	}

	update := make(substate.SubstateAlloc)
	if !skipOperaWorldState {
		first = utils.FirstOperaBlock
		log.Notice("Load initial worldstate and store its substateAlloc")
		ws, err := utils.GenerateFirstOperaWorldState(worldState, cfg)
		if err != nil {
			return err
		}
		size := update.EstimateIncrementalSize(ws)
		log.Infof("Write block %v to updateDB", first-1)
		db.PutUpdateSet(first-1, &ws, destroyedAccounts)
		log.Infof("\tAccounts: %v, Size: %v", len(ws), size)
	}

	iter := substate.NewSubstateIterator(first, cfg.Workers)
	defer iter.Release()
	deletedAccountDB, err := substate.OpenDestroyedAccountDBReadOnly(cfg.DeletionDb)
	if err != nil {
		return err
	}
	defer deletedAccountDB.Close()

	var (
		txCount       uint64                 // transaction counter
		curBlock      uint64                 // current block
		checkPoint    uint64                 // block number of the next interval
		isFirst       = true                 // first block
		estimatedSize uint64                 // estimated size of current update set
		maxSize       = cfg.UpdateBufferSize // recommended size 700 MB
	)

	log.Noticef("Generate update sets from block %v to block %v", first, cfg.Last)
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
				if tx.Block > checkPoint {
					checkPoint += interval
				}
				estimatedSize = 0
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

		// estimate update-set size after merge
		estimatedSize += update.EstimateIncrementalSize(tx.Substate.OutputAlloc)
		// perform substate merge
		update.Merge(tx.Substate.OutputAlloc)
		txCount++
	}

	return err
}
