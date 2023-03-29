package apireplay

import (
	"encoding/json"
	"errors"
	"log"
	"sync"

	"github.com/Fantom-foundation/Aida/iterator"
	"github.com/Fantom-foundation/Aida/state"
	"github.com/Fantom-foundation/Aida/utils"
	substate "github.com/Fantom-foundation/Substate"
	"github.com/ethereum/go-ethereum/params"
)

// executorInput represents data needed for executing request into StateDB
type executorInput struct {
	archive state.StateDB
	req     *iterator.RequestWithResponse
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
	ParamsRaw  []byte
}

// ReplayExecutor represents a goroutine in which requests are executed into StateDB
type ReplayExecutor struct {
	iterBlock      uint64
	iter           substate.SubstateIterator
	cfg            *utils.Config
	input          chan *executorInput
	output         chan *OutData
	wg             *sync.WaitGroup
	closed         chan any
	currentBlockID uint64
	vmImpl         string
	chainCfg       *params.ChainConfig
	verbose        bool
}

// newExecutor returns new instance of ReplayExecutor
func newExecutor(output chan *OutData, chainCfg *params.ChainConfig, input chan *executorInput, vmImpl string, wg *sync.WaitGroup, closed chan any, verbose bool) *ReplayExecutor {
	return &ReplayExecutor{
		chainCfg: chainCfg,
		vmImpl:   vmImpl,
		closed:   closed,
		input:    input,
		output:   output,
		wg:       wg,
		verbose:  verbose,
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

		// are we at debugging state?
		if e.verbose {
			e.logReq(in)
		}

		// doExecute into db
		res = e.doExecute(in)

		// send to compare
		e.output <- createOutData(in, res)

	}
}

// getEVMTimestamp retrieves timestamp for current block from substate
func (e *ReplayExecutor) getEVMTimestamp(wantedBlockID uint64) (uint64, error) {
	// same block?
	if wantedBlockID != e.iterBlock {
		e.iter = substate.NewSubstateIterator(wantedBlockID, 1)
	}

	if e.iter.Next() {
		if e.iter.Value() != nil {
			return e.iter.Value().Substate.Env.Timestamp, nil
		}
	}

	return 0, errors.New("substate for this block does not exist")

}

// createOutData and send it to Comparator
func createOutData(in *executorInput, r *StateDBData) *OutData {

	out := new(OutData)
	out.Recorded = new(RecordedData)

	// StateDB result
	out.StateDB = r

	// set blockID
	out.BlockID = in.blockID

	// set the method
	out.Method = in.req.Query.Method
	out.MethodBase = in.req.Query.MethodBase

	// add recorded result to output data
	out.Recorded.Result = in.result

	// add recorded error to output data
	out.Recorded.Error = in.error

	// add params
	out.Params = in.req.Query.Params

	// add raw params for clear logging
	out.ParamsRaw = in.req.ParamsRaw

	return out
}

// doExecute calls correct executive func for given MethodBase
func (e *ReplayExecutor) doExecute(in *executorInput) *StateDBData {
	switch in.req.Query.MethodBase {
	// ftm/eth methods
	case "getBalance":
		return executeGetBalance(in.req.Query.Params[0], in.archive)

	case "getTransactionCount":
		return executeGetTransactionCount(in.req.Query.Params[0], in.archive)

	case "call":
		req := newEVMRequest(in.req.Query.Params[0].(map[string]interface{}))
		timestamp, err := e.getEVMTimestamp(in.blockID)
		if err != nil {
			if e.verbose {
				log.Printf("substate for block #%v does not seem to exist; skipping request", in.blockID)
			}
			return nil
		}
		evm := newEVM(in.blockID, in.archive, e.vmImpl, e.chainCfg, req, timestamp)
		return executeCall(evm)

	case "estimateGas":
		req := newEVMRequest(in.req.Query.Params[0].(map[string]interface{}))
		timestamp, err := e.getEVMTimestamp(in.blockID)
		if err != nil {
			log.Printf("substate for block #%v does not seem to exist; skipping request", in.blockID)
			return nil
		}
		evm := newEVM(in.blockID, in.archive, e.vmImpl, e.chainCfg, req, timestamp)
		return executeEstimateGas(evm)

	default:
		break
	}
	return nil
}

func (e *ReplayExecutor) logReq(in *executorInput) {
	log.Printf("executing %v with these params: \n\t%v", in.req.Query.Method, string(in.req.ParamsRaw))
}
