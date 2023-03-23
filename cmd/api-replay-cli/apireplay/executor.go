package apireplay

import (
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
	"sync"

	"github.com/Fantom-foundation/Aida/iterator"
	"github.com/Fantom-foundation/Aida/state"
	"github.com/Fantom-foundation/Aida/utils"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/op/go-logging"
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
	logging     chan logMsg
	closed      chan any
	log         *logging.Logger
	appWg       *sync.WaitGroup
	workersWg   *sync.WaitGroup
}

// newReplayExecutor returns new instance of ReplayExecutor
func newReplayExecutor(db state.StateDB, reader *iterator.FileReader, cfg *utils.Config, l *logging.Logger, wg *sync.WaitGroup) *ReplayExecutor {
	return &ReplayExecutor{
		cfg:         cfg,
		db:          db,
		reader:      reader,
		log:         l,
		logging:     make(chan logMsg),
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

	// start executors loops
	go e.doLog()
	go e.readRequests()

	// start its workers
	e.initWorkers(workers)
}

// initWorkers creates and starts given number of ExecutorWorkers
func (e *ReplayExecutor) initWorkers(workers int) {
	// send info about workers
	e.logging <- logMsg{
		lvl: logging.INFO,
		msg: fmt.Sprintf("starting %v ExecutorWorkers", workers),
	}

	e.workers = make([]*ExecutorWorker, workers)
	for i := 0; i < workers; i++ {
		e.workers[i] = newWorker(e.workerInput, e.output, e.workersWg, e.closed, e.cfg, e.logging)
		e.workers[i].Start()
	}
}

// Stop the ReplayExecutor
func (e *ReplayExecutor) Stop() {
	select {
	case <-e.closed:
		return
	default:
		close(e.closed)
		e.workersWg.Wait()
	}
}

// readRequests retrieves req from reader (if not at the end) and pass the data alongside wanted archive
// to ExecutorWorker which executes the request into StateDB
func (e *ReplayExecutor) readRequests() {
	var (
		req    *iterator.RequestWithResponse
		wInput *workerInput
	)
	defer func() {
		e.reader.Close()
		close(e.output)
		close(e.workerInput)
		e.appWg.Done()
	}()

	for e.reader.Next() {
		select {
		case <-e.closed:
			return
		default:
		}

		// did reader emit an error?
		if e.reader.Error() != nil {
			e.logging <- logMsg{
				lvl: logging.CRITICAL,
				msg: fmt.Sprintf("error loading recordings; %v", e.reader.Error().Error()),
			}
			e.Stop()
		}

		// retrieve the data from iterator
		req = e.reader.Value()

		wInput = e.createWorkerInput(req)
		if wInput != nil {
			e.workerInput <- wInput
		}

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
		e.logging <- logMsg{
			lvl: logging.ERROR,
			msg: "both recorded response and recorded error are nil; skipping\"",
		}
		return nil
	}

	// request
	wInput.req = req.Query

	if !e.decodeBlockNumber(req.Query.Params, recordedBlockID, &wInput.blockID) {
		e.logging <- logMsg{
			lvl: logging.ERROR,
			msg: fmt.Sprintf("cannot decode block number; skipping\nParams: %v", req.Query.Params[1]),
		}
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
		e.logging <- logMsg{
			lvl: logging.DEBUG,
			msg: "request out of StateDB block range\nSKIPPING\n",
		}
		return nil
	}

	// load the archive itself
	var err error
	archive, err := e.db.GetArchiveState(wantedBlockNumber)
	if err != nil {
		e.logging <- logMsg{
			lvl: logging.DEBUG,
			msg: fmt.Sprintf("cannot retrieve archive for block id #%v; skipping; err: %v\n", wantedBlockNumber, err),
		}
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

// doLog is a thread for logging any messages from ReplayExecutor or ExecutorWorker
func (e *ReplayExecutor) doLog() {
	var l logMsg

	for {
		select {
		case <-e.closed:
			return
		default:
		}

		l = <-e.logging
		switch l.lvl {
		case logging.CRITICAL:
			e.log.Critical(l.msg)
		case logging.ERROR:
			e.log.Error(l.msg)
		case logging.WARNING:
			e.log.Warning(l.msg)
		case logging.NOTICE:
			e.log.Notice(l.msg)
		case logging.INFO:
			e.log.Info(l.msg)
		case logging.DEBUG:
			e.log.Debug(l.msg)
		default:
		}
	}
}
