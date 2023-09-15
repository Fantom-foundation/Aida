package executor

import (
	"errors"
	"fmt"
	"sync"
	"testing"

	"github.com/Fantom-foundation/Aida/state"
	substate "github.com/Fantom-foundation/Substate"
	"go.uber.org/mock/gomock"
)

func TestProcessor_ProcessorGetsCalledForEachTransaction(t *testing.T) {
	ctrl := gomock.NewController(t)
	substate := NewMockSubstateProvider(ctrl)
	processor := NewMockProcessor(ctrl)

	substate.EXPECT().
		Run(10, 12, gomock.Any()).
		DoAndReturn(func(from int, to int, consume Consumer) error {
			// We simulate two transactions per block.
			for i := from; i < to; i++ {
				consume(TransactionInfo{i, 0, nil})
				consume(TransactionInfo{i, 1, nil})
			}
			return nil
		})

	gomock.InOrder(
		processor.EXPECT().Process(AtTransaction(10, 0), gomock.Any()),
		processor.EXPECT().Process(AtTransaction(10, 1), gomock.Any()),
		processor.EXPECT().Process(AtTransaction(11, 0), gomock.Any()),
		processor.EXPECT().Process(AtTransaction(11, 1), gomock.Any()),
	)

	executor := NewExecutor(substate)
	if err := executor.Run(Params{From: 10, To: 12}, processor, nil); err != nil {
		t.Errorf("execution failed: %v", err)
	}
}

func TestProcessor_FailingProcessorStopsExecution(t *testing.T) {
	ctrl := gomock.NewController(t)
	substate := NewMockSubstateProvider(ctrl)
	processor := NewMockProcessor(ctrl)

	substate.EXPECT().
		Run(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(from int, to int, consume Consumer) error {
			for i := from; i < to; i++ {
				if err := consume(TransactionInfo{i, 0, nil}); err != nil {
					return err
				}
			}
			return nil
		})

	stop := fmt.Errorf("stop!")
	gomock.InOrder(
		processor.EXPECT().Process(gomock.Any(), gomock.Any()).Times(3),
		processor.EXPECT().Process(gomock.Any(), gomock.Any()).Return(stop),
	)

	executor := NewExecutor(substate)
	if got, want := executor.Run(Params{From: 10, To: 20}, processor, nil), stop; !errors.Is(got, want) {
		t.Errorf("execution did not produce expected error, wanted %v, got %v", got, want)
	}
}

func TestProcessor_ExtensionsGetSignaledAboutEvents(t *testing.T) {
	ctrl := gomock.NewController(t)
	substate := NewMockSubstateProvider(ctrl)
	processor := NewMockProcessor(ctrl)
	extension := NewMockExtension(ctrl)

	substate.EXPECT().
		Run(10, 12, gomock.Any()).
		DoAndReturn(func(from int, to int, consume Consumer) error {
			// We simulate two transactions per block.
			for i := from; i < to; i++ {
				consume(TransactionInfo{i, 7, nil})
				consume(TransactionInfo{i, 9, nil})
			}
			return nil
		})

	gomock.InOrder(
		extension.EXPECT().PreRun(AtBlock(10), gomock.Any()),

		extension.EXPECT().PreBlock(AtBlock(10), gomock.Any()),
		extension.EXPECT().PreTransaction(AtTransaction(10, 7), gomock.Any()),
		processor.EXPECT().Process(AtTransaction(10, 7), gomock.Any()),
		extension.EXPECT().PostTransaction(AtTransaction(10, 7), gomock.Any()),
		extension.EXPECT().PreTransaction(AtTransaction(10, 9), gomock.Any()),
		processor.EXPECT().Process(AtTransaction(10, 9), gomock.Any()),
		extension.EXPECT().PostTransaction(AtTransaction(10, 9), gomock.Any()),
		extension.EXPECT().PostBlock(AtTransaction(10, 9), gomock.Any()),

		extension.EXPECT().PreBlock(AtBlock(11), gomock.Any()),
		extension.EXPECT().PreTransaction(AtTransaction(11, 7), gomock.Any()),
		processor.EXPECT().Process(AtTransaction(11, 7), gomock.Any()),
		extension.EXPECT().PostTransaction(AtTransaction(11, 7), gomock.Any()),
		extension.EXPECT().PreTransaction(AtTransaction(11, 9), gomock.Any()),
		processor.EXPECT().Process(AtTransaction(11, 9), gomock.Any()),
		extension.EXPECT().PostTransaction(AtTransaction(11, 9), gomock.Any()),
		extension.EXPECT().PostBlock(AtTransaction(11, 9), gomock.Any()),

		extension.EXPECT().PostRun(AtBlock(12), gomock.Any(), nil),
	)

	executor := NewExecutor(substate)
	if err := executor.Run(Params{From: 10, To: 12}, processor, []Extension{extension}); err != nil {
		t.Errorf("execution failed: %v", err)
	}
}

