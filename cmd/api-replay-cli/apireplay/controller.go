package apireplay

import (
	"fmt"
	"sync"

	"github.com/Fantom-foundation/Aida/iterator"
	"github.com/Fantom-foundation/Aida/state"
	"github.com/Fantom-foundation/Aida/utils"
	"github.com/ethereum/go-ethereum/params"
	"github.com/google/martian/log"
	"github.com/op/go-logging"
	"github.com/urfave/cli/v2"
)

const (
	bufferSize        = 3000
	counterBufferSize = 2400
)

// Controller controls and looks after all threads within the api-replay package
// Reader reads data from iterator - one thread is used for reader
// Executors execute requests into StateDB - number of Executors is defined by the WorkersFlag
// Comparators compare results returned by StateDB with recorded results
// - number of Comparators is defined by the WorkersFlag divided by two
type Controller struct {
	ctx                                              *cli.Context
	Reader                                           *Reader
	Executors                                        []*ReplayExecutor
	Comparators                                      []*Comparator
	readerClosed, executorsClosed, comparatorsClosed chan any
	readerWg, executorsWg, comparatorsWg             *sync.WaitGroup
	log                                              *logging.Logger
	failure                                          chan any
	counter                                          *requestCounter
	counterWg                                        *sync.WaitGroup
	counterClosed                                    chan any
}

// newController creates new instances of Controller, ReplayExecutors and Comparators
func newController(ctx *cli.Context, cfg *utils.Config, db state.StateDB, iter *iterator.FileReader) *Controller {

	// create close signals
	readerClosed := make(chan any)
	counterClosed := make(chan any)
	executorsClosed := make(chan any)
	comparatorsClosed := make(chan any)

	// create wait groups
	readerWg := new(sync.WaitGroup)
	executorsWg := new(sync.WaitGroup)
	comparatorsWg := new(sync.WaitGroup)
	counterWg := new(sync.WaitGroup)

	// create instances
	reader := newReader(iter, utils.NewLogger(cfg.LogLevel, "Reader"), readerClosed, readerWg)

	executors, output, counterInput := createExecutors(cfg, db, ctx, utils.GetChainConfig(cfg.ChainID), reader.output, executorsClosed, executorsWg)

	counter := newCounter(counterClosed, counterInput, utils.NewLogger(cfg.LogLevel, "Counter"), counterWg)

	comparators, failure := createComparators(cfg, output, comparatorsClosed, comparatorsWg)

	return &Controller{
		failure:           failure,
		log:               utils.NewLogger(ctx.String(utils.LogLevelFlag.Name), "Controller"),
		ctx:               ctx,
		Reader:            reader,
		Executors:         executors,
		Comparators:       comparators,
		counter:           counter,
		readerClosed:      readerClosed,
		comparatorsClosed: comparatorsClosed,
		executorsClosed:   executorsClosed,
		counterClosed:     counterClosed,
		readerWg:          readerWg,
		executorsWg:       executorsWg,
		comparatorsWg:     comparatorsWg,
		counterWg:         counterWg,
	}
}

// Start all the services
func (r *Controller) Start() {
	r.Reader.Start()

	r.startExecutors()

	r.startComparators()

	r.counter.Start()

	go r.control()
}

// Stop all the services
func (r *Controller) Stop() {
	r.stopComparators()
	r.stopReader()
	r.stopExecutors()
	r.stopCounter()

	r.comparatorsWg.Wait()
	r.executorsWg.Wait()
	r.counterWg.Wait()
	r.readerWg.Wait()
	r.log.Notice("All services has been stopped")
}

// startExecutors and their loops
func (r *Controller) startExecutors() {
	r.log.Infof("Starting %v executor", len(r.Executors))
	for _, e := range r.Executors {
		e.Start()
		r.executorsWg.Add(1)
	}
}

// startComparators and their loops
func (r *Controller) startComparators() {
	r.log.Infof("Starting %v comparators", len(r.Comparators))
	for _, c := range r.Comparators {
		c.Start()
		r.comparatorsWg.Add(1)
	}

}

// stopCounter closes the counters close signal
func (r *Controller) stopCounter() {
	select {
	case <-r.counterClosed:
		return
	default:
		r.log.Info("Stopping counter")
		close(r.counterClosed)
	}

}

// stopReader closes the Readers close signal
func (r *Controller) stopReader() {
	select {
	case <-r.readerClosed:
		return
	default:
		r.log.Info("Stopping reader")
		close(r.readerClosed)
	}

}

// stopExecutors closes the Executors close signal
func (r *Controller) stopExecutors() {
	select {
	case <-r.executorsClosed:
		return
	default:
		r.log.Info("Stopping executors")
		close(r.executorsClosed)
	}

}

// stopComparators closes the Comparators close signal
func (r *Controller) stopComparators() {
	// stop comparators input, so it still reads the rest of data in the chanel and exits once its empty
	select {
	case <-r.comparatorsClosed:
		return
	default:
		r.log.Info("Stopping comparators")
		close(r.comparatorsClosed)
	}

}

// Wait until all wgs are done
func (r *Controller) Wait() {
	r.readerWg.Wait()

	r.executorsWg.Wait()

	r.comparatorsWg.Wait()

	r.counterWg.Wait()
}

// control looks for ctx.Done, if it triggers, Controller stops all the services
func (r *Controller) control() {
	for {
		select {
		case <-r.ctx.Done():
			r.Stop()
			r.log.Errorf("ctx err: %v", r.ctx.Err())
			return
		case <-r.failure:
			r.Stop()
			return
		case <-r.readerClosed:
			r.Stop()
			return
		case <-r.counterClosed:
			r.Stop()
			return
		}
	}
}

// createExecutors creates number of Executors defined by the flag WorkersFlag
func createExecutors(cfg *utils.Config, db state.StateDB, ctx *cli.Context, chainCfg *params.ChainConfig, input chan *iterator.RequestWithResponse, closed chan any, wg *sync.WaitGroup) ([]*ReplayExecutor, chan *OutData, chan requestLog) {
	var executors int

	log.Infof("creating %v executors", cfg.Workers)

	output := make(chan *OutData, bufferSize)

	// do we want a single-thread replay
	if cfg.Workers == 1 {
		executors = 1
	} else {
		executors = cfg.Workers / 2
	}

	e := make([]*ReplayExecutor, executors)
	counterInput := make(chan requestLog, counterBufferSize)
	for i := 0; i < executors; i++ {
		e[i] = newExecutor(cfg.First, cfg.Last, db, output, chainCfg, input, cfg.VmImpl, wg, closed, utils.NewLogger(cfg.LogLevel, fmt.Sprintf("Executor #%v", i)), counterInput)
	}

	return e, output, counterInput
}

// createComparators creates number of Comparators defined by the flag WorkersFlag divided by two
func createComparators(cfg *utils.Config, input chan *OutData, closed chan any, wg *sync.WaitGroup) ([]*Comparator, chan any) {
	var (
		comparators int
	)

	// do we want a single-thread replay
	if cfg.Workers == 1 {
		comparators = 1
	} else {
		comparators = cfg.Workers / 2
	}

	log.Infof("creating %v comparators", comparators)

	c := make([]*Comparator, comparators)
	failure := make(chan any)
	for i := 0; i < comparators; i++ {
		c[i] = newComparator(input, utils.NewLogger(cfg.LogLevel, fmt.Sprintf("Comparator #%v", i)), closed, wg, cfg.ContinueOnFailure, failure)
	}

	return c, failure
}
