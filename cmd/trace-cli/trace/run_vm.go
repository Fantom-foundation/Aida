package trace

import (
	"fmt"
	"io/ioutil"
	"log"
	"math/big"
	"os"
	"runtime"
	"runtime/pprof"
	"time"

	"github.com/Fantom-foundation/Aida/tracer"
	"github.com/Fantom-foundation/Aida/tracer/operation"
	"github.com/Fantom-foundation/Aida/tracer/state"
	"github.com/Fantom-foundation/go-opera/evmcore"
	"github.com/Fantom-foundation/go-opera/opera"
	"github.com/Fantom-foundation/substate-cli/cmd/substate-cli/replay"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/substate"
	"github.com/urfave/cli/v2"
)

const MaxErrorCount = 50

var errCount int

// runVMCommand data structure for the record app
var RunVMCommand = cli.Command{
	Action:    runVM,
	Name:      "run-vm",
	Usage:     "run VM on the world-state",
	ArgsUsage: "<blockNumFirst> <blockNumLast>",
	Flags: []cli.Flag{
		&archiveModeFlag,
		&chainIDFlag,
		&continueOnFailureFlag,
		&cpuProfileFlag,
		&deletedAccountDirFlag,
		&disableProgressFlag,
		&epochLengthFlag,
		&memoryBreakdownFlag,
		&memProfileFlag,
		&primeSeedFlag,
		&primeThresholdFlag,
		&profileFlag,
		&randomizePrimingFlag,
		&stateDbImplementationFlag,
		&stateDbVariantFlag,
		&stateDbTempDirFlag,
		&stateDbLoggingFlag,
		&shadowDbImplementationFlag,
		&shadowDbVariantFlag,
		&substate.WorkersFlag,
		&substate.SubstateDirFlag,
		&updateDBDirFlag,
		&validateTxStateFlag,
		&validateWorldStateFlag,
		&validateFlag,
		&vmImplementation,
	},
	Description: `
The trace run-vm command requires two arguments:
<blockNumFirst> <blockNumLast>

<blockNumFirst> and <blockNumLast> are the first and
last block of the inclusive range of blocks to trace transactions.`,
}

