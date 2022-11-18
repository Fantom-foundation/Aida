package trace

import (
	"bufio"
	"fmt"
	"log"
	"math"
	"math/big"
	"os"
	"runtime/pprof"
	"time"

	"github.com/Fantom-foundation/Aida/tracer"
	"github.com/Fantom-foundation/Aida/tracer/dict"
	"github.com/Fantom-foundation/Aida/tracer/operation"
	"github.com/Fantom-foundation/go-opera/evmcore"
	"github.com/Fantom-foundation/go-opera/opera"
	"github.com/Fantom-foundation/substate-cli/cmd/substate-cli/replay"
	"github.com/Fantom-foundation/substate-cli/state"
	"github.com/dsnet/compress/bzip2"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/substate"
	"github.com/urfave/cli/v2"
)

// TraceRecordCommand data structure for the record app
var TraceRecordCommand = cli.Command{
	Action:    traceRecordAction,
	Name:      "record",
	Usage:     "captures and records StateDB operations while processing blocks",
	ArgsUsage: "<blockNumFirst> <blockNumLast>",
	Flags: []cli.Flag{
		&cpuProfileFlag,
		&epochLengthFlag,
		&disableProgressFlag,
		&substate.WorkersFlag,
		&substate.SubstateDirFlag,
		&chainIDFlag,
		&traceDirectoryFlag,
		&traceDebugFlag,
	},
	Description: `
The trace record command requires two arguments:
<blockNumFirst> <blockNumLast>

<blockNumFirst> and <blockNumLast> are the first and
last block of the inclusive range of blocks to trace transactions.`,
}

// traceRecordTask generates storage traces for a transaction.
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
	statedb = tracer.NewProxyRecorder(statedb, dCtx, ch, traceDebug)

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

// OperationWriter reads operations from the operation channel and writes
// them into a trace file.
func OperationWriter(ch chan operation.Operation) {

	// open trace file, write buffer, and compressed stream
	file, err := os.OpenFile(tracer.TraceDir+"trace.dat", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		log.Fatalf("Cannot open trace file. Error: %v", err)
	}
	bfile := bufio.NewWriterSize(file, 65536*16)
	zfile, err := bzip2.NewWriter(bfile, &bzip2.WriterConfig{Level: 9})
	if err != nil {
		log.Fatalf("Cannot open bzip2 stream. Error: %v", err)
	}

	// defer closing compressed stream, flushing buffer, and closing trace file
	defer func() {
		if err := zfile.Close(); err != nil {
			log.Fatalf("Cannot close bzip2 writer. Error: %v", err)
		}
		if err := bfile.Flush(); err != nil {
			log.Fatalf("Cannot flush buffer. Error: %v", err)
		}
		if err := file.Close(); err != nil {
			log.Fatalf("Cannot close trace file. Error: %v", err)
		}
	}()

	// read operations from channel, and write them
	for op := range ch {
		operation.Write(zfile, op)
	}
}

// sendOperation sends an operation onto the channel.
func sendOperation(dCtx *dict.DictionaryContext, ch chan operation.Operation, op operation.Operation) {
	ch <- op
	if traceDebug {
		operation.Debug(dCtx, op)
	}
}

// traceRecordAction implements trace command for recording.
func traceRecordAction(ctx *cli.Context) error {
	substate.RecordReplay = true
	var err error

	if ctx.Args().Len() != 2 {
		return fmt.Errorf("trace record command requires exactly 2 arguments")
	}

	// Fetch length of epoch from command line.
	epochLength := ctx.Uint64(epochLengthFlag.Name)
	if epochLength <= 0 {
		epochLength = 300
	}
	log.Printf("Using epoch length of %d blocks\n", epochLength)

	// start CPU profiling if enabled.
	if profileFileName := ctx.String(cpuProfileFlag.Name); profileFileName != "" {
		f, err := os.Create(profileFileName)
		if err != nil {
			return err
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	// get progress flag
	enableProgress := !ctx.Bool(disableProgressFlag.Name)

	// create dictionary and index contexts
	dCtx := dict.NewDictionaryContext()

	// spawn writer
	opChannel := make(chan operation.Operation, 100000)
	go OperationWriter(opChannel)

	// process arguments
	chainID = ctx.Int(chainIDFlag.Name)
	tracer.TraceDir = ctx.String(traceDirectoryFlag.Name) + "/"
	dict.DictionaryContextDir = ctx.String(traceDirectoryFlag.Name) + "/"
	if ctx.Bool(traceDebugFlag.Name) {
		traceDebug = true
	}
	first, last, argErr := replay.SetBlockRange(ctx.Args().Get(0), ctx.Args().Get(1))
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

	curEpoch := first / epochLength
	sendOperation(dCtx, opChannel, operation.NewBeginEpoch())

	for iter.Next() {
		tx := iter.Value()
		// close off old block with an end-block operation
		if oldBlock != tx.Block {
			if tx.Block > last {
				break
			}
			if oldBlock != math.MaxUint64 {
				sendOperation(dCtx, opChannel, operation.NewEndBlock(oldBlock))

				// Record epoch changes.
				newEpoch := tx.Block / epochLength
				for curEpoch < newEpoch {
					sendOperation(dCtx, opChannel, operation.NewEndEpoch(curEpoch))
					curEpoch++
					sendOperation(dCtx, opChannel, operation.NewBeginEpoch())
				}
			}
			oldBlock = tx.Block
			// open new block with a begin-block operation and clear index cache
			sendOperation(dCtx, opChannel, operation.NewBeginBlock(tx.Block))
		}
		sendOperation(dCtx, opChannel, operation.NewBeginTransaction())
		traceRecordTask(tx.Block, tx.Transaction, tx.Substate, dCtx, opChannel)
		sendOperation(dCtx, opChannel, operation.NewEndTransaction(uint32(tx.Transaction)))
		if enableProgress {
			// report progress
			sec = time.Since(start).Seconds()
			if sec-lastSec >= 15 {
				fmt.Printf("trace record: Elapsed time: %.0f s, at block %v\n", sec, oldBlock)
				lastSec = sec
			}
		}

	}

	// end last block
	if oldBlock != math.MaxUint64 {
		sendOperation(dCtx, opChannel, operation.NewEndBlock(oldBlock))
	}
	sendOperation(dCtx, opChannel, operation.NewEndEpoch(curEpoch))

	if enableProgress {
		sec = time.Since(start).Seconds()
		fmt.Printf("trace record: Total elapsed time: %.3f s, processed %v blocks\n", sec, last-first+1)
	}

	// close channel
	close(opChannel)

	// write dictionaries and indexes
	dCtx.Write()

	return err
}
