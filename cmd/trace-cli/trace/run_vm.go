package trace

import (
	"fmt"
	"io/ioutil"
	"math/big"
	"os"
	"runtime/pprof"
	"time"

	"github.com/Fantom-foundation/Aida/tracer"
	"github.com/Fantom-foundation/Aida/tracer/state"
	"github.com/Fantom-foundation/go-opera/evmcore"
	"github.com/Fantom-foundation/go-opera/opera"
	"github.com/Fantom-foundation/substate-cli/cmd/substate-cli/replay"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/substate"
	"github.com/urfave/cli/v2"
)

// runVMCommand data structure for the record app
var RunVMCommand = cli.Command{
	Action:    runVM,
	Name:      "run-vm",
	Usage:     "run VM on the world-state",
	ArgsUsage: "<blockNumFirst> <blockNumLast>",
	Flags: []cli.Flag{
		&chainIDFlag,
		&cpuProfileFlag,
		&disableProgressFlag,
		&profileFlag,
		&stateDbImplementation,
		&stateDbVariant,
		&substate.WorkersFlag,
		&substate.SubstateDirFlag,
		&validateEndState,
		&worldStateDirFlag,
	},
	Description: `
The trace record command requires two arguments:
<blockNumFirst> <blockNumLast>

<blockNumFirst> and <blockNumLast> are the first and
last block of the inclusive range of blocks to trace transactions.`,
}

var profile bool

// runVMTask executes VM on a chosen storage system.
func runVMTask(db state.StateDB, block uint64, tx int, recording *substate.Substate) error {

	inputAlloc := recording.InputAlloc
	inputEnv := recording.Env
	inputMessage := recording.Message

	outputAlloc := recording.OutputAlloc
	outputResult := recording.Result

	var (
		vmConfig    vm.Config
		chainConfig *params.ChainConfig
	)

	vmConfig = opera.DefaultVMConfig
	vmConfig.NoBaseFee = true

	// mainnet chain configuration
	chainConfig = params.AllEthashProtocolChanges
	chainConfig.ChainID = big.NewInt(int64(chainID))
	chainConfig.LondonBlock = new(big.Int).SetUint64(37534833)
	chainConfig.BerlinBlock = new(big.Int).SetUint64(37455223)

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
	if err := validateStateDB(inputAlloc, db); err != nil {
		msg := fmt.Sprintf("Block: %v Transaction: %v\n", block, tx)
		return fmt.Errorf(msg+"Input alloc is not contained in the stateDB. %v", err)
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

	msg := inputMessage.AsMessage()

	db.Prepare(txHash, txIndex)

	txCtx := evmcore.NewEVMTxContext(msg)

	evm := vm.NewEVM(blockCtx, txCtx, db, chainConfig, vmConfig)

	snapshot := db.Snapshot()

	msgResult, err := evmcore.ApplyMessage(evm, msg, gaspool)

	if err != nil {
		db.RevertToSnapshot(snapshot)
		return err
	}
	if hashError != nil {
		return hashError
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
	if err := validateStateDB(outputAlloc, db); err != nil {
		msg := fmt.Sprintf("Block: %v Transaction: %v\n", block, tx)
		return fmt.Errorf(msg+"Output alloc is not contained in the stateDB. %v", err)
	}
	r := outputResult.Equal(evmResult)
	if !r {
		fmt.Printf("Block: %v Transaction: %v\n", block, tx)
		fmt.Printf("inconsistent output: result\n")
		replay.PrintResultDiffSummary(outputResult, evmResult)
		return fmt.Errorf("inconsistent output")
	}
	return nil
}

// runVM implements trace command for executing VM on a chosen storage system.
func runVM(ctx *cli.Context) error {
	var err error

	cfg, argErr := NewTraceConfig(ctx)
	if argErr != nil {
		return argErr
	}
	profile = ctx.Bool(profileFlag.Name)

	// Start CPU profiling if requested.
	if profileFileName := ctx.String(cpuProfileFlag.Name); profileFileName != "" {
		f, err := os.Create(profileFileName)
		if err != nil {
			return err
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	// process arguments
	chainID = ctx.Int(chainIDFlag.Name)

	// iterate through subsets in sequence
	substate.SetSubstateFlags(ctx)
	substate.OpenSubstateDBReadOnly()
	defer substate.CloseSubstateDB()

	fmt.Printf("trace replay: Load and advance worldstate to block %v\n", cfg.first-1)
	ws, err := generateWorldState(cfg.worldStateDir, cfg.first-1, cfg.workers)
	if err != nil {
		return err
	}

	// create a directory for the store to place all its files.
	stateDirectory, err := ioutil.TempDir("", "state_db_*")
	if err != nil {
		return err
	}
	// instantiate the state DB under testing
	var db state.StateDB
	db, err = makeStateDB(stateDirectory, cfg.impl, cfg.variant)
	if err != nil {
		return err
	}
	profileStateDB := tracer.NewProxyProfiler(db)
	db = profileStateDB

	// prime stateDB
	fmt.Printf("trace replay: Prime stateDB\n")
	if cfg.impl == "memory" {
		db.PrepareSubstate(&ws)
	} else {
		primeStateDB(ws, db)
	}
	if cfg.enableValidation {
		if err := validateStateDB(ws, db); err != nil {
			return fmt.Errorf("Pre: World state is not contained in the stateDB. %v", err)
		}
	}

	var (
		start   time.Time
		sec     float64
		lastSec float64
	)
	if cfg.enableProgress {
		start = time.Now()
		sec = time.Since(start).Seconds()
		lastSec = time.Since(start).Seconds()
	}

	iter := substate.NewSubstateIterator(cfg.first, cfg.workers)
	defer iter.Release()
	for iter.Next() {
		tx := iter.Value()
		// close off old block with an end-block operation
		if tx.Block > cfg.last {
			break
		}
		if err := runVMTask(db, tx.Block, tx.Transaction, tx.Substate); err != nil {
			return fmt.Errorf("VM execution failed. %v", err)
		}
		if cfg.enableProgress {
			// report progress
			sec = time.Since(start).Seconds()
			if sec-lastSec >= 15 {
				fmt.Printf("trace record: Elapsed time: %.0f s, at block %v\n", sec, tx.Block)
				lastSec = sec
			}
		}
	}

	if cfg.enableProgress {
		sec = time.Since(start).Seconds()
		fmt.Printf("trace record: Total elapsed time: %.3f s, processed %v blocks\n", sec, cfg.last-cfg.first+1)
	}
	if cfg.enableValidation {
		advanceWorldState(ws, cfg.first, cfg.last, cfg.workers)
		if err := validateStateDB(ws, db); err != nil {
			return fmt.Errorf("World state is not contained in the stateDB. %v", err)
		}
	}
	return err
}
