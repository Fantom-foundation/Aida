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

// dbRange represents first and last block of StateDB
type dbRange struct {
	first, last uint64
}

// Reader reads data from iterator, creates logical structure and pass it alongside wanted archive, to
// ReplayExecutor which executes the request into StateDB
type Reader struct {
	db            state.StateDB
	executors     []*ReplayExecutor
	executorInput chan *executorInput
	reader        *iterator.FileReader
	output        chan *OutData
	logging       chan logMsg
	closed        chan any
	log           *logging.Logger
	appWg         *sync.WaitGroup
	executorsWg   *sync.WaitGroup
	dbRange       dbRange
}

// newReader returns new instance of Reader
func newReader(first, last uint64, db state.StateDB, reader *iterator.FileReader, l *logging.Logger, wg *sync.WaitGroup) *Reader {
	return &Reader{
		dbRange: dbRange{
			first: first,
			last:  last,
		},
		db:            db,
		reader:        reader,
		log:           l,
		logging:       make(chan logMsg),
		executorInput: make(chan *executorInput),
		output:        make(chan *OutData),
		closed:        make(chan any),
		appWg:         wg,
		executorsWg:   new(sync.WaitGroup),
	}
}

// Start the Reader
func (e *Reader) Start(executors int, cfg *utils.Config) {
	e.appWg.Add(1)

	// start executors loops
	go e.doLog()
	go e.read()

	// start its executors
	e.initExecutors(executors, cfg)
}

// initExecutors creates and starts given number of ReplayExecutor
func (e *Reader) initExecutors(executors int, cfg *utils.Config) {
	// send info about executors
	e.logging <- logMsg{
		lvl: logging.INFO,
		msg: fmt.Sprintf("starting %v Executors", executors),
	}

	e.executors = make([]*ReplayExecutor, executors)
	for i := 0; i < executors; i++ {

		e.executors[i] = newExecutor(utils.GetChainConfig(cfg.ChainID), e.executorInput, e.output, e.executorsWg, e.closed, cfg.VmImpl, e.logging)
		e.executors[i].Start()
	}
}

// Stop the Reader
func (e *Reader) Stop() {
	select {
	case <-e.closed:
		return
	default:
		close(e.closed)
		e.executorsWg.Wait()
	}
}

// read retrieves req from reader (if not at the end) and pass the data alongside wanted archive
// to ReplayExecutor which executes the request into StateDB
func (e *Reader) read() {
	var (
		req    *iterator.RequestWithResponse
		wInput *executorInput
	)
	defer func() {
		e.reader.Close()
		close(e.output)
		close(e.executorInput)
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

		wInput = e.createExecutorInput(req)
		if wInput != nil {
			e.executorInput <- wInput
		}

	}

}

// createExecutorInput with data worker need to doExecute request into archive
func (e *Reader) createExecutorInput(req *iterator.RequestWithResponse) *executorInput {
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

	// todo remove
	wInput.blockID = 8999005

	// archive
	wInput.archive = e.getStateArchive(wInput.blockID)
	if wInput.archive == nil {
		return nil
	}

	return wInput
}

// getStateArchive for given block
func (e *Reader) getStateArchive(wantedBlockNumber uint64) state.StateDB {
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
func (e *Reader) decodeBlockNumber(params []interface{}, recordedBlockNumber uint64, returnedBlockID *uint64) bool {
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
func (e *Reader) isBlockNumberWithinRange(blockNumber uint64) bool {
	return blockNumber >= e.dbRange.first && blockNumber <= e.dbRange.last
}

// doLog is a thread for logging any messages from Reader or ReplayExecutor
func (e *Reader) doLog() {
	var l logMsg

	for {
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
		select {
		case <-e.closed:
			return
		default:
		}
	}
}
