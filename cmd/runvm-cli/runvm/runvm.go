package runvm

import (
	"fmt"
	"math/big"
	"os"
	"time"

	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/tracer/profile"
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
		elapsed, lastLog        time.Duration
		hours, minutes, seconds uint32
		err                     error
		start, beginning        time.Time
		txCount                 int
		lastTxCount             int
		totalGas                = new(big.Int)
		currentGas              = new(big.Int)
		lastGasCount            = new(big.Int)
		d                       = new(big.Int)
		g                       = new(big.Float)
		calcTime                = new(big.Float)
		currentGasCountFloat    = new(big.Float)
		// Progress reporting (block based)
		lastBlockProgressReportBlock    uint64
		lastBlockProgressReportTime     time.Time
		lastBlockProgressReportTxCount  int
		lastBlockProgressReportGasCount = new(big.Int)
		lastProfileBlock                uint64
		stateDbDir                      string
	)
	beginning = time.Now()

	// process general arguments
	cfg, argErr := utils.NewConfig(ctx, utils.BlockRangeArgs)
	// if source db is supplied, make a copy and modify the copy
	if argErr != nil {
		return argErr
	}

	cfg.StateValidationMode = utils.SubsetCheck
	cfg.CopySrcDb = true

	log := logger.NewLogger(cfg.LogLevel, "Run-VM")

	// start CPU profiling if requested.
	if err := utils.StartCPUProfile(cfg); err != nil {
		return err
	}
	defer utils.StopCPUProfile(cfg)

	// iterate through subsets in sequence
	substate.SetSubstateDb(cfg.SubstateDb)
	substate.OpenSubstateDBReadOnly()
	defer substate.CloseSubstateDB()

	db, stateDbDir, err := utils.PrepareStateDB(cfg)
	if err != nil {
		return err
	}
	if !cfg.KeepDb {
		log.Warningf("--keep-db is not used. Directory %v with DB will be removed at the end of this run.", stateDbDir)
		defer os.RemoveAll(stateDbDir)
	}

	ws := substate.SubstateAlloc{}
	if cfg.SkipPriming || cfg.StateDbSrc != "" {
		log.Warning("Skipping DB priming.\n")
	} else {
		log.Notice("Prime stateDB")
		start = time.Now()
		if err := utils.LoadWorldStateAndPrime(db, cfg, cfg.First-1); err != nil {
			return fmt.Errorf("priming failed. %v", err)
		}
		elapsed = time.Since(start)
		hours, minutes, seconds = logger.ParseTime(elapsed)
		log.Infof("\tPriming elapsed time: %vh %vm %vs\n", hours, minutes, seconds)
		if err != nil {
			return err
		}
	}

	// print memory usage after priming
	utils.MemoryBreakdown(db, cfg, log)

	// wrap stateDB for profiling
	var stats *profile.Stats
	if cfg.Profile {
		db, stats = NewProxyProfiler(db, cfg.ProfileFile)
	}

	if cfg.ValidateWorldState {
		if len(ws) == 0 {
			ws, err = utils.GenerateWorldStateFromUpdateDB(cfg, cfg.First-1)
			if err != nil {
				return err
			}
		}
		if err := utils.ValidateStateDB(ws, db, false); err != nil {
			return fmt.Errorf("pre: World state is not contained in the stateDB. %v", err)
		}
	}

	// Release world state to free memory.
	ws = substate.SubstateAlloc{}

	if !cfg.Quiet {
		start = time.Now()
	}

	log.Notice("Run VM")
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
			lastProfileBlock = tx.Block - (tx.Block % cfg.ProfileInterval)
			isFirstBlock = false
			// close off old block and possibly sync-periods
		} else if curBlock != tx.Block {
			if tx.Block > cfg.Last {
				break
			}

			if cfg.DbImpl != "erigon" {
				db.EndBlock()
			}

			// Move on sync-periods if needed.
			newSyncPeriod := tx.Block / cfg.SyncPeriodLength
			for curSyncPeriod < newSyncPeriod {
				if cfg.DbImpl != "erigon" {
					db.EndSyncPeriod()
				}
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
		if tx.Transaction >= utils.PseudoTx {
			log.Debugf("Block %v Tx %v: pseudo tx", tx.Block, tx.Transaction)
			utils.ProcessPseudoTx(tx.Substate.OutputAlloc, db)
		} else {
			log.Debugf("Block %v Tx %v: external tx", tx.Block, tx.Transaction)
			err = utils.ProcessTx(db, cfg, tx.Block, tx.Transaction, tx.Substate)
			if err != nil {
				log.Critical("\tFAILED")
				err = fmt.Errorf("VM execution failed; %v", err)
				break
			}
			txCount++
			totalGas.Add(totalGas, currentGas.SetUint64(tx.Substate.Result.GasUsed))
		}

		if !cfg.Quiet {
			// report progress
			elapsed = time.Since(start)

			// Report progress on a regular time interval (wall time).
			if elapsed-lastLog >= logFrequency {
				d.Sub(totalGas, lastGasCount)
				currentGasCountFloat.SetUint64(d.Uint64())
				calcTime.SetFloat64(elapsed.Seconds() - lastLog.Seconds())

				g.Quo(currentGasCountFloat, calcTime)

				f, _ := g.Float64()

				txRate := float64(txCount-lastTxCount) / (elapsed.Seconds() - lastLog.Seconds())
				hours, minutes, seconds = logger.ParseTime(elapsed)
				log.Infof("Elapsed time: %vh %vm %vs, at block %v (~ %.0f Tx/s, ~ %.0f Gas/s)", hours, minutes, seconds, tx.Block, txRate, f)
				lastLog = elapsed
				lastTxCount = txCount
				lastGasCount.Set(totalGas)
			}

			// Report progress on a regular block interval (simulation time).
			for tx.Block >= lastBlockProgressReportBlock+progressReportBlockInterval {
				numTransactions := txCount - lastBlockProgressReportTxCount
				lastBlockProgressReportTxCount = txCount

				gasUsed := new(big.Int).Sub(totalGas, lastBlockProgressReportGasCount)
				lastBlockProgressReportGasCount.Set(totalGas)

				now := time.Now()
				intervalTime := now.Sub(lastBlockProgressReportTime)
				lastBlockProgressReportTime = now

				txRate := float64(numTransactions) / intervalTime.Seconds()
				gasRate, _ := new(big.Float).SetInt(gasUsed).Float64()
				gasRate = gasRate / intervalTime.Seconds()

				memoryUsage := db.GetMemoryUsage()
				diskUsage := utils.GetDirectorySize(stateDbDir)

				log.Noticef("Reached block %d using ~ %d bytes of memory, ~ %d bytes of disk, last interval rate ~ %.0f Tx/s, ~ %.0f Gas/s",
					tx.Block, memoryUsage.UsedBytes, diskUsage, txRate, gasRate)
				lastBlockProgressReportBlock += progressReportBlockInterval
			}
			if cfg.Profile {
				if tx.Block >= lastProfileBlock+cfg.ProfileInterval {
					// print stats
					if err := stats.PrintProfiling(lastProfileBlock, lastProfileBlock+cfg.ProfileInterval); err != nil {
						return err
					}
					// reset
					stats.Reset()
					lastProfileBlock += cfg.ProfileInterval
				}
			}
		}
	}

	if !isFirstBlock && err == nil {
		db.EndBlock()
		db.EndSyncPeriod()
	}

	runTime := time.Since(start).Seconds()

	if cfg.ContinueOnFailure {
		log.Warningf("%v errors found", utils.NumErrors)
	}

	if cfg.ValidateWorldState && err == nil {
		log.Notice("Validate final state\n")
		if ws, err = utils.GenerateWorldStateFromUpdateDB(cfg, cfg.Last); err != nil {
			return err
		}
		if err := utils.ValidateStateDB(ws, db, false); err != nil {
			return fmt.Errorf("World state is not contained in the stateDB. %v", err)
		}
	}

	utils.MemoryBreakdown(db, cfg, log)

	// write memory profile if requested
	if err := utils.StartMemoryProfile(cfg); err != nil {
		return err
	}

	if cfg.Profile && curBlock != lastProfileBlock {
		if err := stats.PrintProfiling(lastProfileBlock, curBlock); err != nil {
			return err
		}
	}

	if cfg.KeepDb && !isFirstBlock {
		rootHash, _ := db.Commit(true)
		if err := utils.WriteStateDbInfo(stateDbDir, cfg, curBlock, rootHash); err != nil {
			log.Error(err)
		}
		//rename directory after closing db.
		defer utils.RenameTempStateDBDirectory(cfg, stateDbDir, curBlock)
	} else if cfg.KeepDb && isFirstBlock {
		// no blocks were processed.
		log.Warning("No blocks were processed. StateDB is not saved.")
		defer os.RemoveAll(stateDbDir)
	}

	// close the DB and print disk usage
	log.Info("Close StateDB")
	start = time.Now()
	if err := db.Close(); err != nil {
		log.Errorf("Failed to close database: %v", err)
	}

	// print progress summary
	if !cfg.Quiet {
		g := new(big.Float).Quo(new(big.Float).SetInt(totalGas), new(big.Float).SetFloat64(runTime))

		hours, minutes, seconds = logger.ParseTime(time.Since(beginning))

		log.Infof("Total elapsed time: %vh %vm %vs, processed %v blocks, %v transactions (~ %.1f Tx/s) (~ %.1f Gas/s)\n", hours, minutes, seconds, cfg.Last-cfg.First+1, txCount, float64(txCount)/(runTime), g)
		log.Infof("Closing DB took %v\n", time.Since(start))
		log.Infof("Final disk usage: %v MiB\n", float32(utils.GetDirectorySize(stateDbDir))/float32(1024*1024))
	}

	return err
}
