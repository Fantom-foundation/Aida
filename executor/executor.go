package executor

//go:generate mockgen -source executor.go -destination executor_mocks.go -package executor

import (
	"errors"
	"sync"

	"github.com/Fantom-foundation/Aida/state"
	substate "github.com/Fantom-foundation/Substate"
)

// ----------------------------------------------------------------------------
//                             Interfaces
// ----------------------------------------------------------------------------

// Executor is an entity coordinating the execution of transactions within a
// requested block range. It implements the decorator pattern, allowing
// extensions to monitor and annotate the execution at various hook-in points.
//
// When running sequentially, the general execution is structured as follows:
//
//	PreRun()
//	for each block {
//	   PreBlock()
//	   for each transaction {
//	       PreTransaction()
//	       Processor.Process(transaction)
//	       PostTransaction()
//	   }
//	   PostBlock()
//	}
//	PostRun()
//
// When running with multiple workers, the execution is structures like this:
//
//	PreRun()
//	for transaction in parallel {
//	    PreTransaction()
//	    Processor.Process(transaction)
//	    PostTransaction()
//	}
//	PostRun()
//
// Note that there are no block boundary events in the parallel mode.
//
// Each PreXXX() and PostXXX() is a hook-in point at which extensions may
// track information and/or interfere with the execution. For more details on
// the specific call-backs see the Extension interface below.
type Executor interface {
	// Run feeds all transactions of the given block range [from,to) to the
	// provided processor and performs the needed call-backs on the provided
	// extensions. If a processor or an extension returns an error, execution
	// stops with the reported error.
	// PreXXX events are delivered to the extensions in the given order, while
	// PostXXX events are delivered in reverse order. If any of the extensions
	// reports an error during processing of an event, the same event is still
	// delivered to the remaining extensions before processing is aborted.
	Run(params Params, processor Processor, extensions []Extension) error
}

// NewExecutor creates a new executor based on the given substate provider.
func NewExecutor(substate SubstateProvider) Executor {
	return &executor{substate}
}

// Params summarizes input parameters for a run of the executor.
type Params struct {
	// From is the begin of the range of blocks to be processed (inclusive).
	From int
	// From is the end of the range of blocks to be processed (exclusive).
	To int
	// State is an optional StateDB instance to be made available to the
	// processor and extensions during execution.
	State state.StateDB
	// NumProcessors is the number of concurrent goroutines to be used to
	// process blocks. If the number of processors is 1, transactions are
	// guranteed to be processed in-order. Any number <= 1 is considered to
	// be 1, thus the default value of 0 is valid.
	NumProcessors int
}

// Processor is an interface for the entity to which an executor is feeding
// transactions to.
type Processor interface {
	// Process is called on each transaction in the range of blocks covered
	// by an Executor run. When running with multiple workers, the Process
	// function is required to be thread safe.
	Process(State) error
}

// Extension is an interface for modulare annotations to the execution of
// a range of transactions. During various stages, methods of extensions are
// called, enabling them to monitor and/or interfere with the execution.
// Since blocks may be processed in parallel, callbacks are generally
// required to be thread safe (with the exception of the Pre-/ and PostRun)
// callback.
type Extension interface {
	// PreRun is called before the begin of the execution of a block range,
	// even if the range is empty. The provided state lists the initial block
	// of the range. For every run, PreRun is only called once, before any
	// other call-back. If an error is reported, execution will abort after
	// PreRun has been called on all registered Extensions.
	PreRun(State) error

	// PostRun is guranteed to be called at the end of each execution. An
	// execution may end successfully, if no exception has been produced by
	// the Processor or any Extension, or in a failure state, if errors
	// have been produced. In case of a successful execution, the provided
	// state lists the first non-executiond block, while in an error case
	// it references the last transaction attempted to be processed. Also,
	// the second parameter contains the error causing the abort.
	PostRun(State, error) error

	// PreBlock is called once before the begin of processing a block with
	// the state containing the number of the Block. This function is not
	// called when running with multiple workers.
	PreBlock(State) error

	// PreBlock is called once after the end of processing a block with
	// the state containing the number of the Block and the last transaction
	// processed in the block. This function is not called when running with
	// multiple workers.
	PostBlock(State) error

	// PreTransaction is called once before each transaction with the state
	// listing the block number, the transaction number, and the substate data
	// providing the input for the subsequent execution of the transaction.
	// When running with multiple workers, this function may be called
	// concurrently, and must thus be thread safe.
	PreTransaction(State) error

	// PostTransaction is called once after each transaction with the state
	// listing the block number, the transaction number, and the substate data
	// providing the input for the subsequent execution of the transaction.
	// When running with multiple workers, this function may be called
	// concurrently, and must thus be thread safe.
	PostTransaction(State) error
}

// State summarizes the current state of an execution and is passed to
// Processors and Extensions as an input for their actions.
type State struct {
	// Block the current block number, valid for all call-backs.
	Block int

	// Transaction is the transaction number of the current transaction within
	// its respective block. It is only valid for PreTransaction, PostTransaction,
	// PostBlock, and for PostRun events in case of an abort.
	Transaction int

	// Substate is the input required for the current transaction. It is only
	// valid for Pre- and PostTransaction events.
	Substate *substate.Substate

	// State is an optional StateDB instance manipulated during by the processor
	// and extensions of a block-range execution.
	State state.StateDB
}