func TestProcessor_FailingProcessorShouldStopExecutionButEndEventsAreDelivered(t *testing.T) {
	ctrl := gomock.NewController(t)
	substate := NewMockSubstateProvider(ctrl)
	processor := NewMockProcessor(ctrl)
	extension := NewMockExtension(ctrl)

	substate.EXPECT().
		Run(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(from int, to int, consume Consumer) error {
			for i := from; i < to; i++ {
				if err := consume(TransactionInfo{i, 7, nil}); err != nil {
					return err
				}
			}
			return nil
		})

	stop := fmt.Errorf("stop!")
	gomock.InOrder(
		extension.EXPECT().PreRun(AtBlock(10), gomock.Any()),
		extension.EXPECT().PreBlock(AtBlock(10), gomock.Any()),
		extension.EXPECT().PreTransaction(AtTransaction(10, 7), gomock.Any()),
		processor.EXPECT().Process(AtTransaction(10, 7), gomock.Any()).Return(stop),
		extension.EXPECT().PostRun(AtTransaction(10, 7), gomock.Any(), stop),
	)

	executor := NewExecutor(substate)
	if got, want := executor.Run(Params{From: 10, To: 20}, processor, []Extension{extension}), stop; !errors.Is(got, want) {
		t.Errorf("execution did not fail as expected, wanted %v, got %v", want, got)
	}
}

func TestProcessor_EmptyIntervalEmitsNoEvents(t *testing.T) {
	ctrl := gomock.NewController(t)
	substate := NewMockSubstateProvider(ctrl)
	processor := NewMockProcessor(ctrl)
	extension := NewMockExtension(ctrl)

	substate.EXPECT().Run(10, 10, gomock.Any()).Return(nil)

	gomock.InOrder(
		extension.EXPECT().PreRun(AtBlock(10), gomock.Any()),
		extension.EXPECT().PostRun(AtBlock(10), gomock.Any(), nil),
	)

	executor := NewExecutor(substate)
	if err := executor.Run(Params{From: 10, To: 10}, processor, []Extension{extension}); err != nil {
		t.Errorf("execution failed: %v", err)
	}
}

