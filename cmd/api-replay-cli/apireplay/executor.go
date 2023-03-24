package apireplay

import (
	"encoding/json"
	"sync"

	"github.com/Fantom-foundation/Aida/iterator"
	"github.com/Fantom-foundation/Aida/state"
	"github.com/Fantom-foundation/Aida/utils"
	"github.com/ethereum/go-ethereum/params"
)

// executorInput represents data needed for executing request into StateDB
type executorInput struct {
	archive state.StateDB
	req     *iterator.Body
	result  json.RawMessage
	error   *iterator.ErrorMessage
	blockID uint64
}

// OutData are sent to comparator with result from StateDB and req/resp Recorded on API server
type OutData struct {
	Method     string
	MethodBase string
	Recorded   *RecordedData
	StateDB    *StateDBData
	BlockID    uint64
	Params     []interface{}
}

// ReplayExecutor represents a goroutine in which requests are executed into StateDB
type ReplayExecutor struct {
	cfg            *utils.Config
	input          chan *executorInput
	output         chan *OutData
	wg             *sync.WaitGroup
	closed         chan any
	currentBlockID uint64
	vmImpl         string
	chainCfg       *params.ChainConfig
}

// newExecutor returns new instance of ReplayExecutor
func newExecutor(output chan *OutData, chainCfg *params.ChainConfig, input chan *executorInput, vmImpl string, wg *sync.WaitGroup, closed chan any) *ReplayExecutor {
	return &ReplayExecutor{
		chainCfg: chainCfg,
		vmImpl:   vmImpl,
		closed:   closed,
		input:    input,
		output:   output,
		wg:       wg,
	}
}

// Start the ReplayExecutor
func (e *ReplayExecutor) Start() {
	e.wg.Add(1)
	go e.execute()
}

// execute reads request from Reader and executes it into given archive
func (e *ReplayExecutor) execute() {
	var (
		ok  bool
		in  *executorInput
		res *StateDBData
	)

	defer func() {
		e.wg.Done()
	}()

	for {
		select {
		case <-e.closed:
			return
		default:

		}
		in, ok = <-e.input

		// if input is closed, stop the Executor
		if !ok {
			return
		}

		// doExecute into db
		res = e.doExecute(in)

		// send to compare
		e.output <- createOutData(in, res)

	}
}

// createOutData and send it to Comparator
func createOutData(in *executorInput, r *StateDBData) *OutData {

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
func (e *ReplayExecutor) doExecute(in *executorInput) *StateDBData {
	switch in.req.MethodBase {
	// ftm/eth methods
	case "getBalance":
		return executeGetBalance(in.req.Params[0], in.archive)

	case "getTransactionCount":
		return executeGetTransactionCount(in.req.Params[0], in.archive)

	case "call":
		req := newEVMRequest(in.req.Params[0].(map[string]interface{}))
		evm := newEVM(in.blockID, in.archive, e.vmImpl, e.chainCfg, req)
		return executeCall(evm)

	case "estimateGas":
		req := newEVMRequest(in.req.Params[0].(map[string]interface{}))
		evm := newEVM(in.blockID, in.archive, e.vmImpl, e.chainCfg, req)
		return executeEstimateGas(evm)

	default:
		break
	}
	return nil
}
