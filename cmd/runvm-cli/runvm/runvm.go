package runvm

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"os"
	"path/filepath"
	"time"

	"github.com/Fantom-foundation/Aida/tracer/operation"
	"github.com/Fantom-foundation/Aida/utils"
	substate "github.com/Fantom-foundation/Substate"
	lru "github.com/hashicorp/golang-lru"
	"github.com/urfave/cli/v2"

	"github.com/Fantom-foundation/go-opera-fvm/erigon"

	"github.com/ledgerwatch/erigon-lib/kv"
	estate "github.com/ledgerwatch/erigon/core/state"
	erigonethdb "github.com/ledgerwatch/erigon/ethdb"
	"github.com/ledgerwatch/erigon/ethdb/olddb"
)

// RunVM implements trace command for executing VM on a chosen storage system.
func RunVM(ctx *cli.Context) error {
	const progressReportBlockInterval uint64 = 100_000
	var (
		err          error
		start        time.Time
		sec          float64
		lastSec      float64
		txCount      int
		lastTxCount  int
		gasCount     = new(big.Int)
		lastGasCount = new(big.Int)
		// Progress reporting (block based)
		lastBlockProgressReportBlock    uint64
		lastBlockProgressReportTime     time.Time
		lastBlockProgressReportTxCount  int
		lastBlockProgressReportGasCount = new(big.Int)
	)
	// process general arguments
	cfg, argErr := utils.NewConfig(ctx, utils.BlockRangeArgs)
	cfg.StateValidationMode = utils.SubsetCheck
	if argErr != nil {
		return argErr
	}

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

	defer db.Close()

	if !cfg.KeepStateDB {
		log.Printf("WARNING: directory %v will be removed at the end of this run.\n", stateDirectory)
		defer os.RemoveAll(stateDirectory)
	}

	ws := substate.SubstateAlloc{}
	if cfg.SkipPriming || loadedExistingDB {
		log.Printf("Skipping DB priming.\n")
	} else {
		// load the world state
		log.Printf("Load and advance world state to block %v\n", cfg.First-1)
		start = time.Now()
		ws, err = utils.GenerateWorldStateFromUpdateDB(cfg, cfg.First-1)
		if err != nil {
			return err
		}
		sec = time.Since(start).Seconds()
		log.Printf("\tElapsed time: %.2f s, accounts: %v\n", sec, len(ws))

		// prime stateDB
		log.Printf("Prime stateDB \n")
		start = time.Now()

		utils.PrimeStateDB(ws, db, cfg)
		sec = time.Since(start).Seconds()
		log.Printf("\tElapsed time: %.2f s\n", sec)

		// delete destroyed accounts from stateDB
		log.Printf("Delete destroyed accounts \n")
		start = time.Now()
		// remove destroyed accounts until one block before the first block

		/*
			db.BeginEpoch(0)
			db.BeginBlock(target) // block 0 is the priming, block (first-1) the deletion
			db.BeginTransaction(0)
			for _, cur := range list {
				db.Suicide(cur)
			}
			db.Finalise(true)
			db.EndTransaction()
			db.EndBlock()
			db.EndEpoch()
		*/
		err = utils.DeleteDestroyedAccountsFromStateDB(db, cfg, cfg.First-1)
		sec = time.Since(start).Seconds()
		log.Printf("\tElapsed time: %.2f s\n", sec)
		if err != nil {
			return err
		}
	}

	// print memory usage after priming
	if cfg.MemoryBreakdown {
		if usage := db.GetMemoryUsage(); usage != nil {
			log.Printf("State DB memory usage: %d byte\n%s\n", usage.UsedBytes, usage.Breakdown)
		} else {
			log.Printf("Utilized storage solution does not support memory breakdowns.\n")
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
			return fmt.Errorf("Failed to remove deleted accoount from the world state. %v", err)
		}

		// a lot of db.GetState
		if err := utils.ValidateStateDB(ws, db, false); err != nil {
			return fmt.Errorf("Pre: World state is not contained in the stateDB. %v", err)
		}
	}

	// Release world state to free memory.
	ws = substate.SubstateAlloc{}

	if cfg.EnableProgress {
		start = time.Now()
		lastSec = time.Since(start).Seconds()
	}

	log.Printf("Run VM\n")
	var curBlock uint64 = 0
	var curEpoch uint64
	isFirstBlock := true

	// start erigon block execution
	var rwTx kv.RwTx
	var batch erigonethdb.DbWithPendingMutations
	const lruDefaultSize = 1_000_000 // 56 MB  // put it inside function
	if cfg.DbImpl == "erigon" {
		rwTx, err = db.DB().RwKV().BeginRw(context.Background())
		if err != nil {
			return err
		}

		defer rwTx.Rollback()

		// Contract code is unlikely to change too much, so let's keep it cached
		contractCodeCache, err := lru.New(lruDefaultSize)
		if err != nil {
			return err
		}

		// state is stored through ethdb batches
		whitelistedTables := []string{kv.Code, kv.ContractCode}
		batch = olddb.NewHashBatch(rwTx, nil, filepath.Join(stateDirectory, "erigon", "state"), whitelistedTables, contractCodeCache)
		defer batch.Rollback()
	}

	iter := substate.NewSubstateIterator(cfg.First, cfg.Workers)

	defer iter.Release()
	for iter.Next() {
		tx := iter.Value()
		// initiate first epoch and block.

		// initiate stateReader and stateWriter
		// wrap it into standalone function
		var (
			stateReader estate.StateReader
			stateWriter estate.WriterWithChangeSets
		)
		if cfg.DbImpl == "erigon" {
			stateReader = estate.NewPlainStateReader(batch)
			stateWriter = estate.NewPlainStateWriter(batch, rwTx, tx.Block)
		}

		if isFirstBlock {
			if tx.Block > cfg.Last {
				break
			}

			curEpoch = tx.Block / cfg.EpochLength
			curBlock = tx.Block
			db.BeginEpoch(curEpoch)
			db.BeginBlock(curBlock)
			lastBlockProgressReportBlock = tx.Block
			lastBlockProgressReportBlock -= lastBlockProgressReportBlock % progressReportBlockInterval
			lastBlockProgressReportTime = time.Now()
			isFirstBlock = false
			// close off old block and possibly epochs
		} else if curBlock != tx.Block { //curBlock 4564026, txc.Block 4564026 +1
			if tx.Block > cfg.Last {
				break
			}

			//curBlock = from, tx.Block = to

			// Mark the end of the old block.
			if cfg.DbImpl == "erigon" {
				from, to := curBlock, tx.Block
				if err := db.CommitBlock(stateWriter); err != nil {
					return fmt.Errorf("writing changesets for block %d failed: %w", tx.Block, err)
				}

				if err := stateWriter.WriteChangeSets(); err != nil {
					return fmt.Errorf("writing changesets for block %d failed: %w", tx.Block, err)
				}

				if err := erigon.PromoteHashedStateIncrementally("hashedstate", from, to, filepath.Join(stateDirectory, "erigon", "hashedstate"), rwTx, nil); err != nil {
					return err
				}

				if _, err := erigon.IncrementIntermediateHashes("IH", rwTx, from, to, filepath.Join(stateDirectory, "erigon", "IH"), false, nil); err != nil {
					return err
				}

			} else {
				db.SetTxBlock(tx.Block) // TODO later remove it
				db.EndBlock()
			}

			// Move on epochs if needed.
			newEpoch := tx.Block / cfg.EpochLength
			for curEpoch < newEpoch {
				if cfg.DbImpl != "erigon" {
					db.EndEpoch()
				}
				curEpoch++
				db.BeginEpoch(curEpoch)
			}
			// Mark the beginning of a new block
			curBlock = tx.Block
			db.BeginBlock(curBlock)
			if cfg.DbImpl != "erigon" {
				db.BeginBlockApplyWithStateReader(stateReader)
			} else {
				db.BeginBlockApply()
			}
			// new erigonAdfapter is nitiated
		}
		if cfg.MaxNumTransactions >= 0 && txCount >= cfg.MaxNumTransactions {
			break
		}
		// run VM
		db.PrepareSubstate(&tx.Substate.InputAlloc, tx.Substate.Env.Number)
		db.BeginTransaction(uint32(tx.Transaction))
		err = utils.ProcessTx(db, cfg, tx.Block, tx.Transaction, tx.Substate)
		if err != nil {
			log.Printf("\tRun VM failed.\n")
			err = fmt.Errorf("Error: VM execution failed. %w", err)
			break
		}

		db.EndTransaction()
		txCount++
		log.Println("txCount", txCount)
		gasCount = new(big.Int).Add(gasCount, new(big.Int).SetUint64(tx.Substate.Result.GasUsed))

		if cfg.EnableProgress {
			// report progress
			sec = time.Since(start).Seconds()

			// Report progress on a regular time interval (wall time).
			if sec-lastSec >= 15 {
				d := new(big.Int).Sub(gasCount, lastGasCount)
				g := new(big.Float).Quo(new(big.Float).SetInt(d), new(big.Float).SetFloat64(sec-lastSec))

				txRate := float64(txCount-lastTxCount) / (sec - lastSec)

				fmt.Printf("run-vm: Elapsed time: %.0f s, at block %v (~ %.1f Tx/s, ~ %.1f Gas/s)\n", sec, tx.Block, txRate, g)
				lastSec = sec
				lastTxCount = txCount
				lastGasCount.Set(gasCount)
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

				fmt.Printf("run-vm: Reached block %d, last interval rate ~ %.1f Tx/s, ~ %.1f Gas/s\n", tx.Block, txRate, gasRate)
				lastBlockProgressReportBlock += progressReportBlockInterval
			}
		}
	}

	if cfg.DbImpl == "erigon" {
		// finalize erigon execution
		if err = batch.Commit(); err != nil {
			return fmt.Errorf("batch commit: %v", err)
		}

		if err = rwTx.Commit(); err != nil {
			return err
		}
	} else {
		// end of execution
		if !isFirstBlock && err == nil {
			db.SetTxBlock(curBlock)
			db.EndBlock()
			db.EndEpoch()
		}
	}

	runTime := time.Since(start).Seconds()

	if cfg.ContinueOnFailure {
		log.Printf("run-vm: %v errors found\n", utils.NumErrors)
	}

	if cfg.ValidateWorldState && err == nil {
		log.Printf("Validate final state\n")
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
			log.Printf("State DB memory usage: %d byte\n%s\n", usage.UsedBytes, usage.Breakdown)
		} else {
			log.Printf("Utilized storage solution does not support memory breakdowns.\n")
		}
	}

	// write memory profile if requested
	if err := utils.StartMemoryProfile(cfg); err != nil {
		return err
	}

	if cfg.Profile {
		fmt.Printf("=================Statistics=================\n")
		stats.PrintProfiling()
		fmt.Printf("============================================\n")
	}

	if cfg.KeepStateDB && !isFirstBlock {
		rootHash, _ := db.Commit(true)
		if err := utils.WriteStateDbInfo(stateDirectory, cfg, curBlock, rootHash); err != nil {
			log.Println(err)
		}
		//rename directory after closing db.
		defer utils.RenameTempStateDBDirectory(cfg, stateDirectory, curBlock)
	} else if cfg.KeepStateDB && isFirstBlock {
		// no blocks were processed.
		log.Printf("No blocks were processed. StateDB is not saved.\n")
		defer os.RemoveAll(stateDirectory)
	}

	// close the DB and print disk usage
	log.Printf("Close StateDB database")
	start = time.Now()
	if err := db.Close(); err != nil {
		log.Printf("Failed to close database: %v", err)
	}

	// print progress summary
	if cfg.EnableProgress {
		g := new(big.Float).Quo(new(big.Float).SetInt(gasCount), new(big.Float).SetFloat64(runTime))

		log.Printf("run-vm: Total elapsed time: %.3f s, processed %v blocks, %v transactions (~ %.1f Tx/s) (~ %.1f Gas/s)\n", runTime, cfg.Last-cfg.First+1, txCount, float64(txCount)/(runTime), g)
		log.Printf("run-vm: Closing DB took %v\n", time.Since(start))
		log.Printf("run-vm: Final disk usage: %v MiB\n", float32(utils.GetDirectorySize(stateDirectory))/float32(1024*1024))
	}

	return err
}