// ----------------------------------------------------------------------------
//                               Implementations
// ----------------------------------------------------------------------------

type executor struct {
	substate SubstateProvider
}

func (e *executor) Run(params Params, processor Processor, extensions []Extension) (err error) {
	state := State{State: params.State}

	defer func() {
		err = errors.Join(
			err,
			signalPostRun(state, err, extensions),
		)
	}()

	state.Block = params.From
	if err := signalPreRun(state, extensions); err != nil {
		return err
	}

	if params.NumProcessors <= 1 {
		return e.runSequential(params, processor, extensions, &state)
	}
	return e.runParallel(params, processor, extensions, &state)
}

func (e *executor) runSequential(params Params, processor Processor, extensions []Extension, state *State) error {
	first := true
	err := e.substate.Run(params.From, params.To, func(tx Transaction) error {
		if first {
			state.Block = tx.Block
			if err := signalPreBlock(*state, extensions); err != nil {
				return err
			}
			first = false
		} else if state.Block != tx.Block {
			if err := signalPostBlock(*state, extensions); err != nil {
				return err
			}
			state.Block = tx.Block
			if err := signalPreBlock(*state, extensions); err != nil {
				return err
			}
		}
		state.Transaction = tx.Transaction
		if err := signalPreTransaction(*state, extensions); err != nil {
			return err
		}
		if err := processor.Process(*state); err != nil {
			return err
		}
		if err := signalPostTransaction(*state, extensions); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return err
	}

	// Finish final block.
	if !first {
		if err := signalPostBlock(*state, extensions); err != nil {
			return err
		}
		state.Block = params.To
	}

	return nil
}

func (e *executor) runParallel(params Params, processor Processor, extensions []Extension, state *State) error {
	numWorkers := params.NumProcessors

	// A channel that is closed if an error occurs.
	isAborted := make(chan bool)

	// Start one go-routine forwarding transactions from the provider to a local channel.
	var forwardErr error
	transactions := make(chan *Transaction, 10*numWorkers)
	go func() {
		defer close(transactions)
		abortErr := errors.New("aborted")
		err := e.substate.Run(params.From, params.To, func(tx Transaction) error {
			select {
			case transactions <- &tx:
				return nil
			case <-isAborted:
				return abortErr
			}
		})
		if err != abortErr {
			forwardErr = err
		}
	}()

	// Start numWorkers go-routines processing transactions in parallel.
	var wg sync.WaitGroup
	wg.Add(numWorkers)
	workerErrs := make([]error, numWorkers)
	for i := 0; i < numWorkers; i++ {
		go func(i int) {
			defer wg.Done()
			abort := func(err error) {
				workerErrs[i] = err
				select {
				case <-isAborted:
					return
				default:
					close(isAborted)
				}
			}
			for {
				select {
				case tx := <-transactions:
					if tx == nil {
						return
					}
					localState := *state
					localState.Block = tx.Block
					localState.Transaction = tx.Transaction
					if err := signalPreTransaction(localState, extensions); err != nil {
						abort(err)
						return
					}
					if err := processor.Process(localState); err != nil {
						abort(err)
						return
					}
					if err := signalPostTransaction(localState, extensions); err != nil {
						abort(err)
						return
					}
				case <-isAborted:
					return
				}
			}
		}(i)
	}
	wg.Wait()
	return errors.Join(
		forwardErr,
		errors.Join(workerErrs...),
	)
}

func signalPreRun(state State, extensions []Extension) error {
	return forEachForward(extensions, func(extension Extension) error {
		return extension.PreRun(state)
	})
}

func signalPostRun(state State, err error, extensions []Extension) error {
	return forEachBackward(extensions, func(extension Extension) error {
		extension.PostRun(state, err)
		return nil
	})
}

func signalPreBlock(state State, extensions []Extension) error {
	return forEachForward(extensions, func(extension Extension) error {
		return extension.PreBlock(state)
	})
}

func signalPostBlock(state State, extensions []Extension) error {
	return forEachBackward(extensions, func(extension Extension) error {
		return extension.PostBlock(state)
	})
}

func signalPreTransaction(state State, extensions []Extension) error {
	return forEachForward(extensions, func(extension Extension) error {
		return extension.PreTransaction(state)
	})
}

func signalPostTransaction(state State, extensions []Extension) error {
	return forEachBackward(extensions, func(extension Extension) error {
		return extension.PostTransaction(state)
	})
}

func forEachForward(extensions []Extension, op func(extension Extension) error) error {
	errs := []error{}
	for _, extension := range extensions {
		if err := op(extension); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func forEachBackward(extensions []Extension, op func(extension Extension) error) error {
	errs := []error{}
	for i := len(extensions) - 1; i >= 0; i-- {
		if err := op(extensions[i]); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}
