package apireplay

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/Fantom-foundation/Aida/iterator"
	"github.com/Fantom-foundation/Aida/state"
	"github.com/Fantom-foundation/Aida/utils"
	"github.com/ethereum/go-ethereum/params"
	"github.com/op/go-logging"
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
	logging        chan logMsg
	wg             *sync.WaitGroup
	closed         chan any
	currentBlockID uint64
	vmImpl         string
	chainCfg       *params.ChainConfig
}

// newWorker returns new instance of ExecutorWorker
func newWorker(chainCfg *params.ChainConfig, input chan *workerInput, output chan *OutData, wg *sync.WaitGroup, closed chan any, vmImpl string, logging chan logMsg) *ExecutorWorker {
	return &ExecutorWorker{
		chainCfg: chainCfg,
		vmImpl:   vmImpl,
		closed:   closed,
		input:    input,
		logging:  logging,
		output:   output,
		wg:       wg,
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
		default:

		}
		in = <-w.input

		// doExecute into db
		res = w.doExecute(in)

		// send to compare
		w.output <- createOutData(in, res)

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
		evm := newEVM(in.blockID, in.archive, w.vmImpl, w.chainCfg, req)
		return executeCall(evm)

	case "estimateGas":
		req := newEVMRequest(in.req.Params[0].(map[string]interface{}))
		evm := newEVM(in.blockID, in.archive, w.vmImpl, w.chainCfg, req)
		return executeEstimateGas(evm)

	default:
		break
	}

	w.logging <- logMsg{
		lvl: logging.DEBUG,
		msg: fmt.Sprintf("skipping unsupported method %v", in.req.Method),
	}
	return nil
}
