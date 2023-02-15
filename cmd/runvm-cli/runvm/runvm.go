package runvm

import (
	"fmt"
	"log"
	"math/big"
	"os"
	"runtime"
	"runtime/pprof"
	"time"

	"github.com/Fantom-foundation/Aida/cmd/substate-cli/replay"
	"github.com/Fantom-foundation/Aida/state"
	"github.com/Fantom-foundation/Aida/tracer/operation"
	"github.com/Fantom-foundation/Aida/utils"
	"github.com/Fantom-foundation/go-opera/evmcore"
	"github.com/Fantom-foundation/go-opera/opera"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/substate"
	"github.com/urfave/cli/v2"
)

const MaxErrorCount = 50 // maximum number of errors before terminating program
var errCount int         // number of errors encountered

// runVMTask executes VM on a chosen storage system.
func RunVMTask(db state.StateDB, cfg *utils.Config, block uint64, tx int, recording *substate.Substate) (*substate.SubstateResult, error) {

	inputAlloc := recording.InputAlloc
	inputEnv := recording.Env
	inputMessage := recording.Message

	outputAlloc := recording.OutputAlloc
	outputResult := recording.Result

	var (
		vmConfig vm.Config
	)

	vmConfig = opera.DefaultVMConfig
	vmConfig.NoBaseFee = true
	vmConfig.InterpreterImpl = cfg.VmImpl

	// get chain configuration
	chainConfig := utils.GetChainConfig(cfg.ChainID)

	var hashError error
	getHash := func(num uint64) common.Hash {
		if inputEnv.BlockHashes == nil {
			hashError = fmt.Errorf("getHash(%d) invoked, no blockhashes provided", num)
			return common.Hash{}
		}
		h, ok := inputEnv.BlockHashes[num]
		if !ok {
			hashError = fmt.Errorf("getHash(%d) invoked, blockhash for that block not provided", num)
		}
		return h
	}

	// validate whether the input alloc is contained in the db
	if cfg.ValidateTxState {
		if err := utils.ValidateStateDB(inputAlloc, db, true); err != nil {
			errCount++
			errMsg := fmt.Sprintf("Block: %v Transaction: %v\n", block, tx)
			errMsg += fmt.Sprintf("  Input alloc is not contained in the stateDB.\n%v", err)
			if cfg.ContinueOnFailure {
				fmt.Println(errMsg)
			} else {
				return nil, fmt.Errorf(errMsg)
			}
		}
	}

	// Apply Message
	var (
		gaspool = new(evmcore.GasPool)
		//TODO check logs
		//blockHash = common.Hash{0x01}
		txHash  = common.Hash{0x02}
		txIndex = tx
	)

	gaspool.AddGas(inputEnv.GasLimit)
	blockCtx := vm.BlockContext{
		CanTransfer: core.CanTransfer,
		Transfer:    core.Transfer,
		Coinbase:    inputEnv.Coinbase,
		BlockNumber: new(big.Int).SetUint64(inputEnv.Number),
		Time:        new(big.Int).SetUint64(inputEnv.Timestamp),
		Difficulty:  inputEnv.Difficulty,
		GasLimit:    inputEnv.GasLimit,
		GetHash:     getHash,
	}
	// If currentBaseFee is defined, add it to the vmContext.
	if inputEnv.BaseFee != nil {
		blockCtx.BaseFee = new(big.Int).Set(inputEnv.BaseFee)
	}

	// call ApplyMessage
	msg := inputMessage.AsMessage()
	db.Prepare(txHash, txIndex)
	txCtx := evmcore.NewEVMTxContext(msg)
	evm := vm.NewEVM(blockCtx, txCtx, db, chainConfig, vmConfig)
	snapshot := db.Snapshot()
	msgResult, err := evmcore.ApplyMessage(evm, msg, gaspool)

	// if transaction fails, revert to the first snapshot.
	if err != nil {
		db.RevertToSnapshot(snapshot)
		errCount++
		if cfg.ContinueOnFailure {
			fmt.Printf("Block: %v Transaction: %v\n%v", block, tx, err)
		} else {
			return nil, fmt.Errorf("Block: %v Transaction: %v\n%v", block, tx, err)
		}
	}
	if hashError != nil {
		return nil, hashError
	}
	if chainConfig.IsByzantium(blockCtx.BlockNumber) {
		db.Finalise(true)
	} else {
		db.IntermediateRoot(chainConfig.IsEIP158(blockCtx.BlockNumber))
	}

	evmResult := &substate.SubstateResult{}
	if msgResult.Failed() {
		evmResult.Status = types.ReceiptStatusFailed
	} else {
		evmResult.Status = types.ReceiptStatusSuccessful
	}

	// TODO clear state execution context and validate logs
	//evmResult.Logs = db.GetLogs(txHash, blockHash)
	evmResult.Logs = outputResult.Logs
	evmResult.Bloom = types.BytesToBloom(types.LogsBloom(evmResult.Logs))
	if to := msg.To(); to == nil {
		evmResult.ContractAddress = crypto.CreateAddress(evm.TxContext.Origin, msg.Nonce())
	}
	evmResult.GasUsed = msgResult.UsedGas

	// check whether the outputAlloc substate is contained in the world-state db.
	if cfg.ValidateTxState {
		if err := utils.ValidateStateDB(outputAlloc, db, false); err != nil {
			errCount++
			errMsg := fmt.Sprintf("Block: %v Transaction: %v\n", block, tx)
			errMsg += fmt.Sprintf("  Output alloc is not contained in the stateDB. %v\n", err)
			if cfg.ContinueOnFailure {
				fmt.Println(errMsg)
			} else {
				return nil, fmt.Errorf(errMsg)
			}
		}
		r := outputResult.Equal(evmResult)
		if !r {
			errCount++
			errMsg := fmt.Sprintf("Block: %v Transaction: %v\n", block, tx)
			errMsg += fmt.Sprintf("  Inconsistent output result.\n")
			replay.PrintResultDiffSummary(outputResult, evmResult)
			if cfg.ContinueOnFailure {
				fmt.Println(errMsg)
			} else {
				return nil, fmt.Errorf(errMsg)
			}
		}
	}
	if errCount > MaxErrorCount {
		return nil, fmt.Errorf("too many errors")
	}
	return evmResult, nil
}

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
	if argErr != nil {
		return argErr
	}

	// start CPU profiling if requested.
	if profileFileName := ctx.String(utils.CpuProfileFlag.Name); profileFileName != "" {
		f, err := os.Create(profileFileName)
		if err != nil {
			return fmt.Errorf("could not create CPU profile: %s", err)
		}
		if err := pprof.StartCPUProfile(f); err != nil {
			return fmt.Errorf("could not start CPU profile: %s", err)
		}
		defer pprof.StopCPUProfile()
	}

	// iterate through subsets in sequence
	substate.SetSubstateFlags(ctx)
	substate.OpenSubstateDBReadOnly()
	defer substate.CloseSubstateDB()

	db, stateDirectory, loadedExistingDB, err := utils.PrepareStateDB(cfg)
	if err != nil {
		return err
	}
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
	iter := substate.NewSubstateIterator(cfg.First, cfg.Workers)

	defer iter.Release()
	for iter.Next() {
		tx := iter.Value()
		// initiate first epoch and block.
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
		} else if curBlock != tx.Block {
			if tx.Block > cfg.Last {
				break
			}

			// Mark the end of the old block.
			db.EndBlock()

			// Move on epochs if needed.
			newEpoch := tx.Block / cfg.EpochLength
			for curEpoch < newEpoch {
				db.EndEpoch()
				curEpoch++
				db.BeginEpoch(curEpoch)
			}
			// Mark the beginning of a new block
			curBlock = tx.Block
			db.BeginBlock(curBlock)
			db.BeginBlockApply()
		}
		if cfg.MaxNumTransactions >= 0 && txCount >= cfg.MaxNumTransactions {
			break
		}
		// run VM
		db.PrepareSubstate(&tx.Substate.InputAlloc, tx.Substate.Env.Number)
		db.BeginTransaction(uint32(tx.Transaction))
		var result *substate.SubstateResult
		result, err = RunVMTask(db, cfg, tx.Block, tx.Transaction, tx.Substate)
		if err != nil {
			log.Printf("\tRun VM failed.\n")
			err = fmt.Errorf("Error: VM execution failed. %w", err)
			break
		}
		db.EndTransaction()
		txCount++
		gasCount = new(big.Int).Add(gasCount, new(big.Int).SetUint64(result.GasUsed))

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

	if !isFirstBlock && err == nil {
		db.EndBlock()
		db.EndEpoch()
	}

	runTime := time.Since(start).Seconds()

	if cfg.ContinueOnFailure {
		log.Printf("run-vm: %v errors found\n", errCount)
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
	if profileFileName := ctx.String(utils.MemProfileFlag.Name); profileFileName != "" && err == nil {
		f, err := os.Create(profileFileName)
		if err != nil {
			return fmt.Errorf("could not create memory profile: %s", err)
		}
		runtime.GC() // get up-to-date statistics
		if err := pprof.WriteHeapProfile(f); err != nil {
			return fmt.Errorf("could not write memory profile: %s", err)
		}
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
