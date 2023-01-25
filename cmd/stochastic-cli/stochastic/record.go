package stochastic

import (
	"bufio"
	"context"
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
	"github.com/Fantom-foundation/Aida/utils"
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

// StochasticRecordCommand data structure for the record app
var StochasticRecordCommand = cli.Command{
	Action:    stochasticRecordAction,
	Name:      "stochastic-record",
	Usage:     "captures and records StateDB operations while processing blocks",
	ArgsUsage: "<blockNumFirst> <blockNumLast>",
	Flags: []cli.Flag{
		&utils.CpuProfileFlag,
		&utils.DisableProgressFlag,
		&substate.WorkersFlag,
		&substate.SubstateDirFlag,
		&utils.ChainIDFlag,
		&utils.TraceDirectoryFlag,
		&utils.TraceDebugFlag,
		&utils.StochasticMatrixFlag,
		&utils.StochasticMatrixFormatFlag,
	},
	Description: `
The trace record command requires two arguments:
<blockNumFirst> <blockNumLast>

<blockNumFirst> and <blockNumLast> are the first and
last block of the inclusive range of blocks to trace transactions.`,
}

// stochasticRecordTask generates storage traces for a transaction.
func stochasticRecordTask(block uint64, tx, chainID int, recording *substate.Substate, dCtx *dict.DictionaryContext) error {

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

	var statedb state.StateDB
	statedb = state.MakeInMemoryStateDB(&inputAlloc)
	proxyStochastic := tracer.NewProxyStochastic(statedb, dCtx, utils.TraceDebug)
	statedb = proxyStochastic

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

// OperationStochasticWriter reads operations from the operation channel and writes
// them into a trace file.
func OperationStochasticWriter(ctx context.Context, done chan struct{}, ch chan operation.Operation) {
	// send done signal when closing writer
	defer close(done)

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

	// read operations from channel, and write them until receiving a cancel signal
	for {
		select {
		case op := <-ch:
			operation.Write(zfile, op)
		case <-ctx.Done():
			if len(ch) == 0 {
				return
			}
		}
	}
}

// sendStochasticOperation sends an operation onto the channel.
func sendStochasticOperation(dCtx *dict.DictionaryContext, ch chan operation.Operation, op operation.Operation) {
	ch <- op
	dCtx.RecordOp(op.GetId())
	if utils.TraceDebug {
		operation.Debug(dCtx, op)
	}
}

// stochasticRecordAction implements trace command for recording.
func stochasticRecordAction(ctx *cli.Context) error {
	substate.RecordReplay = true
	var err error

	if ctx.Args().Len() != 2 {
		return fmt.Errorf("trace record command requires exactly 2 arguments")
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

	// create dictionary and index contexts
	dCtx := dict.NewDictionaryStochasticContext(operation.BeginBlockID, operation.NumOperations)

	// spawn writer
	opChannel := make(chan operation.Operation, 100000)
	cctx, cancel := context.WithCancel(context.Background())
	cancelChannel := make(chan struct{})
	go OperationStochasticWriter(cctx, cancelChannel, opChannel)

	// process arguments
	chainID := ctx.Int(utils.ChainIDFlag.Name)
	tracer.TraceDir = ctx.String(utils.TraceDirectoryFlag.Name) + "/"
	dict.DictionaryContextDir = ctx.String(utils.TraceDirectoryFlag.Name) + "/"
	if ctx.Bool(utils.TraceDebugFlag.Name) {
		utils.TraceDebug = true
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
	for iter.Next() {
		tx := iter.Value()
		// close off old block with an end-block operation
		if oldBlock != tx.Block {
			if tx.Block > last {
				break
			}
			if oldBlock != math.MaxUint64 {
				sendStochasticOperation(dCtx, opChannel, operation.NewEndBlock())
			}
			oldBlock = tx.Block
			// open new block with a begin-block operation and clear index cache
			sendStochasticOperation(dCtx, opChannel, operation.NewBeginBlock(tx.Block))
		}
		stochasticRecordTask(tx.Block, tx.Transaction, chainID, tx.Substate, dCtx)
		sendStochasticOperation(dCtx, opChannel, operation.NewEndTransaction())
		if enableProgress {
			// report progress
			sec = time.Since(start).Seconds()
			if sec-lastSec >= 15 {
				fmt.Printf("trace record: Elapsed time: %.0f s, at block %v\n", sec, oldBlock)
				lastSec = sec
			}
		}

	}
	if enableProgress {
		sec = time.Since(start).Seconds()
		fmt.Printf("trace record: Total elapsed time: %.3f s, processed %v blocks\n", sec, last-first+1)
	}

	// insert the last EndBlock
	sendStochasticOperation(dCtx, opChannel, operation.NewEndBlock())

	// cancel writer
	(cancel)()        // stop writer
	<-(cancelChannel) // wait for writer to finish

	// write dictionaries and indexes
	dCtx.Write()

	// write recorded frequencies
	dCtx.FrequenciesWriter()

	dCtx.WriteDistributions()

	// if only one block was recorded - add EndBlock to BeginBlock transition
	if last-first == 0 {
		dCtx.TFreq[[2]byte{operation.EndBlockID, operation.BeginBlockID}]++
	}

	// write stochastic matrix
	writeStochasticMatrix(utils.StochasticMatrixFlag.Value, utils.StochasticMatrixFormatFlag.Value, dCtx.TFreq)

	return err
}

func writeStochasticMatrix(smFile string, f string, tFreq map[[2]byte]uint64) {
	fmt.Println("freq", tFreq)
	// write stochastic-matrix
	if f == "csv" {
		writeStochasticMatrixCsv(smFile, tFreq)
	} else {
		writeStochasticMatrixDot(smFile, tFreq)
	}
}

// writeStochasticMatrixCsv writes frequencies of operation chaining in csv format
func writeStochasticMatrixCsv(smFile string, tFreq map[[2]byte]uint64) {
	file, err := os.Create(smFile)
	if err != nil {
		log.Fatalf("Cannot open stochastic matrix file. Error: %v", err)
	}
	defer func() {
		err := file.Close()
		if err != nil {
			log.Fatalf("Cannot close stochastic matrix file. Error: %v", err)
		}
	}()

	for i := byte(0); i < operation.NumOperations; i++ {
		total := uint64(0)
		for j := byte(0); j < operation.NumOperations; j++ {
			total += tFreq[[2]byte{i, j}]
		}

		for j := byte(0); j < operation.NumOperations; j++ {
			//fmt.Printf("\t%v -> %v [%v] \n",
			//	operation.GetLabel(i),
			//	operation.GetLabel(j),
			//	float64(tFreq[[2]byte{i, j}])/float64(total))

			var n float64
			if total == 0 {
				n = 0
			} else {
				n = float64(tFreq[[2]byte{i, j}]) / float64(total)
			}

			fmt.Fprintf(file, "%v", n)

			if j != operation.NumOperations-1 {
				fmt.Fprint(file, ",")
			}
		}
		fmt.Fprintf(file, "\n")
	}
}

// writeStochasticMatrixDot writes frequencies of operation chaining in dot format
func writeStochasticMatrixDot(smFile string, tFreq map[[2]byte]uint64) {
	file, err := os.Create(smFile)
	if err != nil {
		log.Fatalf("Cannot open stochastic matrix file. Error: %v", err)
	}
	defer func() {
		err := file.Close()
		if err != nil {
			log.Fatalf("Cannot close stochastic matrix file. Error: %v", err)
		}
	}()
	fmt.Fprintf(file, "digraph C {\n")
	for i := byte(0); i < operation.NumOperations; i++ {
		total := uint64(0)
		for j := byte(0); j < operation.NumOperations; j++ {
			total += tFreq[[2]byte{i, j}]
		}
		maxFreq := uint64(0)
		for j := byte(0); j < operation.NumOperations; j++ {
			if tFreq[[2]byte{i, j}] > maxFreq {
				maxFreq = tFreq[[2]byte{i, j}]
			}
		}
		for j := byte(0); j < operation.NumOperations; j++ {
			if tFreq[[2]byte{i, j}] != 0 {
				if tFreq[[2]byte{i, j}] != maxFreq {
					fmt.Fprintf(file, "\t%v -> %v [label=\"%v\"]\n",
						operation.GetLabel(i),
						operation.GetLabel(j),
						float64(tFreq[[2]byte{i, j}])/float64(total))
				} else {
					fmt.Fprintf(file, "\t%v -> %v [label=\"%v\", color=red]\n",
						operation.GetLabel(i),
						operation.GetLabel(j),
						float64(tFreq[[2]byte{i, j}])/float64(total))
				}
			}
		}
	}
	fmt.Fprintf(file, "}\n")
}