// runVMTask executes VM on a chosen storage system.
func runVMTask(db state.StateDB, cfg *TraceConfig, block uint64, tx int, recording *substate.Substate) (*substate.SubstateResult, error) {

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
	vmConfig.InterpreterImpl = cfg.vmImpl

	// get chain configuration
	chainConfig := getChainConfig(cfg.chainID)

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
	if cfg.validateTxState {
		if err := validateStateDB(inputAlloc, db, true); err != nil {
			errCount++
			errMsg := fmt.Sprintf("Block: %v Transaction: %v\n", block, tx)
			errMsg += fmt.Sprintf("  Input alloc is not contained in the stateDB.\n%v", err)
			if cfg.continueOnFailure {
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
		if cfg.continueOnFailure {
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
	if cfg.validateTxState {
		if err := validateStateDB(outputAlloc, db, false); err != nil {
			errCount++
			errMsg := fmt.Sprintf("Block: %v Transaction: %v\n", block, tx)
			errMsg += fmt.Sprintf("  Output alloc is not contained in the stateDB. %v\n", err)
			if cfg.continueOnFailure {
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
			if cfg.continueOnFailure {
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

// runVM implements trace command for executing VM on a chosen storage system.
func runVM(ctx *cli.Context) error {
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
	cfg, argErr := NewTraceConfig(ctx)
	if argErr != nil {
		return argErr
	}

	// start CPU profiling if requested.
	if profileFileName := ctx.String(cpuProfileFlag.Name); profileFileName != "" {
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

	// create a directory for the store to place all its files.
	stateDirectory, err := ioutil.TempDir(cfg.stateDbDir, "state_db_*")
	if err != nil {
		return fmt.Errorf("Failed to create a temp directory. %v", err)
	}
	defer os.RemoveAll(stateDirectory)
	log.Printf("\tTemporary state DB directory: %v\n", stateDirectory)

	// instantiate the state DB under testing
	var db state.StateDB
	db, err = MakeStateDB(stateDirectory, cfg)
	if err != nil {
		return err
	}

	// load the world state
	log.Printf("Load and advance world state to block %v\n", cfg.first-1)
	start = time.Now()
	ws, err := generateWorldStateFromUpdateDB(cfg.updateDBDir, cfg.first-1, cfg.workers)
	if err != nil {
		return err
	}
	sec = time.Since(start).Seconds()
	log.Printf("\tElapsed time: %.2f s, accounts: %v\n", sec, len(ws))

	// prime stateDB
	log.Printf("Prime stateDB \n")
	start = time.Now()
	primeStateDB(ws, db, cfg)
	sec = time.Since(start).Seconds()
	log.Printf("\tElapsed time: %.2f s\n", sec)

	// delete destroyed accounts from stateDB
	log.Printf("Delete destroyed accounts \n")
	start = time.Now()
	// remove destroyed accounts until one block before the first block
	err = deleteDestroyedAccountsFromStateDB(db, cfg.deletedAccountDir, cfg.first-1)
	sec = time.Since(start).Seconds()
	log.Printf("\tElapsed time: %.2f s\n", sec)
	if err != nil {
		return err
	}

	// print memory usage after priming
	if cfg.memoryBreakdown {
		if usage := db.GetMemoryUsage(); usage != nil {
			log.Printf("State DB memory usage: %d byte\n%s\n", usage.UsedBytes, usage.Breakdown)
		} else {
			log.Printf("Utilized storage solution does not support memory breakdowns.\n")
		}
	}

	// wrap stateDB for profiling
	var stats *operation.ProfileStats
	if cfg.profile {
		db, stats = tracer.NewProxyProfiler(db)
	}

	if cfg.validateWorldState {
		if err := deleteDestroyedAccountsFromWorldState(ws, cfg.deletedAccountDir, cfg.first-1); err != nil {
			return fmt.Errorf("Failed to remove deleted accoount from the world state. %v", err)
		}
		if err := validateStateDB(ws, db, false); err != nil {
			return fmt.Errorf("Pre: World state is not contained in the stateDB. %v", err)
		}
	} else {
		// Release world state to free memory.
		ws = substate.SubstateAlloc{}
	}

	if cfg.enableProgress {
		start = time.Now()
		lastSec = time.Since(start).Seconds()
	}

	log.Printf("Run VM\n")
	var curBlock uint64 = 0
	var curEpoch uint64
	isFirstBlock := true
	iter := substate.NewSubstateIterator(cfg.first, cfg.workers)

	defer iter.Release()
	for iter.Next() {
		tx := iter.Value()
		// initiate first epoch and block.
		if isFirstBlock {
			curEpoch = tx.Block / cfg.epochLength
			curBlock = tx.Block
			db.BeginEpoch(curEpoch)
			db.BeginBlock(curBlock)
			lastBlockProgressReportBlock = tx.Block
			lastBlockProgressReportBlock -= lastBlockProgressReportBlock % progressReportBlockInterval
			lastBlockProgressReportTime = time.Now()
			isFirstBlock = false
			// close off old block and possibly epochs
		} else if curBlock != tx.Block {
			if tx.Block > cfg.last {
				break
			}

			// Mark the end of the old block.
			db.EndBlock()

			// Move on epochs if needed.
			newEpoch := tx.Block / cfg.epochLength
			for curEpoch < newEpoch {
				db.EndEpoch()
				curEpoch++
				db.BeginEpoch(curEpoch)
			}
			// Mark the begin of a new block
			curBlock = tx.Block
			db.BeginBlock(curBlock)
			db.BeginBlockApply()
		}

		// run VM
		db.PrepareSubstate(&tx.Substate.InputAlloc)
		db.BeginTransaction(uint32(tx.Transaction))
		result, err := runVMTask(db, cfg, tx.Block, tx.Transaction, tx.Substate)
		if err != nil {
			return fmt.Errorf("VM execution failed. %v", err)
		}
		db.EndTransaction()
		txCount++
		gasCount = new(big.Int).Add(gasCount, new(big.Int).SetUint64(result.GasUsed))

		if cfg.enableProgress {
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

	if !isFirstBlock {
		db.EndBlock()
		db.EndEpoch()
	}

	runTime := time.Since(start).Seconds()

	if cfg.continueOnFailure {
		fmt.Printf("run-vm: %v errors found\n", errCount)
	}

	if cfg.validateWorldState {
		log.Printf("Validate final state\n")
		advanceWorldState(ws, cfg.first, cfg.last, cfg.workers)
		if err := deleteDestroyedAccountsFromWorldState(ws, cfg.deletedAccountDir, cfg.last); err != nil {
			return fmt.Errorf("Failed to remove deleted accoount from the world state. %v", err)
		}
		if err := validateStateDB(ws, db, false); err != nil {
			return fmt.Errorf("World state is not contained in the stateDB. %v", err)
		}
	}

	if cfg.memoryBreakdown {
		if usage := db.GetMemoryUsage(); usage != nil {
			log.Printf("State DB memory usage: %d byte\n%s\n", usage.UsedBytes, usage.Breakdown)
		} else {
			log.Printf("Utilized storage solution does not support memory breakdowns.\n")
		}
	}

	if cfg.profile {
		stats.PrintProfiling()
	}

	// close the DB and print disk usage
	log.Printf("Close StateDB database")
	start = time.Now()
	if err := db.Close(); err != nil {
		log.Printf("Failed to close database: %v", err)
	}

	// print progress summary
	if cfg.enableProgress {
		g := new(big.Float).Quo(new(big.Float).SetInt(gasCount), new(big.Float).SetFloat64(runTime))

		log.Printf("run-vm: Total elapsed time: %.3f s, processed %v blocks (~ %.1f Tx/s) (~ %.1f Gas/s)\n", runTime, cfg.last-cfg.first+1, float64(txCount)/(runTime), g)
		log.Printf("run-vm: Closing DB took %v\n", time.Since(start))
		log.Printf("run-vm: Final disk usage: %v MiB\n", float32(getDirectorySize(stateDirectory))/float32(1024*1024))
	}

	// write memory profile if requested
	if profileFileName := ctx.String(memProfileFlag.Name); profileFileName != "" && err == nil {
		f, err := os.Create(profileFileName)
		if err != nil {
			return fmt.Errorf("could not create memory profile: %s", err)
		}
		runtime.GC() // get up-to-date statistics
		if err := pprof.WriteHeapProfile(f); err != nil {
			return fmt.Errorf("could not write memory profile: %s", err)
		}
	}

	return err
}
