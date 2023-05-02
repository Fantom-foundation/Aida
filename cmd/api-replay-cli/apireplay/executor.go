package apireplay

import (
	"encoding/json"
	"math/big"
	"strings"
	"sync"

	"github.com/Fantom-foundation/Aida/iterator"
	"github.com/Fantom-foundation/Aida/state"
	"github.com/Fantom-foundation/Aida/utils"
	substate "github.com/Fantom-foundation/Substate"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/op/go-logging"
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

// dbRange represents first and last block of StateDB
type dbRange struct {
	first, last uint64
}

// ReplayExecutor represents a goroutine in which requests are executed into StateDB
type ReplayExecutor struct {
	dbRange      dbRange
	iterBlock    uint64
	cfg          *utils.Config
	input        chan *iterator.RequestWithResponse
	output       chan *OutData
	wg           *sync.WaitGroup
	closed       chan any
	vmImpl       string
	chainCfg     *params.ChainConfig
	log          *logging.Logger
	db           state.StateDB
	counterInput chan requestLog
	timestamps   map[uint64]uint64
}

// newExecutor returns new instance of ReplayExecutor
func newExecutor(first, last uint64, db state.StateDB, output chan *OutData, chainCfg *params.ChainConfig, input chan *iterator.RequestWithResponse, vmImpl string, wg *sync.WaitGroup, closed chan any, log *logging.Logger, counterInput chan requestLog) *ReplayExecutor {
	return &ReplayExecutor{
		dbRange: dbRange{
			first: first,
			last:  last,
		},
		db:           db,
		chainCfg:     chainCfg,
		vmImpl:       vmImpl,
		closed:       closed,
		input:        input,
		output:       output,
		wg:           wg,
		log:          log,
		counterInput: counterInput,
		timestamps:   make(map[uint64]uint64),
	}
}

// Start the ReplayExecutor
func (e *ReplayExecutor) Start() {
	go e.execute()
}

// execute reads request from Reader and executes it into given archive
func (e *ReplayExecutor) execute() {
	var (
		ok      bool
		req     *iterator.RequestWithResponse
		in      *executorInput
		res     *StateDBData
		logType reqLogType
	)

	defer func() {
		e.wg.Done()
	}()

	for {
		select {
		case <-e.closed:
			return
		case req, ok = <-e.input:
			// if input is closed, stop the Executor
			if !ok {
				return
			}

			in = e.createInput(req)

			// are we in block range?
			if in == nil {
				// send statistics
				e.counterInput <- requestLog{
					method:  req.Query.Method,
					logType: outOfStateDBRange,
				}

				// no need to executed rest of the loop
				continue
			}

			// doExecute into db
			res = e.doExecute(in)

			// was execution successful?
			if res != nil {
				logType = executed

				select {
				case <-e.closed:
					return
				case e.output <- createOutData(in, res):
				}
			} else {
				logType = noSubstateForGivenBlock
			}
		}

		select {
		case <-e.closed:
			return
		// send statistics
		case e.counterInput <- requestLog{
			method:  req.Query.Method,
			logType: logType,
		}:
		}

	}
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
		timestamp := e.getTimestamp(in.blockID)
		if timestamp == 0 {
			return nil
		}
		evm := newEVMExecutor(in.blockID, in.archive, e.vmImpl, e.chainCfg, in.req.Query.Params[0].(map[string]interface{}), timestamp, e.log)
		return executeCall(evm)

	case "estimateGas":
		// estimateGas is currently not suitable for replay since the estimation  in geth is always calculated for current state
		// that means recorded result and result returned by StateDB are not comparable
	case "getCode":
		return executeGetCode(in.req.Query.Params[0], in.archive)

	case "getStorageAt":
		return executeGetStorageAt(in.req.Query.Params, in.archive)

	default:
		break
	}
	return nil
}

// createInput with data worker need to doExecute request into archive
func (e *ReplayExecutor) createInput(req *iterator.RequestWithResponse) *executorInput {
	var recordedBlockID uint64
	var wInput = new(executorInput)

	// response
	if req.Error != nil {
		recordedBlockID = req.Error.BlockID
		wInput.error = &req.Error.Error
	} else if req.Response != nil {
		recordedBlockID = req.Response.BlockID
		wInput.result = req.Response.Result
	} else {
		e.log.Error("both recorded response and recorded error are nil; skipping")
		return nil
	}

	// request
	wInput.req = req

	if !decodeBlockNumber(req.Query.Params, recordedBlockID, &wInput.blockID) {
		return nil
	}

	// archive
	wInput.archive = e.getStateArchive(wInput.blockID)
	if wInput.archive == nil {
		return nil
	}

	return wInput
}

// getStateArchive for given block
func (e *ReplayExecutor) getStateArchive(wantedBlockNumber uint64) state.StateDB {
	if !e.isBlockNumberWithinRange(wantedBlockNumber) {
		return nil
	}

	// load the archive itself
	var err error
	archive, err := e.db.GetArchiveState(wantedBlockNumber)
	if err != nil {
		e.log.Errorf("cannot retrieve archive for block id #%v; err: %v", wantedBlockNumber, err)
		return nil
	}

	return archive
}

// isBlockNumberWithinRange returns whether given block number is in StateDB block range
func (e *ReplayExecutor) isBlockNumberWithinRange(blockNumber uint64) bool {
	return blockNumber >= e.dbRange.first && blockNumber <= e.dbRange.last
}

// getTimestamp looks whether current block is the same as wanted. If not, retrieves new timestamp from substate
func (e *ReplayExecutor) getTimestamp(blockID uint64) uint64 {
	var (
		ok        bool
		timestamp uint64
	)
	if timestamp, ok = e.timestamps[blockID]; ok {
		return timestamp
	}

	if substate.HasSubstate(blockID, 0) {
		timestamp = substate.GetSubstate(blockID, 0).Env.Timestamp
	}
	e.timestamps[blockID] = timestamp

	return timestamp
}

// decodeBlockNumber finds what block number request wants
func decodeBlockNumber(params []interface{}, recordedBlockNumber uint64, returnedBlockID *uint64) bool {

	// request does not demand specific currentBlockID, so we take the recorded one
	if len(params) < 2 {
		*returnedBlockID = recordedBlockNumber
		return true
	}

	// request does not have blockID specification
	str, ok := params[1].(string)
	if !ok {
		*returnedBlockID = recordedBlockNumber
		return true
	}

	switch str {
	case "latest":
		// request required latest currentBlockID so we return the recorded one
		*returnedBlockID = recordedBlockNumber
		break
	case "earliest":
		*returnedBlockID = uint64(rpc.EarliestBlockNumber)
		break
	case "pending":
		*returnedBlockID = recordedBlockNumber
	default:
		// request requires specific currentBlockID
		var (
			bigID *big.Int
			ok    bool
		)

		bigID = new(big.Int)
		str = strings.TrimPrefix(str, "0x")
		_, ok = bigID.SetString(str, 16)

		if !ok {
			return false
		}
		*returnedBlockID = bigID.Uint64()
		break
	}

	return true
}
