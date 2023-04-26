package updateset

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/Fantom-foundation/Aida/utils"
	substate "github.com/Fantom-foundation/Substate"
	"github.com/ethereum/go-ethereum/common"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/urfave/cli/v2"
)

// GenUpdateSet command generates a series of update sets from substate db.
func GenUpdateSet(ctx *cli.Context) error {
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
	worldStateDir := ctx.String(utils.WorldStateDirFlag.Name)

	// initialize updateDB
	db := substate.OpenUpdateDB(cfg.UpdateDBDir)
	defer db.Close()
	update := make(substate.SubstateAlloc)

	// iterate through subsets in sequence
	substate.SetSubstateDirectory(cfg.SubstateDBDir)
	substate.OpenSubstateDBReadOnly()
	defer substate.CloseSubstateDB()

	// store world state
	cfg.First = utils.FirstSubstateBlock
	log.Notice("Load initial World-State and store its substateAlloc")

	ws, err := utils.GenerateWorldState(worldStateDir, cfg.First-1, cfg)
	if err != nil {
		return err
	}
	log.Noticef("write block %v to UpdateDB", cfg.First-1)
	db.PutUpdateSet(cfg.First-1, &ws, destroyedAccounts)
	log.Noticef("Accounts: %v", len(ws))

	iter := substate.NewSubstateIterator(cfg.First, cfg.Workers)
	defer iter.Release()
	deletedAccountDB := substate.OpenDestroyedAccountDBReadOnly(cfg.DeletedAccountDir)
	defer deletedAccountDB.Close()

	txCount := uint64(0)
	curBlock := uint64(0)
	isFirst := true
	var checkPoint uint64

	for iter.Next() {
		tx := iter.Value()
		if isFirst {
			checkPoint = (((tx.Block/interval)+1)*interval - 1)
			isFirst = false
		}
		// new block
		if curBlock != tx.Block {
			// write an update-set until prev block to update-set db
			if tx.Block > checkPoint {
				log.Infof("Write block %v to UpdateDB", curBlock)
				db.PutUpdateSet(curBlock, &update, destroyedAccounts)
				log.Infof("Tx: %v, Accounts: %v, Suicided: %v", txCount, len(update), len(destroyedAccounts))
				checkPoint += interval
				destroyedAccounts = nil
				if cfg.ValidateTxState {
					if !db.GetUpdateSet(curBlock).Equal(update) {
						return fmt.Errorf("validation failed\n")
					}
				}
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

		update.Merge(tx.Substate.OutputAlloc)
		txCount++
	}
	return err
}