func TestProcessor_MultipleExtensionsGetSignaledInOrder(t *testing.T) {
	ctrl := gomock.NewController(t)
	substate := NewMockSubstateProvider(ctrl)
	processor := NewMockProcessor(ctrl)
	extension1 := NewMockExtension(ctrl)
	extension2 := NewMockExtension(ctrl)

	substate.EXPECT().
		Run(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(from int, to int, consume Consumer) error {
			// We simulate two transactions per block.
			for i := from; i < to; i++ {
				consume(TransactionInfo{i, 7, nil})
				consume(TransactionInfo{i, 9, nil})
			}
			return nil
		})

	gomock.InOrder(
		extension1.EXPECT().PreRun(AtBlock(10), gomock.Any()),
		extension2.EXPECT().PreRun(AtBlock(10), gomock.Any()),

		extension1.EXPECT().PreBlock(AtBlock(10), gomock.Any()),
		extension2.EXPECT().PreBlock(AtBlock(10), gomock.Any()),

		extension1.EXPECT().PreTransaction(AtTransaction(10, 7), gomock.Any()),
		extension2.EXPECT().PreTransaction(AtTransaction(10, 7), gomock.Any()),
		processor.EXPECT().Process(AtTransaction(10, 7), gomock.Any()),
		extension2.EXPECT().PostTransaction(AtTransaction(10, 7), gomock.Any()),
		extension1.EXPECT().PostTransaction(AtTransaction(10, 7), gomock.Any()),

		extension1.EXPECT().PreTransaction(AtTransaction(10, 9), gomock.Any()),
		extension2.EXPECT().PreTransaction(AtTransaction(10, 9), gomock.Any()),
		processor.EXPECT().Process(AtTransaction(10, 9), gomock.Any()),
		extension2.EXPECT().PostTransaction(AtTransaction(10, 9), gomock.Any()),
		extension1.EXPECT().PostTransaction(AtTransaction(10, 9), gomock.Any()),

		extension2.EXPECT().PostBlock(AtBlock(10), gomock.Any()),
		extension1.EXPECT().PostBlock(AtBlock(10), gomock.Any()),

		extension2.EXPECT().PostRun(AtBlock(11), gomock.Any(), nil),
		extension1.EXPECT().PostRun(AtBlock(11), gomock.Any(), nil),
	)

	executor := NewExecutor(substate)
	if err := executor.Run(Params{From: 10, To: 11}, processor, []Extension{extension1, extension2}); err != nil {
		t.Errorf("execution failed: %v", err)
	}
}

func TestProcessor_FailingExtensionPreEventCausesExecutionToStop(t *testing.T) {
	ctrl := gomock.NewController(t)
	substate := NewMockSubstateProvider(ctrl)
	processor := NewMockProcessor(ctrl)
	extension1 := NewMockExtension(ctrl)
	extension2 := NewMockExtension(ctrl)

	stop := fmt.Errorf("stop!")
	resultError := errors.Join(stop)
	gomock.InOrder(
		extension1.EXPECT().PreRun(AtBlock(10), gomock.Any()).Return(stop),
		extension2.EXPECT().PreRun(AtBlock(10), gomock.Any()),

		extension2.EXPECT().PostRun(AtBlock(10), gomock.Any(), resultError),
		extension1.EXPECT().PostRun(AtBlock(10), gomock.Any(), resultError),
	)

	executor := NewExecutor(substate)
	if got, want := executor.Run(Params{From: 10, To: 20}, processor, []Extension{extension1, extension2}), resultError; errors.Is(got, want) {
		t.Errorf("execution failed with wrong error, wanted %v, got %v", want, got)
	}
}

func TestProcessor_FailingExtensionPostEventCausesExecutionToStop(t *testing.T) {
	ctrl := gomock.NewController(t)
	substate := NewMockSubstateProvider(ctrl)
	processor := NewMockProcessor(ctrl)
	extension1 := NewMockExtension(ctrl)
	extension2 := NewMockExtension(ctrl)

	substate.EXPECT().
		Run(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(from int, to int, consume Consumer) error {
			for i := from; i < to; i++ {
				if err := consume(TransactionInfo{i, 7, nil}); err != nil {
					return err
				}
			}
			return nil
		})

	processor.EXPECT().Process(gomock.Any(), gomock.Any()).AnyTimes()

	stop := fmt.Errorf("stop!")
	resultError := errors.Join(stop)
	gomock.InOrder(
		extension1.EXPECT().PreRun(AtBlock(10), gomock.Any()),
		extension2.EXPECT().PreRun(AtBlock(10), gomock.Any()),

		extension1.EXPECT().PreBlock(AtBlock(10), gomock.Any()),
		extension2.EXPECT().PreBlock(AtBlock(10), gomock.Any()),

		extension1.EXPECT().PreTransaction(AtBlock(10), gomock.Any()),
		extension2.EXPECT().PreTransaction(AtBlock(10), gomock.Any()),

		extension2.EXPECT().PostTransaction(AtBlock(10), gomock.Any()).Return(stop),
		extension1.EXPECT().PostTransaction(AtBlock(10), gomock.Any()),

		extension2.EXPECT().PostRun(AtBlock(10), gomock.Any(), resultError),
		extension1.EXPECT().PostRun(AtBlock(10), gomock.Any(), resultError),
	)

	executor := NewExecutor(substate)
	if got, want := executor.Run(Params{From: 10, To: 20}, processor, []Extension{extension1, extension2}), resultError; errors.Is(got, want) {
		t.Errorf("execution failed with wrong error, wanted %v, got %v", want, got)
	}
}

