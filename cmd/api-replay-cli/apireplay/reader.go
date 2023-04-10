package apireplay

import (
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"strings"
	"sync"
	"time"

	"github.com/Fantom-foundation/Aida/iterator"
	"github.com/Fantom-foundation/Aida/state"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/op/go-logging"
)

const (
	statisticsLogFrequency = 10 * time.Second
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
	skipN   uint64
	dbRange dbRange
}

// newReader returns new instance of Reader
func newReader(first, last uint64, db state.StateDB, iterator *iterator.FileReader, l *logging.Logger, closed chan any, wg *sync.WaitGroup, skipN uint64) *Reader {
	l.Info("creating reader")
	return &Reader{
		dbRange: dbRange{
			first: first,
			last:  last,
		},
		db:     db,
		iter:   iterator,
		output: make(chan *executorInput, bufferSize),
		log:    l,
		closed: closed,
		skipN:  skipN,
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
		req             *iterator.RequestWithResponse
		wInput          *executorInput
		start           time.Time
		ticker          *time.Ticker
		total, executed uint64
		methods         = make(map[string]uint32)
	)
	defer func() {
		r.logStatistics(start, total, executed)
		close(r.output)
		r.wg.Done()
	}()

	if r.skipN > 0 {
		r.log.Noticef("skipping first %v requests", r.skipN)
	}

	start = time.Now()
	ticker = time.NewTicker(statisticsLogFrequency)

	for r.iter.Next() {
		select {
		case <-r.closed:
			return
		case <-ticker.C:
			r.logStatistics(start, total, executed)
			r.log.Notice(methods)
		default:
			total++

			// do we want to skip first n requests?
			if r.skipN > total {
				continue
			} else if r.skipN == total {
				// reset counter
				r.skipN = 0
				total = 1
			}

			// did iter emit an error?
			if r.iter.Error() != nil {
				if r.iter.Error() == io.EOF || r.iter.Error().Error() == "unexpected EOF" {
					return
				}
				r.log.Fatalf("unexpected iter err; %v", r.iter.Error())
			}

			// retrieve the data from iterator
			req = r.iter.Value()

			methods[req.Query.Method]++

			if !strings.Contains(req.Query.Method, "getBalance") {
				continue
			} else {
				fmt.Println("a")
			}

			wInput = r.createExecutorInput(req)

			if wInput != nil {

				if strings.Contains(req.Query.Method, "getBalance") {
					var err error
					wInput.archive, err = r.db.GetArchiveState(40109000)
					if err != nil {
						r.log.Fatal(err)
					}
					wInput.req.Query.Params[0] = "0x8975967c70e21a9f054eDcAC82126B239a23b19E"
				}
				r.output <- wInput
				executed++
			}

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
		r.log.Debugf("cannot decode block number; skipping\nParams: %v", req.Query.Params[1])
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
		r.log.Debugf("request with blockID #%v out of StateDB block range; SKIPPING", wantedBlockNumber)
		return nil
	}

	// load the archive itself
	var err error
	archive, err := r.db.GetArchiveState(wantedBlockNumber)
	if err != nil {
		r.log.Errorf("cannot retrieve archive for block id #%v; skipping; err: %v", wantedBlockNumber, err)
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

// logStatistics about time, executed and total read requests. Frequency of logging depends on statisticsLogFrequency
func (r *Reader) logStatistics(start time.Time, total uint64, executed uint64) {
	elapsed := time.Since(start)
	r.log.Noticef("Elapsed time: %v\n"+
		"Read requests:%v\n"+
		"Out of which were skipped due to not being in StateDB block range: %v", elapsed, total, total-executed)
}
