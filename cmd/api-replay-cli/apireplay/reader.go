package apireplay

import (
	"encoding/json"
	"math/big"
	"strings"
	"sync"

	"github.com/Fantom-foundation/Aida/iterator"
	"github.com/Fantom-foundation/Aida/state"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/op/go-logging"
)

const (
	maxIterErrors = 5 // maximum consecutive errors emitted by comparator before program panics
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
	db      state.StateDB
	output  chan *executorInput
	iter    *iterator.FileReader
	closed  chan any
	log     *logging.Logger
	wg      *sync.WaitGroup
	dbRange dbRange
}

// newReader returns new instance of Reader
func newReader(first, last uint64, db state.StateDB, iterator *iterator.FileReader, l *logging.Logger, closed chan any, wg *sync.WaitGroup) *Reader {
	return &Reader{
		dbRange: dbRange{
			first: first,
			last:  last,
		},
		db:     db,
		iter:   iterator,
		output: make(chan *executorInput),
		log:    l,
		closed: closed,
		wg:     wg,
	}
}

// Start the Reader
func (r *Reader) Start() {
	r.log.Info("starting reader")
	// start readers loop
	go r.read()
	r.wg.Add(1)
}

// read retrieves req from iter (if not at the end) and pass the data alongside wanted archive
// to ReplayExecutor which executes the request into StateDB
func (r *Reader) read() {
	var (
		iterErrors uint16 = 1
		req        *iterator.RequestWithResponse
		wInput     *executorInput
	)
	defer func() {
		r.iter.Close()
		close(r.output)
		r.wg.Done()
	}()

	for r.iter.Next() {
		select {
		case <-r.closed:
			return
		default:
		}

		// did iter emit an error?
		if r.iter.Error() != nil {
			// if iterator errors 5 times in a row, exit the program with an error
			if iterErrors >= maxIterErrors {
				r.log.Fatalf("iterator reached limit of number of consecutive errors; err: %v", r.iter.Error())
			}
			r.log.Errorf("error loading recordings; %v\nretry number %v\n", r.iter.Error().Error(), iterErrors)
			iterErrors++
			continue
		}

		// reset the error counter
		iterErrors = 1

		// retrieve the data from iterator
		req = r.iter.Value()

		wInput = r.createExecutorInput(req)
		if wInput != nil {
			r.output <- wInput
		}

	}

}

// createExecutorInput with data worker need to doExecute request into archive
func (r *Reader) createExecutorInput(req *iterator.RequestWithResponse) *executorInput {
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
		r.log.Error("both recorded response and recorded error are nil; skipping")
		return nil
	}

	// request
	wInput.req = req

	if !r.decodeBlockNumber(req.Query.Params, recordedBlockID, &wInput.blockID) {
		r.log.Errorf("cannot decode block number; skipping\nParams: %v", req.Query.Params[1])
		return nil
	}

	// archive
	wInput.archive = r.getStateArchive(wInput.blockID)
	if wInput.archive == nil {
		return nil
	}

	return wInput
}

// getStateArchive for given block
func (r *Reader) getStateArchive(wantedBlockNumber uint64) state.StateDB {
	if !r.isBlockNumberWithinRange(wantedBlockNumber) {
		r.log.Debug("request out of StateDB block range\nSKIPPING\n")
		return nil
	}

	// load the archive itself
	var err error
	archive, err := r.db.GetArchiveState(wantedBlockNumber)
	if err != nil {
		r.log.Debugf("cannot retrieve archive for block id #%v; skipping; err: %v", wantedBlockNumber, err)
		return nil
	}

	return archive
}

// decodeBlockNumber finds what block number request wants
func (r *Reader) decodeBlockNumber(params []interface{}, recordedBlockNumber uint64, returnedBlockID *uint64) bool {

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
func (r *Reader) isBlockNumberWithinRange(blockNumber uint64) bool {
	return blockNumber >= r.dbRange.first && blockNumber <= r.dbRange.last
}