func TestProcessor_StateDbIsPropagatedToTheProcessorAndAllExtensions(t *testing.T) {
	ctrl := gomock.NewController(t)
	substate := NewMockSubstateProvider(ctrl)
	processor := NewMockProcessor(ctrl)
	extension := NewMockExtension(ctrl)
	state := state.NewMockStateDB(ctrl)

	substate.EXPECT().
		Run(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(from int, to int, consume Consumer) error {
			// We simulate two transactions per block.
			for i := from; i < to; i++ {
				consume(TransactionInfo{i, 7, nil})
				consume(TransactionInfo{i, 9, nil})
			}
			return nil
		})

	gomock.InOrder(
		extension.EXPECT().PreRun(gomock.Any(), WithState(state)),
		extension.EXPECT().PreBlock(gomock.Any(), WithState(state)),
		extension.EXPECT().PreTransaction(gomock.Any(), WithState(state)),
		processor.EXPECT().Process(gomock.Any(), WithState(state)),
		extension.EXPECT().PostTransaction(gomock.Any(), WithState(state)),
		extension.EXPECT().PreTransaction(gomock.Any(), WithState(state)),
		processor.EXPECT().Process(gomock.Any(), WithState(state)),
		extension.EXPECT().PostTransaction(gomock.Any(), WithState(state)),
		extension.EXPECT().PostBlock(gomock.Any(), WithState(state)),
		extension.EXPECT().PostRun(gomock.Any(), WithState(state), nil),
	)

	err := NewExecutor(substate).Run(
		Params{From: 10, To: 11, State: state},
		processor,
		[]Extension{extension},
	)
	if err != nil {
		t.Errorf("execution failed: %v", err)
	}
}

func TestProcessor_StateDbCanBeModifiedByExtensionsAndProcessorInSequentialRun(t *testing.T) {
	ctrl := gomock.NewController(t)
	substate := NewMockSubstateProvider(ctrl)
	processor := NewMockProcessor(ctrl)
	extension := NewMockExtension(ctrl)

	stateA := state.NewMockStateDB(ctrl)
	stateB := state.NewMockStateDB(ctrl)
	stateC := state.NewMockStateDB(ctrl)
	stateD := state.NewMockStateDB(ctrl)
	stateE := state.NewMockStateDB(ctrl)
	stateF := state.NewMockStateDB(ctrl)
	stateG := state.NewMockStateDB(ctrl)

	substate.EXPECT().
		Run(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(from int, to int, consume Consumer) error {
			consume(TransactionInfo{from, 7, nil})
			return nil
		})

	setState := func(state state.StateDB) func(State, *Context) {
		return func(_ State, c *Context) {
			c.State = state
		}
	}

	gomock.InOrder(
		extension.EXPECT().PreRun(gomock.Any(), WithState(stateA)).Do(setState(stateB)),
		extension.EXPECT().PreBlock(gomock.Any(), WithState(stateB)).Do(setState(stateC)),
		extension.EXPECT().PreTransaction(gomock.Any(), WithState(stateC)).Do(setState(stateD)),
		processor.EXPECT().Process(gomock.Any(), WithState(stateD)).Do(setState(stateE)),
		extension.EXPECT().PostTransaction(gomock.Any(), WithState(stateE)).Do(setState(stateF)),
		extension.EXPECT().PostBlock(gomock.Any(), WithState(stateF)).Do(setState(stateG)),
		extension.EXPECT().PostRun(gomock.Any(), WithState(stateG), nil),
	)

	err := NewExecutor(substate).Run(
		Params{State: stateA},
		processor,
		[]Extension{extension},
	)
	if err != nil {
		t.Errorf("execution failed: %v", err)
	}
}

