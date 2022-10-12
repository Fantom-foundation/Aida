package trace

import (
	"context"
	"fmt"
	"log"
	"math"
	"math/big"
	"os"

	"github.com/Fantom-foundation/aida/tracer"
	"github.com/Fantom-foundation/aida/tracer/dict"
	"github.com/Fantom-foundation/aida/tracer/operation"
	"github.com/Fantom-foundation/go-opera/evmcore"
	"github.com/Fantom-foundation/go-opera/opera"
	"github.com/Fantom-foundation/substate-cli/cmd/substate-cli/replay"
	"github.com/Fantom-foundation/substate-cli/state"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/substate"
	cli "gopkg.in/urfave/cli.v1"
)

// record-trace: trace record command
var TraceRecordCommand = cli.Command{
	Action:    traceRecordAction,
	Name:      "record",
	Usage:     "executes full state transitions and check output consistency",
	ArgsUsage: "<blockNumFirst> <blockNumLast>",
	Flags: []cli.Flag{
		substate.WorkersFlag,
		substate.SubstateDirFlag,
		ChainIDFlag,
		TraceDirectoryFlag,
		TraceDebugFlag,
	},
	Description: `
The substate-cli trace command requires two arguments:
<blockNumFirst> <blockNumLast>

<blockNumFirst> and <blockNumLast> are the first and
last block of the inclusive range of blocks to trace transactions.`,
}

// traceRecordTask generates storage traces for each transaction
func traceRecordTask(block uint64, tx int, recording *substate.Substate, dCtx *dict.DictionaryContext, ch chan operation.Operation) error {

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

	var statedb state.StateDB
	statedb = state.MakeInMemoryStateDB(&inputAlloc)
	statedb = tracer.NewProxyRecorder(statedb, dCtx, ch)

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
			replay.PrintResultDiffSummary(outputResult, evmResult)
		}
		if !a {
			fmt.Printf("inconsistent output: alloc\n")
			replay.PrintAllocationDiffSummary(&outputAlloc, &evmAlloc)
		}
		return fmt.Errorf("inconsistent output")
	}

	return nil
}

// Read state operations from channel and write them into a trace file.
func OperationWriter(ctx context.Context, done chan struct{}, ch chan operation.Operation, dCtx *dict.DictionaryContext, iCtx *tracer.IndexContext) {
	defer close(done)

	// open trace file
	file, err := os.OpenFile(tracer.TraceDir+"trace.dat", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		log.Fatalf("Cannot open trace file. Error: %v", err)
	}

	// read from channel until receiving cancel signal
	for {
		select {
		case op := <-ch:
			// print debug information
			if traceDebug {
				operation.Debug(dCtx, op)
			}

			// update index
			switch op.GetOpId() {
			case operation.BeginBlockID:
				// downcast op to BeginBlock for accessing block number
				tOp, ok := op.(*operation.BeginBlock)
				if !ok {
					log.Fatalf("Begin block operation downcasting failed")
				}
				// retrieve current file position
				offset, err := file.Seek(0, 1)
				if err != nil {
					log.Fatalf("Cannot retrieve current file position. Error: %v", err)
				}
				// add to block-index
				iCtx.AddBlock(tOp.BlockNumber, offset)
			}

			// write operation to file
			operation.WriteOperation(file, op)

		case <-ctx.Done():
			if len(ch) == 0 {
				return
			}
		}
	}

	// close trace file
	err = file.Close()
	if err != nil {
		log.Fatalf("Cannot close trace file. Error: %v", err)
	}
}

// record-trace: func traceAction for trace command
func traceRecordAction(ctx *cli.Context) error {
	var err error

	if len(ctx.Args()) != 2 {
		return fmt.Errorf("trace record command requires exactly 2 arguments")
	}

	dCtx := dict.NewDictionaryContext()
	iCtx := tracer.NewIndexContext()
	opChannel := make(chan operation.Operation, 10000)

	cctx, cancel := context.WithCancel(context.Background())
	cancelChannel := make(chan struct{})
	go OperationWriter(cctx, cancelChannel, opChannel, dCtx, iCtx)
	defer func() {
		// cancel writers
		(cancel)()        // stop writer
		<-(cancelChannel) // wait for writer to finish
	}()

	// process arguments
	chainID = ctx.Int(ChainIDFlag.Name)
	tracer.TraceDir = ctx.String(TraceDirectoryFlag.Name) + "/"
	dict.DictDir = ctx.String(TraceDirectoryFlag.Name) + "/"
	if ctx.Bool("trace-debug") {
		traceDebug = true
	}
	first, last, argErr := replay.SetBlockRange(ctx.Args().Get(0), ctx.Args().Get(1))
	if argErr != nil {
		return argErr
	}

	substate.SetSubstateFlags(ctx)
	substate.OpenSubstateDBReadOnly()
	defer substate.CloseSubstateDB()

	iter := substate.NewSubstateIterator(first, ctx.Int(substate.WorkersFlag.Name))
	defer iter.Release()
	oldBlock := uint64(math.MaxUint64) // set to an infeasible block
	for iter.Next() {
		tx := iter.Value()
		// close off old block with an end-block operation
		if oldBlock != tx.Block {
			if oldBlock != math.MaxUint64 {
				opChannel <- operation.NewEndBlock()
			}
			if tx.Block > last {
				break
			}
			oldBlock = tx.Block
			// open new block with a begin-block operation
			opChannel <- operation.NewBeginBlock(tx.Block)
		}
		traceRecordTask(tx.Block, tx.Transaction, tx.Substate, dCtx, opChannel)
		opChannel <- operation.NewEndTransaction()
	}

	// write dictionaries and indexes
	dCtx.Write()
	iCtx.Write()

	return err
}
