package apireplay

import (
	"fmt"
	"sync"

	"github.com/Fantom-foundation/Aida/cmd/api-replay-cli/flags"
	"github.com/Fantom-foundation/Aida/iterator"
	"github.com/Fantom-foundation/Aida/logger"
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
	writer                                           *logWriter
	writerWg                                         *sync.WaitGroup
	writerClosed                                     chan any
}

// newController creates new instances of Controller, ReplayExecutors and Comparators
func newController(ctx *cli.Context, cfg *utils.Config, db state.StateDB, iter *iterator.FileReader) *Controller {

	// create close signals
	readerClosed := make(chan any)
	counterClosed := make(chan any)
	executorsClosed := make(chan any)
	comparatorsClosed := make(chan any)
	writerClosed := make(chan any)

	// create wait groups
	readerWg := new(sync.WaitGroup)
	executorsWg := new(sync.WaitGroup)
	comparatorsWg := new(sync.WaitGroup)
	counterWg := new(sync.WaitGroup)
	writerWg := new(sync.WaitGroup)

	// create instances
	reader := newReader(iter, logger.NewLogger(cfg.LogLevel, "Reader"), ctx.Uint64(flags.Skip.Name), readerClosed, readerWg)

	executors, output, counterInput := createExecutors(cfg, db, utils.GetChainConfig(cfg.ChainID), reader.output, executorsClosed, executorsWg)

	writer, writerInput := newWriter(cfg.LogLevel, writerClosed, writerWg)

	comparators, failure := createComparators(cfg, output, comparatorsClosed, writerInput, counterInput, comparatorsWg)

	counter := newCounter(counterClosed, counterInput, logger.NewLogger(cfg.LogLevel, "Counter"), counterWg)

	return &Controller{
		failure:           failure,
		log:               logger.NewLogger(ctx.String(logger.LogLevelFlag.Name), "Controller"),
		ctx:               ctx,
		Reader:            reader,
		Executors:         executors,
		Comparators:       comparators,
		counter:           counter,
		writer:            writer,
		readerClosed:      readerClosed,
		comparatorsClosed: comparatorsClosed,
		executorsClosed:   executorsClosed,
		counterClosed:     counterClosed,
		writerClosed:      writerClosed,
		readerWg:          readerWg,
		executorsWg:       executorsWg,
		comparatorsWg:     comparatorsWg,
		counterWg:         counterWg,
		writerWg:          writerWg,
	}
}

// Start all the services
func (r *Controller) Start() {
	r.Reader.Start()

	r.startExecutors()

	r.startComparators()

	r.counter.Start()
	r.writer.Start()

	go r.control()
}

// Stop all the services
func (r *Controller) Stop() {
	r.stopComparators()
	r.stopReader()
	r.stopExecutors()
	r.stopCounter()
	r.stopWriter()

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

func (r *Controller) stopWriter() {
	select {
	case <-r.writerClosed:
		return
	default:
		r.log.Info("Stopping log writer")
		close(r.writerClosed)
	}

}

// Wait until all wgs are done
func (r *Controller) Wait() {
	r.readerWg.Wait()

	r.executorsWg.Wait()

	r.comparatorsWg.Wait()

	r.counterWg.Wait()

	r.writerWg.Wait()
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
func createExecutors(cfg *utils.Config, db state.StateDB, chainCfg *params.ChainConfig, input chan *iterator.RequestWithResponse, closed chan any, wg *sync.WaitGroup) ([]*ReplayExecutor, chan *OutData, chan requestLog) {
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
		e[i] = newExecutor(cfg.First, cfg.Last, db, output, chainCfg, input, cfg.VmImpl, wg, closed, logger.NewLogger(cfg.LogLevel, fmt.Sprintf("Executor #%v", i)), counterInput)
	}

	return e, output, counterInput
}

// createComparators creates number of Comparators defined by the flag WorkersFlag divided by two
func createComparators(cfg *utils.Config, input chan *OutData, closed chan any, writerInput chan *comparatorError, counterInput chan requestLog, wg *sync.WaitGroup) ([]*Comparator, chan any) {
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
		c[i] = newComparator(input, logger.NewLogger(cfg.LogLevel, fmt.Sprintf("Comparator #%v", i)), closed, wg, cfg.ContinueOnFailure, writerInput, failure, counterInput)
	}

	return c, failure
}