func TestProcessor_StateDbCanBeModifiedByExtensionsAndProcessorInParallelRun(t *testing.T) {
	ctrl := gomock.NewController(t)
	substate := NewMockSubstateProvider(ctrl)
	processor := NewMockProcessor(ctrl)
	extension := NewMockExtension(ctrl)

	stateA := state.NewMockStateDB(ctrl)
	stateB := state.NewMockStateDB(ctrl)
	stateC := state.NewMockStateDB(ctrl)
	stateD := state.NewMockStateDB(ctrl)
	stateE := state.NewMockStateDB(ctrl)

	substate.EXPECT().
		Run(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(from int, to int, consume Consumer) error {
			consume(TransactionInfo{from, 7, nil})
			return nil
		})

	setState := func(state state.StateDB) func(State, *Context) {
		return func(_ State, c *Context) {
			c.State = state
		}
	}

	gomock.InOrder(
		extension.EXPECT().PreRun(gomock.Any(), WithState(stateA)).Do(setState(stateB)),
		extension.EXPECT().PreTransaction(gomock.Any(), WithState(stateB)).Do(setState(stateC)),
		processor.EXPECT().Process(gomock.Any(), WithState(stateC)).Do(setState(stateD)),
		extension.EXPECT().PostTransaction(gomock.Any(), WithState(stateD)).Do(setState(stateE)),
		// the context from a parallel execution is not merged back to the top-level context
		extension.EXPECT().PostRun(gomock.Any(), WithState(stateB), nil),
	)

	err := NewExecutor(substate).Run(
		Params{State: stateA, NumWorkers: 2},
		processor,
		[]Extension{extension},
	)
	if err != nil {
		t.Errorf("execution failed: %v", err)
	}
}

func TestProcessor_TransactionsAreProcessedWithMultipleWorkersIfRequested(t *testing.T) {
	ctrl := gomock.NewController(t)
	substate := NewMockSubstateProvider(ctrl)
	processor := NewMockProcessor(ctrl)

	substate.EXPECT().
		Run(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(from int, to int, consume Consumer) error {
			// We simulate two transactions per block.
			for i := from; i < to; i++ {
				consume(TransactionInfo{i, 7, nil})
				consume(TransactionInfo{i, 9, nil})
			}
			return nil
		})

	// Simulate two processors that need to be called in parallel.
	var wg sync.WaitGroup
	wg.Add(2)
	processor.EXPECT().Process(gomock.Any(), gomock.Any()).Times(2).Do(func(State, *Context) {
		wg.Done()
		wg.Wait()
	})

	err := NewExecutor(substate).Run(
		Params{From: 10, To: 11, NumWorkers: 2},
		processor,
		nil,
	)
	if err != nil {
		t.Errorf("execution failed: %v", err)
	}
}

