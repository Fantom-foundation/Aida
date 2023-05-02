package apireplay

import (
	"encoding/json"
	"io"
	"strings"
	"sync"

	"github.com/Fantom-foundation/Aida/iterator"
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
	defer func() {
		close(r.output)
		r.wg.Done()
	}()

	var val *iterator.RequestWithResponse

	for r.iter.Next() {
		select {
		case <-r.closed:
			return

		default:
			// did iter emit an error?
			if r.iter.Error() != nil {
				if r.iter.Error() == io.EOF || r.iter.Error().Error() == "unexpected EOF" {
					return
				}
				r.log.Fatalf("unexpected iter err; %v", r.iter.Error())
			}

			val = r.iter.Value()

		}

		select {
		case <-r.closed:
			return

		// retrieve the data from iterator and send them to executors
		case r.output <- val:

		}
	}
}
