package stochastic

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"math/big"
	"os"
	"runtime/pprof"
	"time"

	replay2 "github.com/Fantom-foundation/Aida/cmd/substate-cli/replay"

	"github.com/Fantom-foundation/Aida/stochastic"
	"github.com/Fantom-foundation/Aida/substate-cli/state"
	"github.com/Fantom-foundation/Aida/utils"
	"github.com/Fantom-foundation/go-opera/evmcore"
	"github.com/Fantom-foundation/go-opera/opera"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/substate"
	"github.com/urfave/cli/v2"
)

// StochasticRecordCommand data structure for the record app
var StochasticRecordCommand = cli.Command{
	Action:    stochasticRecordAction,
	Name:      "record",
	Usage:     "record StateDB events while processing blocks",
	ArgsUsage: "<blockNumFirst> <blockNumLast>",
	Flags: []cli.Flag{
		&utils.CpuProfileFlag,
		&utils.DisableProgressFlag,
		&utils.OutputFlag,
		&substate.WorkersFlag,
		&substate.SubstateDirFlag,
		&utils.ChainIDFlag,
	},
	Description: `
The stochastic record command requires two arguments:
<blockNumFirst> <blockNumLast>

<blockNumFirst> and <blockNumLast> are the first and
last block for recording events.`,
}

// stochasticRecordTask generates events for a transaction.
func stochasticRecordTask(block uint64, tx, chainID int, recording *substate.Substate, eventRegistry *stochastic.EventRegistry) error {

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
	chainConfig = utils.GetChainConfig(chainID)

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

	var statedb state.StateDB = stochastic.NewEventProxy(state.MakeInMemoryStateDB(&inputAlloc, inputEnv.Number), eventRegistry)

	// Apply Message
	var (
		gaspool   = new(evmcore.GasPool)
		blockHash = common.Hash{0x01}
		txHash    = common.Hash{0x02}
		txIndex   = tx
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

	statedb.Prepare(txHash, txIndex)

	txCtx := evmcore.NewEVMTxContext(msg)

	evm := vm.NewEVM(blockCtx, txCtx, statedb, chainConfig, vmConfig)

	snapshot := statedb.Snapshot()

	msgResult, err := evmcore.ApplyMessage(evm, msg, gaspool)
	if err != nil {
		statedb.RevertToSnapshot(snapshot)
		return err
	}
	if hashError != nil {
		return hashError
	}
	if chainConfig.IsByzantium(blockCtx.BlockNumber) {
		statedb.Finalise(true)
	} else {
		statedb.IntermediateRoot(chainConfig.IsEIP158(blockCtx.BlockNumber))
	}

	evmResult := &substate.SubstateResult{}
	if msgResult.Failed() {
		evmResult.Status = types.ReceiptStatusFailed
	} else {
		evmResult.Status = types.ReceiptStatusSuccessful
	}
	evmResult.Logs = statedb.GetLogs(txHash, blockHash)
	evmResult.Bloom = types.BytesToBloom(types.LogsBloom(evmResult.Logs))
	if to := msg.To(); to == nil {
		evmResult.ContractAddress = crypto.CreateAddress(evm.TxContext.Origin, msg.Nonce())
	}
	evmResult.GasUsed = msgResult.UsedGas
	evmAlloc := statedb.GetSubstatePostAlloc()

	r := outputResult.Equal(evmResult)
	a := outputAlloc.Equal(evmAlloc)
	if !(r && a) {
		fmt.Printf("Block: %v Transaction: %v\n", block, tx)
		if !r {
			fmt.Printf("inconsistent output: result\n")
			replay2.PrintResultDiffSummary(outputResult, evmResult)
		}
		if !a {
			fmt.Printf("inconsistent output: alloc\n")
			replay2.PrintAllocationDiffSummary(&outputAlloc, &evmAlloc)
		}
		return fmt.Errorf("inconsistent output")
	}

	return nil
}

// stochasticRecordAction implements recording of events.
func stochasticRecordAction(ctx *cli.Context) error {
	substate.RecordReplay = true
	var err error

	if ctx.Args().Len() != 2 {
		return fmt.Errorf("stochastic record command requires exactly 2 arguments")
	}

	// start CPU profiling if enabled.
	if profileFileName := ctx.String(utils.CpuProfileFlag.Name); profileFileName != "" {
		f, err := os.Create(profileFileName)
		if err != nil {
			return err
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	// get progress flag
	enableProgress := !ctx.Bool(utils.DisableProgressFlag.Name)

	// process arguments
	chainID := ctx.Int(utils.ChainIDFlag.Name)
	if ctx.Bool(utils.TraceDebugFlag.Name) {
		utils.TraceDebug = true
	}
	first, last, argErr := replay2.SetBlockRange(ctx.Args().Get(0), ctx.Args().Get(1))
	if argErr != nil {
		return argErr
	}

	// iterate through subsets in sequence
	substate.SetSubstateFlags(ctx)
	substate.OpenSubstateDBReadOnly()
	defer substate.CloseSubstateDB()
	iter := substate.NewSubstateIterator(first, ctx.Int(substate.WorkersFlag.Name))
	defer iter.Release()
	oldBlock := uint64(math.MaxUint64) // set to an infeasible block
	var (
		start   time.Time
		sec     float64
		lastSec float64
	)
	if enableProgress {
		start = time.Now()
		sec = time.Since(start).Seconds()
		lastSec = time.Since(start).Seconds()
	}

	// create a new event registry
	eventRegistry := stochastic.NewEventRegistry()

	// iterate over all substates in order
	for iter.Next() {
		tx := iter.Value()
		// close off old block with an end-block operation
		if oldBlock != tx.Block {
			if tx.Block > last {
				break
			}
			oldBlock = tx.Block
		}
		stochasticRecordTask(tx.Block, tx.Transaction, chainID, tx.Substate, &eventRegistry)
		if enableProgress {
			// report progress
			sec = time.Since(start).Seconds()
			if sec-lastSec >= 15 {
				fmt.Printf("stochastic record: Elapsed time: %.0f s, at block %v\n", sec, oldBlock)
				lastSec = sec
			}
		}

	}
	if enableProgress {
		sec = time.Since(start).Seconds()
		fmt.Printf("stochastic record: Total elapsed time: %.3f s, processed %v blocks\n", sec, last-first+1)
	}

	// writing event registry
	fmt.Printf("stochastic record: write events file ...\n")
	outputFileName := ctx.String(utils.OutputFlag.Name)
	if outputFileName == "" {
		outputFileName = "./events.json"
	}
	WriteEvents(&eventRegistry, outputFileName)

	return err
}

// WriteEvent writes event file in JSON format.
func WriteEvents(r *stochastic.EventRegistry, filename string) {
	f, fErr := os.Create(filename)
	if fErr != nil {
		log.Fatalf("cannot open JSON file. Error: %v", fErr)
	}
	defer f.Close()

	jOut, jErr := json.MarshalIndent(r.NewEventRegistryJSON(), "", "    ")
	if jErr != nil {
		log.Fatalf("failed to convert JSON file. Error: %v", jErr)
	}

	_, pErr := fmt.Fprintln(f, string(jOut))
	if pErr != nil {
		log.Fatalf("failed to convert JSON file. Error: %v", pErr)
	}
}
