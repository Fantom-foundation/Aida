package runvm

import (
	"fmt"
	"math/big"
	"os"
	"time"

	"github.com/Fantom-foundation/Aida/tracer/operation"
	"github.com/Fantom-foundation/Aida/utils"
	substate "github.com/Fantom-foundation/Substate"
	"github.com/urfave/cli/v2"
)

const (
	progressReportBlockInterval uint64 = 100_000
	logFrequency                       = 15 * time.Second
)

// RunVM implements trace command for executing VM on a chosen storage system.
func RunVM(ctx *cli.Context) error {
	var (
		elapsed, wsEndTime      time.Duration
		hours, minutes, seconds uint32
		err                     error
		start                   time.Time
		beggining               time.Time
		txCount                 int
		lastTxCount             int
		gasCount                = new(big.Int)
		lastGasCount            = new(big.Int)
		// Progress reporting (block based)
		lastBlockProgressReportBlock    uint64
		lastBlockProgressReportTime     time.Time
		lastBlockProgressReportTxCount  int
		lastBlockProgressReportGasCount = new(big.Int)
	)
	beggining = time.Now()

	// process general arguments
	cfg, argErr := utils.NewConfig(ctx, utils.BlockRangeArgs)
	cfg.StateValidationMode = utils.SubsetCheck
	if argErr != nil {
		return argErr
	}

	log := utils.NewLogger(ctx, "DB Merger")

	// start CPU profiling if requested.
	if err := utils.StartCPUProfile(cfg); err != nil {
		return err
	}
	defer utils.StopCPUProfile(cfg)

	// iterate through subsets in sequence
	substate.SetSubstateDirectory(cfg.SubstateDBDir)
	substate.OpenSubstateDBReadOnly()
	defer substate.CloseSubstateDB()

	db, stateDirectory, loadedExistingDB, err := utils.PrepareStateDB(cfg)
	if err != nil {
		return err
	}
	if !cfg.KeepStateDB {
		log.Warningf("StateDB at %v will be removed at the end of this run.\n", stateDirectory)
		defer os.RemoveAll(stateDirectory)
	}

	ws := substate.SubstateAlloc{}
	if cfg.SkipPriming || loadedExistingDB {
		log.Warning("Skipping DB priming.\n")
	} else {
		// load the world state
		log.Noticef("\nLoad and advance world state to block %v\n", cfg.First-1)
		start = time.Now()
		ws, err = utils.GenerateWorldStateFromUpdateDB(cfg, cfg.First-1)
		if err != nil {
			return err
		}

		elapsed = time.Since(start).Round(1 * time.Second)
		hours, minutes, seconds = parseTime(elapsed)
		log.Infof("\tElapsed time: %vh %vm %vs, accounts: %v\n", hours, minutes, seconds, len(ws))

		// prime stateDB
		log.Notice("Prime StateDB \n")
		start = time.Now()
		utils.PrimeStateDB(ws, db, cfg)

		elapsed = time.Since(start).Round(1 * time.Second)
		hours, minutes, seconds = parseTime(elapsed)
		log.Infof("\tPriming elapsed time: %vh %vm %vs\n", hours, minutes, seconds)

		// delete destroyed accounts from stateDB
		log.Notice("Delete destroyed accounts \n")
		start = time.Now()
		// remove destroyed accounts until one block before the first block
		err = utils.DeleteDestroyedAccountsFromStateDB(db, cfg, cfg.First-1)

		elapsed = time.Since(start).Round(1 * time.Second)
		hours, minutes, seconds = parseTime(elapsed)
		log.Infof("\tDel-dest-acc elapsed time: %vh %vm %vs\n", hours, minutes, seconds)
		if err != nil {
			return err
		}
	}

	// print memory usage after priming
	if cfg.MemoryBreakdown {
		if usage := db.GetMemoryUsage(); usage != nil {
			log.Noticef("State DB memory usage: %d byte\n%s\n", usage.UsedBytes, usage.Breakdown)
		} else {
			log.Info("Utilized storage solution does not support memory breakdowns.\n")
		}
	}

	// wrap stateDB for profiling
	var stats *operation.ProfileStats
	if cfg.Profile {
		db, stats = NewProxyProfiler(db)
	}

	if cfg.ValidateWorldState {
		if len(ws) == 0 {
			ws, err = utils.GenerateWorldStateFromUpdateDB(cfg, cfg.First-1)
			if err != nil {
				return err
			}
		}
		if err := utils.DeleteDestroyedAccountsFromWorldState(ws, cfg, cfg.First-1); err != nil {
			return fmt.Errorf("failed to remove deleted accoount from the world state. %v", err)
		}
		if err := utils.ValidateStateDB(ws, db, false); err != nil {
			return fmt.Errorf("pre: World state is not contained in the stateDB. %v", err)
		}
	}

	// Release world state to free memory.
	ws = substate.SubstateAlloc{}

	if cfg.EnableProgress {
		start = time.Now()
		wsEndTime = time.Since(start)
	}

	log.Notice("Run VM\n")
	var curBlock uint64 = 0
	var curSyncPeriod uint64
	isFirstBlock := true
	iter := substate.NewSubstateIterator(cfg.First, cfg.Workers)

	defer iter.Release()

	for iter.Next() {
		tx := iter.Value()
		// initiate first sync-period and block.
		if isFirstBlock {
			if tx.Block > cfg.Last {
				break
			}
			curSyncPeriod = tx.Block / cfg.SyncPeriodLength
			curBlock = tx.Block
			db.BeginSyncPeriod(curSyncPeriod)
			db.BeginBlock(curBlock)
			lastBlockProgressReportBlock = tx.Block
			lastBlockProgressReportBlock -= lastBlockProgressReportBlock % progressReportBlockInterval
			lastBlockProgressReportTime = time.Now()
			isFirstBlock = false
			// close off old block and possibly sync-periods
		} else if curBlock != tx.Block {
			if tx.Block > cfg.Last {
				break
			}

			// Mark the end of the old block.
			db.EndBlock()

			// Move on sync-periods if needed.
			newSyncPeriod := tx.Block / cfg.SyncPeriodLength
			for curSyncPeriod < newSyncPeriod {
				db.EndSyncPeriod()
				curSyncPeriod++
				db.BeginSyncPeriod(curSyncPeriod)
			}
			// Mark the beginning of a new block
			curBlock = tx.Block
			db.BeginBlock(curBlock)
		}
		if cfg.MaxNumTransactions >= 0 && txCount >= cfg.MaxNumTransactions {
			break
		}
		// run VM
		db.PrepareSubstate(&tx.Substate.InputAlloc, tx.Substate.Env.Number)
		db.BeginTransaction(uint32(tx.Transaction))
		err = utils.ProcessTx(db, cfg, tx.Block, tx.Transaction, tx.Substate)
		if err != nil {
			log.Critical("\tFAILED\n")
			err = fmt.Errorf("VM execution failed; %v", err)
			break
		}
		db.EndTransaction()
		txCount++
		gasCount = new(big.Int).Add(gasCount, new(big.Int).SetUint64(tx.Substate.Result.GasUsed))

		ticker := time.NewTicker(logFrequency)

		if cfg.EnableProgress {
			// report progress
			elapsed = time.Since(start)

			log.Warning(elapsed.Seconds())

			// Report progress on a regular time interval (wall time).
			select {
			case <-ticker.C:
				d := new(big.Int).Sub(gasCount, lastGasCount)
				g := new(big.Float).Quo(new(big.Float).SetInt(d), new(big.Float).SetFloat64(float64(elapsed-wsEndTime)))

				txRate := float64(txCount-lastTxCount) / float64(elapsed-wsEndTime)
				elapsed = time.Since(start).Round(1 * time.Second)
				hours, minutes, seconds = parseTime(elapsed)
				log.Infof("Elapsed time: %vh %vm %vs, at block %v (~ %v Tx/s, ~ %v Gas/s)\n", hours, minutes, seconds, tx.Block, txRate, g)
				wsEndTime = elapsed
				lastTxCount = txCount
				lastGasCount.Set(gasCount)
			default:
				break
			}

			// Report progress on a regular block interval (simulation time).
			for tx.Block >= lastBlockProgressReportBlock+progressReportBlockInterval {
				numTransactions := txCount - lastBlockProgressReportTxCount
				lastBlockProgressReportTxCount = txCount

				gasUsed := new(big.Int).Sub(gasCount, lastBlockProgressReportGasCount)
				lastBlockProgressReportGasCount.Set(gasCount)

				now := time.Now()
				intervalTime := now.Sub(lastBlockProgressReportTime)
				lastBlockProgressReportTime = now

				txRate := float64(numTransactions) / intervalTime.Seconds()
				gasRate, _ := new(big.Float).SetInt(gasUsed).Float64()
				gasRate = gasRate / intervalTime.Seconds()

				log.Infof("run-vm: Reached block %d, last interval rate ~ %.1f Tx/s, ~ %.1f Gas/s\n", tx.Block, txRate, gasRate)
				lastBlockProgressReportBlock += progressReportBlockInterval
			}
		}
	}

	if !isFirstBlock && err == nil {
		db.EndBlock()
		db.EndSyncPeriod()
	}

	runTime := time.Since(start).Seconds()

	if cfg.ContinueOnFailure {
		log.Warningf("run-vm: %v errors found\n", utils.NumErrors)
	}

	if cfg.ValidateWorldState && err == nil {
		log.Notice("Validate final state\n")
		if ws, err = utils.GenerateWorldStateFromUpdateDB(cfg, cfg.Last); err != nil {
			return err
		}
		if err := utils.DeleteDestroyedAccountsFromWorldState(ws, cfg, cfg.Last); err != nil {
			return fmt.Errorf("Failed to remove deleted accoount from the world state. %v", err)
		}
		if err := utils.ValidateStateDB(ws, db, false); err != nil {
			return fmt.Errorf("World state is not contained in the stateDB. %v", err)
		}
	}

	if cfg.MemoryBreakdown {
		if usage := db.GetMemoryUsage(); usage != nil {
			log.Notice("State DB memory usage: %d byte\n%s\n", usage.UsedBytes, usage.Breakdown)
		} else {
			log.Info("Utilized storage solution does not support memory breakdowns.\n")
		}
	}

	// write memory profile if requested
	if err := utils.StartMemoryProfile(cfg); err != nil {
		return err
	}

	if cfg.Profile {
		fmt.Println("=================Statistics=================")
		stats.PrintProfiling(log)
		fmt.Println("============================================")
	}

	if cfg.KeepStateDB && !isFirstBlock {
		rootHash, _ := db.Commit(true)
		if err := utils.WriteStateDbInfo(stateDirectory, cfg, curBlock, rootHash); err != nil {
			log.Error(err)
		}
		//rename directory after closing db.
		defer utils.RenameTempStateDBDirectory(cfg, stateDirectory, curBlock)
	} else if cfg.KeepStateDB && isFirstBlock {
		// no blocks were processed.
		log.Warning("No blocks were processed. StateDB is not saved.\n")
		defer os.RemoveAll(stateDirectory)
	}

	// close the DB and print disk usage
	log.Info("Close StateDB database")
	start = time.Now()
	if err := db.Close(); err != nil {
		log.Errorf("Failed to close database: %v", err)
	}

	// print progress summary
	if cfg.EnableProgress {
		g := new(big.Float).Quo(new(big.Float).SetInt(gasCount), new(big.Float).SetFloat64(runTime))

		hours, minutes, seconds = parseTime(time.Since(beggining).Round(1 * time.Second))

		log.Infof("Total elapsed time: %vh %vm %vs, processed %v blocks, %v transactions (~ %.1f Tx/s) (~ %.1f Gas/s)\n", hours, minutes, seconds, cfg.Last-cfg.First+1, txCount, float64(txCount)/(runTime), g)
		log.Infof("Closing DB took %v\n", time.Since(start))
		log.Infof("Final disk usage: %v MiB\n", float32(utils.GetDirectorySize(stateDirectory))/float32(1024*1024))
	}

	return err
}

func parseTime(elapsed time.Duration) (uint32, uint32, uint32) {
	var (
		hours, minutes, seconds uint32
	)

	seconds = uint32(elapsed.Seconds())

	if seconds > 60 {
		minutes = seconds / 60
	}

	if minutes > 60 {
		hours = minutes / 60
	}

	return hours, minutes, seconds
}