func TestProcessor_SignalsAreDeliveredInConcurrentExecution(t *testing.T) {
	ctrl := gomock.NewController(t)
	substate := NewMockSubstateProvider(ctrl)
	processor := NewMockProcessor(ctrl)
	extension := NewMockExtension(ctrl)

	substate.EXPECT().
		Run(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(from int, to int, consume Consumer) error {
			// We simulate two transactions per block.
			for i := from; i < to; i++ {
				consume(TransactionInfo{i, 7, nil})
				consume(TransactionInfo{i, 9, nil})
			}
			return nil
		})

	// For each transaction, PreTransaction, Process, and PostTransaction
	// should happen in order. However, Transactions may be processed
	// out-of-order.
	// Note: In the parallel context there is no block boundary.
	pre := extension.EXPECT().PreRun(AtBlock(10), gomock.Any())
	post := extension.EXPECT().PostRun(AtBlock(12), gomock.Any(), nil)

	gomock.InOrder(
		pre,
		extension.EXPECT().PreTransaction(AtTransaction(10, 7), gomock.Any()),
		processor.EXPECT().Process(AtTransaction(10, 7), gomock.Any()),
		extension.EXPECT().PostTransaction(AtTransaction(10, 7), gomock.Any()),
		post,
	)

	gomock.InOrder(
		pre,
		extension.EXPECT().PreTransaction(AtTransaction(10, 9), gomock.Any()),
		processor.EXPECT().Process(AtTransaction(10, 9), gomock.Any()),
		extension.EXPECT().PostTransaction(AtTransaction(10, 9), gomock.Any()),
		post,
	)

	gomock.InOrder(
		pre,
		extension.EXPECT().PreTransaction(AtTransaction(11, 7), gomock.Any()),
		processor.EXPECT().Process(AtTransaction(11, 7), gomock.Any()),
		extension.EXPECT().PostTransaction(AtTransaction(11, 7), gomock.Any()),
		post,
	)

	gomock.InOrder(
		pre,
		extension.EXPECT().PreTransaction(AtTransaction(11, 9), gomock.Any()),
		processor.EXPECT().Process(AtTransaction(11, 9), gomock.Any()),
		extension.EXPECT().PostTransaction(AtTransaction(11, 9), gomock.Any()),
		post,
	)

	err := NewExecutor(substate).Run(
		Params{From: 10, To: 12, NumWorkers: 2},
		processor,
		[]Extension{extension},
	)
	if err != nil {
		t.Errorf("execution failed: %v", err)
	}
}

func TestProcessor_ProcessErrorAbortsParallelProcessing(t *testing.T) {
	ctrl := gomock.NewController(t)
	substate := NewMockSubstateProvider(ctrl)
	processor := NewMockProcessor(ctrl)

	substate.EXPECT().
		Run(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(from int, to int, consume Consumer) error {
			for i := from; i < to; i++ {
				if err := consume(TransactionInfo{i, 7, nil}); err != nil {
					return err
				}
			}
			return nil
		})

	stop := fmt.Errorf("stop!")
	processor.EXPECT().Process(AtBlock(4), gomock.Any()).Return(stop)
	processor.EXPECT().Process(gomock.Any(), gomock.Any()).MaxTimes(20)

	err := NewExecutor(substate).Run(
		Params{To: 1000, NumWorkers: 2},
		processor,
		nil,
	)
	if got, want := err, stop; !errors.Is(got, want) {
		t.Errorf("execution did not stop with correct error, wanted %v, got %v", want, got)
	}
}

func TestProcessor_PreEventErrorAbortsParallelProcessing(t *testing.T) {
	ctrl := gomock.NewController(t)
	substate := NewMockSubstateProvider(ctrl)
	processor := NewMockProcessor(ctrl)
	extension := NewMockExtension(ctrl)

	substate.EXPECT().
		Run(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(from int, to int, consume Consumer) error {
			for i := from; i < to; i++ {
				if err := consume(TransactionInfo{i, 7, nil}); err != nil {
					return err
				}
			}
			return nil
		})

	processor.EXPECT().Process(gomock.Any(), gomock.Any()).MaxTimes(20)

	stop := fmt.Errorf("stop!")
	extension.EXPECT().PreTransaction(AtBlock(4), gomock.Any()).Return(stop)

	extension.EXPECT().PreRun(gomock.Any(), gomock.Any())
	extension.EXPECT().PreTransaction(gomock.Any(), gomock.Any()).MaxTimes(20)
	extension.EXPECT().PostTransaction(gomock.Any(), gomock.Any()).MaxTimes(20)
	extension.EXPECT().PostRun(gomock.Any(), gomock.Any(), WithError(stop))

	err := NewExecutor(substate).Run(
		Params{To: 1000, NumWorkers: 2},
		processor,
		[]Extension{extension},
	)
	if got, want := err, stop; !errors.Is(got, want) {
		t.Errorf("execution did not stop with correct error, wanted %v, got %v", want, got)
	}
}

