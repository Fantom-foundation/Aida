package apireplay

import (
	"strings"
	"sync"

	"github.com/op/go-logging"
)

// Comparator compares data from StateDB and expected data recorded on API server
// This data is retrieved from Reader
type Comparator struct {
	input             chan *OutData
	log               *logging.Logger
	closed            chan any
	wg                *sync.WaitGroup
	continueOnFailure bool
	failure           chan any
	builder           *strings.Builder
}

// newComparator returns new instance of Comparator
func newComparator(input chan *OutData, log *logging.Logger, closed chan any, wg *sync.WaitGroup, continueOnFailure bool, failure chan any) *Comparator {
	return &Comparator{
		failure:           failure,
		input:             input,
		log:               log,
		closed:            closed,
		wg:                wg,
		continueOnFailure: continueOnFailure,
		builder:           new(strings.Builder),
	}
}

// Start the Comparator
func (c *Comparator) Start() {
	c.wg.Add(1)
	go c.compare()
}

// compare reads data from Reader and compares them. If doCompare func returns error,
// the error is logged since the results do not match
func (c *Comparator) compare() {
	var (
		data *OutData
		ok   bool
	)

	defer func() {
		c.wg.Done()
	}()

	for {

		select {
		case <-c.closed:
			return
		case data, ok = <-c.input:

			// stop Comparator if input is closed
			if !ok {
				return
			}

			if err := c.doCompare(data); err != nil {

				// we do not want the program to exit when recording has internal error
				if err.typ == internalError {
					c.log.Debug(err)
					continue
				}

				// log the mismatched data
				c.log.Critical(err)

				// do we want to exit?
				if !c.continueOnFailure {
					c.fail()
				}
			}

		}
	}

}

// doCompare calls adequate comparing function for given method
func (c *Comparator) doCompare(data *OutData) (err *comparatorError) {
	switch data.MethodBase {
	case "getBalance":
		err = compareBalance(data, c.builder)
	case "getTransactionCount":
		err = compareTransactionCount(data, c.builder)
	case "call":
		err = compareCall(data, c.builder)
	case "estimateGas":
		// estimateGas is currently not suitable for replay since the estimation  in geth is always calculated for current state
		// that means recorded result and result returned by StateDB are not comparable
	case "getCode":
		err = compareCode(data, c.builder)
	case "getStorageAt":
		err = compareStorageAt(data, c.builder)
	}

	return
}

// fail sends signal to controller that mismatched results occurred
func (c *Comparator) fail() {
	select {
	case <-c.failure:
		return
	default:
		close(c.failure)
	}
}
