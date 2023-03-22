package apireplay

import (
	"context"
	"encoding/json"
	"github.com/Fantom-foundation/Aida/iterator"
	"github.com/Fantom-foundation/Aida/state"
	"github.com/Fantom-foundation/Aida/utils"
	"github.com/Fantom-foundation/go-opera/evmcore"
	"github.com/ethereum/go-ethereum/common"
	"math/big"
	"sync"
)

// workerInput represents data needed for executing request into StateDB
type workerInput struct {
	archive state.StateDB
	req     *iterator.Body
	result  json.RawMessage
	error   *iterator.ErrorMessage
	blockID uint64
}

// OutData are sent to comparator with data from StateDB and data Recorded on API server
type OutData struct {
	Method     string
	MethodBase string
	Recorded   *RecordedData
	StateDB    *StateDBData
	BlockID    uint64
	Params     []interface{}
}

// ExecutorWorker represents a goroutine in which requests are executed into StateDB
type ExecutorWorker struct {
	cfg            *utils.Config
	ctx            context.Context
	input          chan *workerInput
	output         chan *OutData
	wg             *sync.WaitGroup
	closed         chan any
	currentBlockID uint64
}

// newWorker returns new instance of ExecutorWorker
func newWorker(input chan *workerInput, output chan *OutData, wg *sync.WaitGroup, closed chan any, cfg *utils.Config) *ExecutorWorker {
	return &ExecutorWorker{
		cfg:    cfg,
		closed: closed,
		input:  input,
		output: output,
		wg:     wg,
	}
}

// Start the worker
func (w *ExecutorWorker) Start() {
	w.wg.Add(1)
	go w.execute()
}

// Stop the worker
func (w *ExecutorWorker) Stop() {
	w.wg.Done()
}

// execute is workers thread where it receives requests
func (w *ExecutorWorker) execute() {
	var (
		in  *workerInput
		res *StateDBData
	)

	for {
		select {
		case <-w.closed:
			w.Stop()
			return

		case in = <-w.input:

			// doExecute into db
			res = w.doExecute(in)

			// send to compare
			w.output <- createOutData(in, res)
		}

	}
}

// createOutData as a vessel for data sent to Comparator
func createOutData(in *workerInput, r *StateDBData) *OutData {

	out := new(OutData)
	out.Recorded = new(RecordedData)

	// StateDB result
	out.StateDB = r

	// set the method
	out.Method = in.req.Method
	out.MethodBase = in.req.MethodBase

	// add recorded result to output data
	out.Recorded.Result = in.result

	// add recorded error to output data
	out.Recorded.Error = in.error

	// add params
	out.Params = in.req.Params

	return out
}

// doExecute calls correct executive func for given MethodBase
func (w *ExecutorWorker) doExecute(in *workerInput) *StateDBData {
	switch in.req.MethodBase {
	// ftm/eth methods
	case "getBalance":
		return executeGetBalance(in.req.Params[0], in.archive)

	case "getTransactionCount":
		return executeGetTransactionCount(in.req.Params[0], in.archive)

	case "call":
		req := newEVMRequest(in.req.Params[0].(map[string]interface{}))
		evm := newEVM(in.blockID, in.archive, w.cfg, utils.GetChainConfig(w.cfg.ChainID), req)
		return executeCall(evm)

	case "estimateGas":
		req := newEVMRequest(in.req.Params[0].(map[string]interface{}))
		evm := newEVM(in.blockID, in.archive, w.cfg, utils.GetChainConfig(w.cfg.ChainID), req)
		return executeEstimateGas(evm)

	default:
		break
	}

	// todo log
	//w.log.Debugf("skipping unsupported method %v", body.Method)
	return nil
}

// executeGetBalance request into archive and return the result
func executeGetBalance(param interface{}, archive state.StateDB) (out *StateDBData) {
	var (
		address common.Address
	)

	out = new(StateDBData)
	out.Result = new(big.Int)

	// decode requested address
	address = common.HexToAddress(param.(string))

	// retrieve compareBalance
	out.Result = archive.GetBalance(address)

	return
}

// executeGetTransactionCount request into archive and return the result
func executeGetTransactionCount(param interface{}, archive state.StateDB) (out *StateDBData) {
	var (
		address common.Address
	)

	out = new(StateDBData)

	// decode requested address
	address = common.HexToAddress(param.(string))

	// retrieve nonce
	out.Result = archive.GetNonce(address)

	return
}

// executeCall into EVM and return the result
func executeCall(evm *EVM) (out *StateDBData) {
	var (
		result *evmcore.ExecutionResult
		err    error
	)

	out = new(StateDBData)

	// get the result from EVM
	result, err = evm.sendCall()
	if err != nil {
		out.Error = err
		return
	}

	// If the result contains a revert reason, try to unpack and return it.
	if len(result.Revert()) > 0 {
		out.Error = newRevertError(result)
	} else {
		out.Result = result.Return()
		out.Error = result.Err
	}

	return
}

// executeEstimateGas into EVM which calculates gas needed for a transaction
func executeEstimateGas(evm *EVM) (out *StateDBData) {
	out = new(StateDBData)
	out.Result, out.Error = evm.sendEstimateGas()

	return
}