func TestProcessor_PostEventErrorAbortsParallelProcessing(t *testing.T) {
	ctrl := gomock.NewController(t)
	substate := NewMockSubstateProvider(ctrl)
	processor := NewMockProcessor(ctrl)
	extension := NewMockExtension(ctrl)

	substate.EXPECT().
		Run(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(from int, to int, consume Consumer) error {
			for i := from; i < to; i++ {
				if err := consume(TransactionInfo{i, 7, nil}); err != nil {
					return err
				}
			}
			return nil
		})

	processor.EXPECT().Process(gomock.Any(), gomock.Any()).MaxTimes(20)

	stop := fmt.Errorf("stop!")
	extension.EXPECT().PostTransaction(AtBlock(4), gomock.Any()).Return(stop)

	extension.EXPECT().PreRun(gomock.Any(), gomock.Any())
	extension.EXPECT().PreTransaction(gomock.Any(), gomock.Any()).MaxTimes(20)
	extension.EXPECT().PostTransaction(gomock.Any(), gomock.Any()).MaxTimes(20)
	extension.EXPECT().PostRun(gomock.Any(), gomock.Any(), WithError(stop))

	err := NewExecutor(substate).Run(
		Params{To: 1000, NumWorkers: 2},
		processor,
		[]Extension{extension},
	)
	if got, want := err, stop; !errors.Is(got, want) {
		t.Errorf("execution did not stop with correct error, wanted %v, got %v", want, got)
	}
}

func TestProcessor_SubstateIsPropagatedToTheProcessorAndAllExtensionsInSequentialExecution(t *testing.T) {
	ctrl := gomock.NewController(t)
	provider := NewMockSubstateProvider(ctrl)
	processor := NewMockProcessor(ctrl)
	extension := NewMockExtension(ctrl)

	substateA := &substate.Substate{}
	substateB := &substate.Substate{}

	provider.EXPECT().
		Run(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(from int, to int, consume Consumer) error {
			consume(TransactionInfo{from, 7, substateA})
			consume(TransactionInfo{from, 8, substateB})
			return nil
		})

	gomock.InOrder(
		extension.EXPECT().PreRun(WithSubstate(nil), gomock.Any()),
		extension.EXPECT().PreBlock(WithSubstate(nil), gomock.Any()),
		extension.EXPECT().PreTransaction(WithSubstate(substateA), gomock.Any()),
		processor.EXPECT().Process(WithSubstate(substateA), gomock.Any()),
		extension.EXPECT().PostTransaction(WithSubstate(substateA), gomock.Any()),
		extension.EXPECT().PreTransaction(WithSubstate(substateB), gomock.Any()),
		processor.EXPECT().Process(WithSubstate(substateB), gomock.Any()),
		extension.EXPECT().PostTransaction(WithSubstate(substateB), gomock.Any()),
		extension.EXPECT().PostBlock(WithSubstate(nil), gomock.Any()),
		extension.EXPECT().PostRun(WithSubstate(nil), gomock.Any(), nil),
	)

	err := NewExecutor(provider).Run(
		Params{From: 10, To: 11},
		processor,
		[]Extension{extension},
	)
	if err != nil {
		t.Errorf("execution failed: %v", err)
	}
}

