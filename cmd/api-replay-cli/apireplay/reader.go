package apireplay

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/Fantom-foundation/Aida/iterator"
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
	output  chan *iterator.RequestWithResponse
	iter    *iterator.FileReader
	closed  chan any
	log     *logging.Logger
	wg      *sync.WaitGroup
	builder *strings.Builder // use builder for faster execution when logging and cleaner code
}

// newReader returns new instance of Reader
func newReader(iter *iterator.FileReader, l *logging.Logger, closed chan any, wg *sync.WaitGroup) *Reader {
	l.Info("creating reader")
	return &Reader{
		iter:    iter,
		output:  make(chan *iterator.RequestWithResponse, bufferSize),
		log:     l,
		closed:  closed,
		wg:      wg,
		builder: new(strings.Builder),
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
		start  time.Time
		ticker *time.Ticker
		total  uint64
		stat   = make(map[string]uint64)
	)
	defer func() {
		r.logStatistics(start, total, stat)
		close(r.output)
		r.wg.Done()
	}()

	start = time.Now()
	ticker = time.NewTicker(statisticsLogFrequency)

	for r.iter.Next() {
		select {
		case <-r.closed:
			return
		case <-ticker.C:
			r.logStatistics(start, total, stat)

		default:
			total++

			// did iter emit an error?
			if r.iter.Error() != nil {
				if r.iter.Error() == io.EOF || r.iter.Error().Error() == "unexpected EOF" {
					return
				}
				r.log.Fatalf("unexpected iter err; %v", r.iter.Error())
			}

			val := r.iter.Value()

			// retrieve the data from iterator and send them to executors
			r.output <- val

			// add req to statistics
			stat[val.Query.Method]++
		}
	}
}

// logStatistics about time, executed and total read requests. Frequency of logging depends on statisticsLogFrequency
func (r *Reader) logStatistics(start time.Time, req uint64, stat map[string]uint64) {
	elapsed := time.Since(start)
	r.builder.WriteString(fmt.Sprintf("Elapsed time: %v\n\n", elapsed))

	r.builder.WriteString(fmt.Sprintf("\tTotal requests:%v\n\n", req))

	for m, c := range stat {
		r.builder.WriteString(fmt.Sprintf("\t%v: %v\n", m, c))
	}

	r.log.Notice(r.builder.String())
	r.builder.Reset()
}
