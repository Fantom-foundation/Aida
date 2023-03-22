package apireplay

import (
	"encoding/json"
	"github.com/Fantom-foundation/Aida/iterator"
	"github.com/Fantom-foundation/Aida/state"
	"github.com/Fantom-foundation/Aida/utils"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/op/go-logging"
	"math/big"
	"strings"
	"sync"
)

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

// ReplayExecutor reads data from iterator, creates logical structure and pass it, alongside wanted archive, to
// ExecutorWorker which executes the request into StateDB
type ReplayExecutor struct {
	cfg         *utils.Config
	db          state.StateDB
	workers     []*ExecutorWorker
	workerInput chan *workerInput
	reader      *iterator.FileReader
	output      chan *OutData
	closed      chan any
	log         *logging.Logger
	appWg       *sync.WaitGroup
	workersWg   *sync.WaitGroup
}

// newReplayExecutor returns new instance of ReplayExecutor
func newReplayExecutor(db state.StateDB, reader *iterator.FileReader, cfg *utils.Config, log *logging.Logger, wg *sync.WaitGroup) *ReplayExecutor {
	return &ReplayExecutor{
		cfg:         cfg,
		db:          db,
		reader:      reader,
		log:         log,
		workerInput: make(chan *workerInput),
		output:      make(chan *OutData),
		closed:      make(chan any),
		appWg:       wg,
		workersWg:   new(sync.WaitGroup),
	}
}

// Start the ReplayExecutor
func (e *ReplayExecutor) Start(workers int) {
	e.appWg.Add(1)
	e.initWorkers(workers)
	go e.executeRequests()
}

// initWorkers creates and starts given number of ExecutorWorkers
func (e *ReplayExecutor) initWorkers(workers int) {
	e.workers = make([]*ExecutorWorker, workers)
	for i := 0; i < workers; i++ {
		e.workers[i] = newWorker(e.workerInput, e.output, e.workersWg, e.closed, e.cfg)
		e.workers[i].Start()
	}
}

// Stop the ReplayExecutor
func (e *ReplayExecutor) Stop() {
	select {
	case <-e.closed:
		e.workersWg.Wait()
		return
	default:
		close(e.closed)
	}
}

// executeRequests retrieves req from reader (if not at the end) and pass the data alongside wanted archive
// to ExecutorWorker which executes the request into StateDB
func (e *ReplayExecutor) executeRequests() {
	var (
		req    *iterator.RequestWithResponse
		wInput *workerInput
	)
	defer func() {
		e.reader.Close()
		close(e.output)
		e.appWg.Done()
	}()

	for e.reader.Next() {
		// retrieve the data from iterator
		req = e.reader.Value()

		wInput = e.createWorkerInput(req)
		if wInput != nil {
			e.workerInput <- wInput
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

// createWorkerInput with data worker need to doExecute request into archive
func (e *ReplayExecutor) createWorkerInput(req *iterator.RequestWithResponse) *workerInput {
	var recordedBlockID uint64
	var wInput = new(workerInput)

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
	wInput.req = req.Query

	if !e.decodeBlockNumber(req.Query.Params, recordedBlockID, &wInput.blockID) {
		e.log.Errorf("cannot decode block number\nParams: %v", req.Query.Params[1])
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
		e.log.Debugf("request out of StateDB block range\nSKIPPING\n")
		return nil
	}

	// load the archive itself
	var err error
	archive, err := e.db.GetArchiveState(wantedBlockNumber)
	if err != nil {
		e.log.Error("cannot retrieve archive for block id #%v\nerr: %v\n", wantedBlockNumber, err)
		return nil
	}

	return archive
}

// decodeBlockNumber finds what block number request wants
func (e *ReplayExecutor) decodeBlockNumber(params []interface{}, recordedBlockNumber uint64, returnedBlockID *uint64) bool {
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

// isBlockNumberWithinRange returns whether given block number is in StateDB block range
func (e *ReplayExecutor) isBlockNumberWithinRange(blockNumber uint64) bool {
	return blockNumber >= e.cfg.First && blockNumber <= e.cfg.Last
}