func TestProcessor_SubstateIsPropagatedToTheProcessorAndAllExtensionsInParallelExecution(t *testing.T) {
	ctrl := gomock.NewController(t)
	provider := NewMockSubstateProvider(ctrl)
	processor := NewMockProcessor(ctrl)
	extension := NewMockExtension(ctrl)

	substateA := &substate.Substate{}
	substateB := &substate.Substate{}

	provider.EXPECT().
		Run(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(from int, to int, consume Consumer) error {
			consume(TransactionInfo{from, 7, substateA})
			consume(TransactionInfo{from, 8, substateB})
			return nil
		})

	pre := extension.EXPECT().PreRun(WithSubstate(nil), gomock.Any())
	post := extension.EXPECT().PostRun(WithSubstate(nil), gomock.Any(), nil)

	gomock.InOrder(
		pre,
		extension.EXPECT().PreTransaction(WithSubstate(substateA), gomock.Any()),
		processor.EXPECT().Process(WithSubstate(substateA), gomock.Any()),
		extension.EXPECT().PostTransaction(WithSubstate(substateA), gomock.Any()),
		post,
	)

	gomock.InOrder(
		pre,
		extension.EXPECT().PreTransaction(WithSubstate(substateB), gomock.Any()),
		processor.EXPECT().Process(WithSubstate(substateB), gomock.Any()),
		extension.EXPECT().PostTransaction(WithSubstate(substateB), gomock.Any()),
		post,
	)

	err := NewExecutor(provider).Run(
		Params{From: 10, To: 11, NumWorkers: 2},
		processor,
		[]Extension{extension},
	)
	if err != nil {
		t.Errorf("execution failed: %v", err)
	}
}

func TestProcessor_APanicInAnExecutorSkipsPostRunActions_InSequentialExecution(t *testing.T) {
	ctrl := gomock.NewController(t)
	provider := NewMockSubstateProvider(ctrl)
	processor := NewMockProcessor(ctrl)
	extension := NewMockExtension(ctrl)

	provider.EXPECT().
		Run(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(from int, to int, consume Consumer) error {
			return consume(TransactionInfo{Block: from, Transaction: 7})
		})

	extension.EXPECT().PreRun(gomock.Any(), gomock.Any())
	extension.EXPECT().PreBlock(gomock.Any(), gomock.Any())
	extension.EXPECT().PreTransaction(gomock.Any(), gomock.Any())

	stop := "stop"
	processor.EXPECT().Process(gomock.Any(), gomock.Any()).Do(func(any, any) {
		panic(stop)
	})

	panicReachedCaller := new(bool)
	t.Cleanup(func() {
		if !*panicReachedCaller {
			t.Errorf("expected panic did not reach top-level")
		}
	})
	defer func() {
		if r := recover(); r != nil {
			if r != stop {
				t.Errorf("unexpected panic, wanted %v, got %v", r, stop)
			}
			*panicReachedCaller = true
		}
	}()

	NewExecutor(provider).Run(
		Params{From: 10, To: 11},
		processor,
		[]Extension{extension},
	)
}

func TestProcessor_APanicInAnExecutorSkipsPostRunActions_InParallelExecution(t *testing.T) {
	ctrl := gomock.NewController(t)
	provider := NewMockSubstateProvider(ctrl)
	processor := NewMockProcessor(ctrl)
	extension := NewMockExtension(ctrl)

	provider.EXPECT().
		Run(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(from int, to int, consume Consumer) error {
			return consume(TransactionInfo{Block: from, Transaction: 7})
		})

	extension.EXPECT().PreRun(gomock.Any(), gomock.Any())
	extension.EXPECT().PreTransaction(gomock.Any(), gomock.Any())

	stop := "stop"
	processor.EXPECT().Process(gomock.Any(), gomock.Any()).Do(func(any, any) {
		panic(stop)
	})

	panicReachedCaller := new(bool)
	t.Cleanup(func() {
		if !*panicReachedCaller {
			t.Errorf("expected panic did not reach top-level")
		}
	})
	defer func() {
		if r := recover(); r != nil {
			if r != stop {
				t.Errorf("unexpected panic, wanted %v, got %v", r, stop)
			}
			*panicReachedCaller = true
		}
	}()

	NewExecutor(provider).Run(
		Params{From: 10, To: 11, NumWorkers: 2},
		processor,
		[]Extension{extension},
	)
}
