package updateset

import (
	"errors"
	"fmt"
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
	lastUpdateSetBlk, err := db.GetLastKey()
	if err != nil {
		return fmt.Errorf("cannot get last update-set; %v", err)
	}

	// set first block
	if lastUpdateSetBlk > 0 {
		cfg.First = lastUpdateSetBlk + 1
	}
	err = db.Close()
	if err != nil {
		return err
	}

	// initialize updateDB
	udb, err := substate.OpenUpdateDB(cfg.UpdateDb)
	if err != nil {
		return err
	}
	defer udb.Close()

	// iterate through subsets in sequence
	substate.SetSubstateDb(cfg.SubstateDb)
	substate.OpenSubstateDBReadOnly()
	defer substate.CloseSubstateDB()

	ddb, err := substate.OpenDestroyedAccountDBReadOnly(cfg.DeletionDb)
	if err != nil {
		return err
	}
	defer ddb.Close()

	return GenUpdateSet(cfg, udb, ddb, cfg.First, cfg.Last, interval)
}

// GenUpdateSet generates a series of update sets from substate db
func GenUpdateSet(cfg *utils.Config, udb *substate.UpdateDB, ddb *substate.DestroyedAccountDB, first, last uint64, interval uint64) error {
	var (
		err               error
		destroyedAccounts []common.Address
		log               = logger.NewLogger(cfg.LogLevel, "Generate Update Set")
	)

	log.Infof("Update buffer size: %v bytes", cfg.UpdateBufferSize)

	// start with putting metadata into the udb
	if err = udb.PutMetadata(interval, cfg.UpdateBufferSize); err != nil {
		return err
	}

	update := make(substate.SubstateAlloc)

	iter := substate.NewSubstateIterator(first, cfg.Workers)
	defer iter.Release()

	var (
		txCount       uint64                 // transaction counter
		curBlock      uint64                 // current block
		checkPoint    uint64                 // block number of the next interval
		isFirst       = true                 // first block
		estimatedSize uint64                 // estimated size of current update set
		maxSize       = cfg.UpdateBufferSize // recommended size 700 MB
	)

	log.Noticef("Generate update sets from block %v to block %v", first, last)
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
				udb.PutUpdateSet(curBlock, &update, destroyedAccounts)
				log.Infof("\tTx: %v, Accounts: %v, Suicided: %v, Size: %v",
					txCount, len(update), len(destroyedAccounts), estimatedSize)
				if cfg.ValidateTxState {
					if !udb.GetUpdateSet(curBlock).Equal(update) {
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
			if tx.Block > last {
				break
			}
			curBlock = tx.Block
		}

		// clear storage of destroyed and resurrected accounts in
		// the current transaction before merging its substate
		destroyed, resurrected, err := ddb.GetDestroyedAccounts(curBlock, tx.Transaction)
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
