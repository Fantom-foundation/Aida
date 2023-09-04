package blockprofile

import (
	"fmt"
	"os"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/profile/blockprofile"
	"github.com/Fantom-foundation/Aida/utils"

	substate "github.com/Fantom-foundation/Substate"

	"github.com/urfave/cli/v2"
)

var BlockProfileCommand = cli.Command{
	Action:    blockProfileAction,
	Name:      "blocks",
	Usage:     "produces blockprofile statistics for transactions.",
	ArgsUsage: "<blockNumFirst> <blockNumLast>",
	Flags: []cli.Flag{
		// AidaDb
		&utils.AidaDbFlag,

		// StateDb
		&utils.CarmenSchemaFlag,
		&utils.StateDbImplementationFlag,
		&utils.StateDbVariantFlag,
		&utils.DbTmpFlag,

		// VM
		&utils.VmImplementation,

		// Priming
		&utils.UpdateBufferSizeFlag,

		// Utils
		&substate.WorkersFlag,
		&utils.ChainIDFlag,
		&utils.SyncPeriodLengthFlag,
		&logger.LogLevelFlag,
	},
}

// blockProfileAction produces block processing statistics for transactions.
func blockProfileAction(ctx *cli.Context) error {
	// process arguments
	cfg, argErr := utils.NewConfig(ctx, utils.BlockRangeArgsProfileDB)
	if argErr != nil {
		return argErr
	}
	log := logger.NewLogger(cfg.LogLevel, "Profile block processing")

	// open Aida database
	log.Notice("Open Aida database.")
	substate.SetSubstateDb(cfg.SubstateDb)
	substate.OpenSubstateDBReadOnly()
	defer substate.CloseSubstateDB()

	// construct StateDB object
	db, stateDbDir, err := utils.PrepareStateDB(cfg)
	if err != nil {
		return err
	}
	defer os.RemoveAll(stateDbDir)

	// prime the database for first block
	log.Notice("Prime database.")
	if err := utils.LoadWorldStateAndPrime(db, cfg, cfg.First-1); err != nil {
		return fmt.Errorf("priming failed. %v", err)
	}

	// init sqlite DB that stores blockprofile information
	log.Notice("Open profile database.")
	profileDB, err := blockprofile.NewProfileDB(cfg.ProfileDB)
	if err != nil {
		return fmt.Errorf("unable to open out SQlite DB; %v", err)
	}

	_, err = profileDB.DeleteByBlockRange(cfg.First, cfg.Last)
	if err != nil {
		return fmt.Errorf("unable to delete rows within block range: %d-%d; %v", cfg.First, cfg.Last, err)
	}

	log.Notice("Profile blocks and their transactions.")
	curBlock := uint64(0)
	curSyncPeriod := uint64(0)
	isFirstBlock := true
	var blockTimer time.Time
	var context *blockprofile.Context

	// simple progress report
	var blockPeriod uint64 = 100_000
	var lastBlockReport uint64 = cfg.First - cfg.First%blockPeriod

	// process all transaction in sequential order from first to last block
	iter := substate.NewSubstateIterator(cfg.First, cfg.Workers)
	defer iter.Release()

	for iter.Next() {
		tx := iter.Value()

		// initiate first sync-period and block.
		if isFirstBlock {
			isFirstBlock = false

			if tx.Block > cfg.Last {
				break
			}

			// initiate a sync period
			curSyncPeriod = tx.Block / cfg.SyncPeriodLength

			curBlock = tx.Block
			context = blockprofile.NewContext()

			db.BeginSyncPeriod(curSyncPeriod)

			// initiate a new block
			blockTimer = time.Now()
			db.BeginBlock(curBlock)
		} else if curBlock != tx.Block {

			if tx.Block > cfg.Last {
				break
			}

			// close last block
			db.EndBlock()

			// report progress
			// TODO: reuse progress report of aida-vm-sdb
			if tx.Block >= lastBlockReport+blockPeriod {
				log.Infof("At block %v.", tx.Block)
				lastBlockReport += blockPeriod
			}
			// obtain profile data for block
			data, err := context.GetProfileData(curBlock, time.Since(blockTimer))
			if err != nil {
				return err
			}

			// write profile data into profile database
			profileDB.Add(*data)

			// create a new blockprofile profiling context
			curBlock = tx.Block
			context = blockprofile.NewContext()

			// switch to new sync period if enough blocks processed
			newSyncPeriod := tx.Block / cfg.SyncPeriodLength
			for curSyncPeriod < newSyncPeriod {
				db.EndSyncPeriod()
				curSyncPeriod++
				db.BeginSyncPeriod(curSyncPeriod)
			}
			// open new block
			blockTimer = time.Now()
			db.BeginBlock(curBlock)
		}

		// process current transaction
		txTimer := time.Now()
		if _, err = utils.ProcessTx(db, cfg, tx.Block, tx.Transaction, tx.Substate); err != nil {
			log.Critical("\tFAILED executing transaction.")
			return fmt.Errorf("execution failed; %v", err)
		}

		// record transaction for parallel experiment
		if err = context.RecordTransaction(tx, time.Since(txTimer)); err != nil {
			log.Critical("\tFAILED profiling transaction.")
			return fmt.Errorf("transaction profiling error; %v", err)
		}
	}
	if !isFirstBlock {
		db.EndBlock()

		// obtain profile data for block
		data, err := context.GetProfileData(curBlock, time.Since(blockTimer))
		if err != nil {
			return err
		}

		// write profile data into profile database
		profileDB.Add(*data)

		db.EndSyncPeriod()
	}

	// close databases
	log.Info("Close Aida database.")
	if err = db.Close(); err != nil {
		log.Errorf("Failed to close StateDB: %v", err)
	}
	log.Info("Close Profile database.")
	if err = profileDB.Close(); err != nil {
		log.Errorf("Failed to profiling database: %v", err)
	}

	log.Notice("Finished blockprofile profiling.")

	return err
}
