package apireplay

import (
	"sync"

	"github.com/Fantom-foundation/Aida/cmd/api-replay-cli/flags"
	"github.com/Fantom-foundation/Aida/iterator"
	"github.com/Fantom-foundation/Aida/state"
	"github.com/Fantom-foundation/Aida/utils"
	"github.com/ethereum/go-ethereum/params"
	"github.com/google/martian/log"
	"github.com/op/go-logging"
	"github.com/urfave/cli/v2"
)

const bufferSize = 100

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
}

// newController creates new instances of Controller, ReplayExecutors and Comparators
func newController(ctx *cli.Context, cfg *utils.Config, db state.StateDB, iter *iterator.FileReader) *Controller {

	// create close signals
	readerClosed := make(chan any)
	executorsClosed := make(chan any)
	comparatorsClosed := make(chan any)

	// create wait groups
	readerWg := new(sync.WaitGroup)
	executorsWg := new(sync.WaitGroup)
	comparatorsWg := new(sync.WaitGroup)

	// create instances
	reader := newReader(iter, newLogger(ctx), readerClosed, readerWg, ctx.Uint64(flags.SkipFlag.Name))

	executors, output := createExecutors(cfg.First, cfg.Last, db, ctx, utils.GetChainConfig(cfg.ChainID), reader.output, cfg.VmImpl, executorsClosed, executorsWg)

	comparators, failure := createComparators(ctx, output, comparatorsClosed, comparatorsWg)

	return &Controller{
		failure:           failure,
		log:               newLogger(ctx),
		ctx:               ctx,
		Reader:            reader,
		Executors:         executors,
		Comparators:       comparators,
		readerClosed:      readerClosed,
		comparatorsClosed: comparatorsClosed,
		executorsClosed:   executorsClosed,
		readerWg:          readerWg,
		executorsWg:       executorsWg,
		comparatorsWg:     comparatorsWg,
	}
}

// Start all the services
func (r *Controller) Start() {
	r.Reader.Start()

	r.startExecutors()

	r.startComparators()

	go r.control()
}

// Stop all the services
func (r *Controller) Stop() {
	r.stopComparators()
	r.stopReader()
	r.stopExecutors()

	r.readerWg.Wait()
	r.executorsWg.Wait()
	r.comparatorsWg.Wait()
	r.log.Notice("all services has been stopped")
}

// startExecutors and their loops
func (r *Controller) startExecutors() {
	for i, e := range r.Executors {
		r.log.Infof("starting executor #%v", i+1)
		e.Start()

	}
}

// startComparators and their loops
func (r *Controller) startComparators() {
	for i, c := range r.Comparators {
		r.log.Infof("starting comparator #%v", i+1)
		c.Start()
	}
}

// stopReader closes the Readers close signal and waits until its wg is done
func (r *Controller) stopReader() {
	select {
	case <-r.readerClosed:
		return
	default:
		r.log.Notice("stopping reader")
		close(r.readerClosed)
	}

}

// stopExecutors closes the Executors close signal and waits until all the Executors are done
func (r *Controller) stopExecutors() {
	select {
	case <-r.executorsClosed:
		return
	default:
		r.log.Notice("stopping executors")
		close(r.executorsClosed)
	}

}

// stopComparators closes the Comparators close signal and waits until all the Comparators are done
func (r *Controller) stopComparators() {
	// stop comparators input, so it still reads the rest of data in the chanel and exits once its empty
	select {
	case <-r.comparatorsClosed:
		return
	default:
		r.log.Notice("stopping comparators")
		close(r.comparatorsClosed)
	}

}

// Wait until all wgs are done
func (r *Controller) Wait() {
	r.readerWg.Wait()
	r.log.Info("reader done")

	r.executorsWg.Wait()
	r.log.Info("executors done")

	r.comparatorsWg.Wait()
	r.log.Info("comparators done")

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
		}
	}
}

// createExecutors creates number of Executors defined by the flag WorkersFlag
func createExecutors(first, last uint64, db state.StateDB, ctx *cli.Context, chainCfg *params.ChainConfig, input chan *iterator.RequestWithResponse, vmImpl string, closed chan any, wg *sync.WaitGroup) ([]*ReplayExecutor, chan *OutData) {
	log.Infof("creating %v executors", ctx.Int(flags.WorkersFlag.Name))

	output := make(chan *OutData, bufferSize)

	executors := ctx.Int(flags.WorkersFlag.Name)

	e := make([]*ReplayExecutor, executors)
	for i := 0; i < executors; i++ {
		e[i] = newExecutor(first, last, db, output, chainCfg, input, vmImpl, wg, closed, newLogger(ctx))
	}

	return e, output
}

// createComparators creates number of Comparators defined by the flag WorkersFlag divided by two
func createComparators(ctx *cli.Context, input chan *OutData, closed chan any, wg *sync.WaitGroup) ([]*Comparator, chan any) {
	var (
		comparators int
	)

	// do we want a single-thread replay
	if ctx.Int(flags.WorkersFlag.Name) == 1 {
		comparators = 1
	} else {
		comparators = ctx.Int(flags.WorkersFlag.Name)
	}

	log.Infof("creating %v comparators", comparators)

	c := make([]*Comparator, comparators)
	failure := make(chan any)
	for i := 0; i < comparators; i++ {
		c[i] = newComparator(input, newLogger(ctx), closed, wg, ctx.Bool(flags.ContinueOnFailure.Name), failure)
	}

	return c, failure
}
