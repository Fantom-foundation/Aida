package apireplay

import (
	"encoding/json"
	"math/big"
	"strings"
	"sync"

	"github.com/Fantom-foundation/Aida/iterator"
	"github.com/Fantom-foundation/Aida/state"
	"github.com/Fantom-foundation/Aida/utils"
	"github.com/Fantom-foundation/go-opera/evmcore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/op/go-logging"
)

// OutData are sent to comparator with data from StateDB and data Recorded on API server
type OutData struct {
	Method     string
	MethodBase string
	Recorded   *RecordedData
	StateDB    *StateDBData
	BlockID    uint64
	Params     []interface{}
}

// RecordedData represents data recorded on API server. This is sent to Comparator and compared with StateDBData
type RecordedData struct {
	Result json.RawMessage
	Error  *iterator.ErrorMessage
}

// StateDBData represents data that StateDB returned for requests recorded on API server
// This is sent to Comparator and compared with RecordedData
type StateDBData struct {
	Result any
	Error  error
}

// ReplayExecutor executes recorded requests into StateDB. Returned data is stored as StateDBData
// Recorded data is stored as RecordedData and is sent to Comparator along with StateDB for comparison
type ReplayExecutor struct {
	cfg            *utils.Config
	db             state.StateDB
	archive        state.StateDB
	currentBlockID uint64
	reader         *iterator.FileReader
	output         chan *OutData
	closed         chan any
	log            *logging.Logger
	wg             *sync.WaitGroup
}

// move cli away

// newReplayExecutor returns new instance of ReplayExecutor
func newReplayExecutor(db state.StateDB, reader *iterator.FileReader, cfg *utils.Config, log *logging.Logger, wg *sync.WaitGroup) *ReplayExecutor {
	return &ReplayExecutor{
		cfg:    cfg,
		db:     db,
		reader: reader,
		log:    log,
		output: make(chan *OutData),
		closed: make(chan any),
		wg:     wg,
	}
}

// Start the ReplayExecutor
func (e *ReplayExecutor) Start() {
	e.wg.Add(1)
	go e.executeRequests()
}

// Stop the ReplayExecutor
func (e *ReplayExecutor) Stop() {
	select {
	case <-e.closed:
		return
	default:
		close(e.closed)
	}
}

// executeRequests retrieves req from reader (if not at the end) and executes it into archive,
// then sends the result along with recorded response to Comparator
func (e *ReplayExecutor) executeRequests() {
	var (
		blockID uint64
	)

	defer func() {
		e.reader.Close()
		close(e.output)
		e.wg.Done()
	}()

	for e.reader.Next() {
		// retrieve the data from iterator
		req := e.reader.Value()

		// set block id
		if req.Error != nil {
			blockID = req.Error.BlockID
		} else {
			blockID = req.Response.BlockID
		}

		// retrieve the archive from StateDB for given block id
		if !e.getStateArchive(req.Query, blockID) {
			// if error occurs skip current req
			continue
		}

		// get result from archive
		r := e.execute(req.Query, blockID)

		// if method was recognized
		if r != nil {
			// execute the data to Comparator
			e.output <- createOutData(req, r)
		}

		select {
		case <-e.closed:
			break
		default:
			continue
		}
	}

	if e.reader.Error() != nil {
		e.log.Fatalf("error loading recordings; %e", e.reader.Error().Error())
	}

}

// createOutData as a vessel for data sent to Comparator
func createOutData(req *iterator.RequestWithResponse, r *StateDBData) *OutData {
	var out *OutData

	out = new(OutData)
	out.StateDB = r

	out.Recorded = new(RecordedData)

	// set the method
	out.Method = req.Query.Method
	out.MethodBase = req.Query.MethodBase

	// add recorded result to output data
	if req.Response != nil {
		out.Recorded.Result = req.Response.Result
	}

	// add recorded error to output data
	if req.Error != nil {
		out.Recorded.Error = &req.Error.Error
	}

	// add params
	out.Params = req.Query.Params

	return out
}

// getStateArchive for given block if not already loaded
func (e *ReplayExecutor) getStateArchive(query *iterator.Body, blockID uint64) bool {
	// first find wanted block number
	var wantedBlockNumber uint64

	if !e.decodeBlockNumber(query.Params, blockID, &wantedBlockNumber) {
		e.log.Errorf("cannot decode block number\nParams: %v", query.Params[1])
		return false
	}

	if !e.isBlockNumberWithinRange(wantedBlockNumber) {
		e.log.Debugf("request out of StateDB block range\nSKIPPING\n")
		return false
	}

	// if current archive is the same block number there is no need for reloading it
	if e.currentBlockID == wantedBlockNumber {
		return true
	}

	// load the archive itself
	var err error
	e.archive, err = e.db.GetArchiveState(wantedBlockNumber)
	if err != nil {
		e.log.Error("cannot retrieve archive for block id #%v\nerr: %v\n", wantedBlockNumber, err)
		return false
	}

	e.currentBlockID = wantedBlockNumber
	return true
}

// decodeBlockNumber finds what block number request wants
func (e *ReplayExecutor) decodeBlockNumber(params []interface{}, recordedBlockNumber uint64, returnedBlockID *uint64) bool {
	// request does not demand specific currentBlockID, so we take the recorded one
	if len(params) < 2 {
		*returnedBlockID = recordedBlockNumber
		return true
	}

	str, ok := params[1].(string)
	if !ok {
		return false
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

// isBlockNumberWithinRange returns whether given block number is in StateDB block range
func (e *ReplayExecutor) isBlockNumberWithinRange(blockNumber uint64) bool {
	return blockNumber >= e.cfg.First && blockNumber <= e.cfg.Last
}

// execute calls correct execute func according to method in body
func (e *ReplayExecutor) execute(body *iterator.Body, blockID uint64) *StateDBData {
	switch body.MethodBase {
	// ftm/eth methods
	case "getBalance":
		return executeGetBalance(body.Params[0], e.archive)

	case "getTransactionCount":
		return executeGetTransactionCount(body.Params[0], e.archive)

	case "call":

		req := newEVMRequest(body.Params[0].(map[string]interface{}))
		evm := newEVM(blockID, e.archive, e.cfg, utils.GetChainConfig(e.cfg.ChainID), req)
		return executeCall(evm)

	case "estimateGas":
		req := newEVMRequest(body.Params[0].(map[string]interface{}))
		evm := newEVM(blockID, e.archive, e.cfg, utils.GetChainConfig(e.cfg.ChainID), req)
		return executeEstimateGas(evm)
	default:
		break
	}

	e.log.Debugf("skipping unsupported method %v", body.Method)
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
