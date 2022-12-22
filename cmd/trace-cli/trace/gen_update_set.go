package trace

import (
	"errors"
	"fmt"
	"log"
	"strconv"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/substate"
	"github.com/syndtr/goleveldb/leveldb"
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
		&deletedAccountDirFlag,
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
	var (
		err               error
		destroyedAccounts []common.Address
	)
	// process arguments and flags
	if ctx.Args().Len() != 2 {
		return fmt.Errorf("trace command requires exactly 2 arguments")
	}
	cfg, argErr := NewTraceConfig(ctx, lastBlockArg)
	if argErr != nil {
		return argErr
	}
	interval, ferr := strconv.ParseUint(ctx.Args().Get(1), 10, 64)
	if ferr != nil {
		return ferr
	}
	worldStateDir := ctx.String(worldStateDirFlag.Name)

	// initialize updateDB
	db := substate.OpenUpdateDB(cfg.updateDBDir)
	defer db.Close()
	update := make(substate.SubstateAlloc)

	// iterate through subsets in sequence
	substate.SetSubstateFlags(ctx)
	substate.OpenSubstateDBReadOnly()
	defer substate.CloseSubstateDB()

	// store world state
	cfg.first = FirstSubstateBlock
	log.Printf("Load initial worldstate and store its substateAlloc\n")
	ws, err := generateWorldState(worldStateDir, cfg.first-1, cfg)
	if err != nil {
		return err
	}
	log.Printf("write block %v to updateDB\n", cfg.first-1)
	db.PutUpdateSet(cfg.first-1, &ws, destroyedAccounts)
	log.Printf("\tAccounts: %v\n", len(ws))

	iter := substate.NewSubstateIterator(cfg.first, cfg.workers)
	defer iter.Release()
	deletedAccountDB := substate.OpenDestroyedAccountDBReadOnly(cfg.deletedAccountDir)
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
				log.Printf("write block %v to updateDB\n", curBlock)
				db.PutUpdateSet(curBlock, &update, destroyedAccounts)
				log.Printf("\tTx: %v, Accounts: %v, Suicided: %v\n", txCount, len(update), len(destroyedAccounts))
				checkPoint += interval
				destroyedAccounts = nil
				if cfg.validateTxState {
					if !db.GetUpdateSet(curBlock).Equal(update) {
						return fmt.Errorf("validation failed\n")
					}
				}
				update = make(substate.SubstateAlloc)
				txCount = 0
			}

			// stop when reaching end of block range
			if tx.Block > cfg.last {
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
		clearAccountStorage(update, destroyed)
		clearAccountStorage(update, resurrected)
		destroyedAccounts = append(destroyedAccounts, destroyed...)
		destroyedAccounts = append(destroyedAccounts, resurrected...)

		update.Merge(tx.Substate.OutputAlloc)
		txCount++
	}
	return err
}

// clearAccountStorage clears storage
func clearAccountStorage(update substate.SubstateAlloc, accounts []common.Address) {
	for _, addr := range accounts {
		if _, found := update[addr]; found {
			update[addr].Storage = make(map[common.Hash]common.Hash)
		}
	}
}
