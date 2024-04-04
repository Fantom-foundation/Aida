package executor

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/state"
	"github.com/Fantom-foundation/Substate/substate"
	"go.uber.org/mock/gomock"
)

func TestProcessor_ProcessorGetsCalledForEachTransaction_TransactionLevelParallelism(t *testing.T) {
	ctrl := gomock.NewController(t)
	ss := NewMockProvider[any](ctrl)
	processor := NewMockProcessor[any](ctrl)

	ss.EXPECT().
		Run(10, 12, gomock.Any()).
		DoAndReturn(func(from int, to int, consume Consumer[any]) error {
			// We simulate two transactions per block.
			for i := from; i < to; i++ {
				consume(TransactionInfo[any]{i, 0, nil})
				consume(TransactionInfo[any]{i, 1, nil})
			}
			return nil
		})

	gomock.InOrder(
		processor.EXPECT().Process(AtTransaction[any](10, 0), gomock.Any()),
		processor.EXPECT().Process(AtTransaction[any](10, 1), gomock.Any()),
		processor.EXPECT().Process(AtTransaction[any](11, 0), gomock.Any()),
		processor.EXPECT().Process(AtTransaction[any](11, 1), gomock.Any()),
	)

	executor := NewExecutor[any](ss, "DEBUG")
	if err := executor.Run(Params{From: 10, To: 12, ParallelismGranularity: TransactionLevel}, processor, nil); err != nil {
		t.Errorf("execution failed: %v", err)
	}
}

func TestProcessor_ProcessorGetsCalledForEachTransaction_BlockLevelParallelism(t *testing.T) {
	ctrl := gomock.NewController(t)
	ss := NewMockProvider[any](ctrl)
	processor := NewMockProcessor[any](ctrl)

	ss.EXPECT().
		Run(10, 12, gomock.Any()).
		DoAndReturn(func(from int, to int, consume Consumer[any]) error {
			// We simulate two transactions per block.
			for i := from; i < to; i++ {
				consume(TransactionInfo[any]{i, 0, nil})
				consume(TransactionInfo[any]{i, 1, nil})
			}
			return nil
		})

	gomock.InOrder(
		processor.EXPECT().Process(AtTransaction[any](10, 0), gomock.Any()),
		processor.EXPECT().Process(AtTransaction[any](10, 1), gomock.Any()),
		processor.EXPECT().Process(AtTransaction[any](11, 0), gomock.Any()),
		processor.EXPECT().Process(AtTransaction[any](11, 1), gomock.Any()),
	)

	executor := NewExecutor[any](ss, "DEBUG")
	if err := executor.Run(Params{From: 10, To: 12, ParallelismGranularity: BlockLevel}, processor, nil); err != nil {
		t.Errorf("execution failed: %v", err)
	}
}

func TestProcessor_FailingProcessorStopsExecution_TransactionLevelParallelism(t *testing.T) {
	ctrl := gomock.NewController(t)
	substate := NewMockProvider[any](ctrl)
	processor := NewMockProcessor[any](ctrl)

	substate.EXPECT().
		Run(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(from int, to int, consume Consumer[any]) error {
			for i := from; i < to; i++ {
				if err := consume(TransactionInfo[any]{i, 0, nil}); err != nil {
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

	executor := NewExecutor[any](substate, "DEBUG")
	if got, want := executor.Run(Params{From: 10, To: 20, ParallelismGranularity: TransactionLevel}, processor, nil), stop; !errors.Is(got, want) {
		t.Errorf("execution did not produce expected error, wanted %v, got %v", got, want)
	}
}

func TestProcessor_FailingProcessorStopsExecution_BlockLevelParallelism(t *testing.T) {
	ctrl := gomock.NewController(t)
	substate := NewMockProvider[any](ctrl)
	processor := NewMockProcessor[any](ctrl)

	substate.EXPECT().
		Run(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(from int, to int, consume Consumer[any]) error {
			for i := from; i < to; i++ {
				if err := consume(TransactionInfo[any]{i, 0, nil}); err != nil {
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

	executor := NewExecutor[any](substate, "DEBUG")
	if got, want := executor.Run(Params{From: 10, To: 20, ParallelismGranularity: BlockLevel}, processor, nil), stop; !errors.Is(got, want) {
		t.Errorf("execution did not produce expected error, wanted %v, got %v", got, want)
	}
}

func TestProcessor_ExtensionsGetSignaledAboutEvents_TransactionLevelParallelism(t *testing.T) {
	ctrl := gomock.NewController(t)
	substate := NewMockProvider[any](ctrl)
	processor := NewMockProcessor[any](ctrl)
	extension := NewMockExtension[any](ctrl)

	substate.EXPECT().
		Run(10, 12, gomock.Any()).
		DoAndReturn(func(from int, to int, consume Consumer[any]) error {
			// We simulate two transactions per block.
			for i := from; i < to; i++ {
				consume(TransactionInfo[any]{i, 7, nil})
				consume(TransactionInfo[any]{i, 9, nil})
			}
			return nil
		})

	gomock.InOrder(
		extension.EXPECT().PreRun(AtBlock[any](10), gomock.Any()),

		extension.EXPECT().PreTransaction(AtTransaction[any](10, 7), gomock.Any()),
		processor.EXPECT().Process(AtTransaction[any](10, 7), gomock.Any()),
		extension.EXPECT().PostTransaction(AtTransaction[any](10, 7), gomock.Any()),
		extension.EXPECT().PreTransaction(AtTransaction[any](10, 9), gomock.Any()),
		processor.EXPECT().Process(AtTransaction[any](10, 9), gomock.Any()),
		extension.EXPECT().PostTransaction(AtTransaction[any](10, 9), gomock.Any()),

		extension.EXPECT().PreTransaction(AtTransaction[any](11, 7), gomock.Any()),
		processor.EXPECT().Process(AtTransaction[any](11, 7), gomock.Any()),
		extension.EXPECT().PostTransaction(AtTransaction[any](11, 7), gomock.Any()),
		extension.EXPECT().PreTransaction(AtTransaction[any](11, 9), gomock.Any()),
		processor.EXPECT().Process(AtTransaction[any](11, 9), gomock.Any()),
		extension.EXPECT().PostTransaction(AtTransaction[any](11, 9), gomock.Any()),

		extension.EXPECT().PostRun(AtBlock[any](12), gomock.Any(), nil),
	)

	executor := NewExecutor[any](substate, "DEBUG")
	if err := executor.Run(Params{From: 10, To: 12, ParallelismGranularity: TransactionLevel}, processor, []Extension[any]{extension}); err != nil {
		t.Errorf("execution failed: %v", err)
	}
}

func TestProcessor_ExtensionsGetSignaledAboutEvents_BlockLevelParallelism(t *testing.T) {
	ctrl := gomock.NewController(t)
	substate := NewMockProvider[any](ctrl)
	processor := NewMockProcessor[any](ctrl)
	extension := NewMockExtension[any](ctrl)

	substate.EXPECT().
		Run(10, 12, gomock.Any()).
		DoAndReturn(func(from int, to int, consume Consumer[any]) error {
			// We simulate two transactions per block.
			for i := from; i < to; i++ {
				consume(TransactionInfo[any]{i, 7, nil})
				consume(TransactionInfo[any]{i, 9, nil})
			}
			return nil
		})

	gomock.InOrder(
		extension.EXPECT().PreRun(AtBlock[any](10), gomock.Any()),

		extension.EXPECT().PreBlock(AtBlock[any](10), gomock.Any()),
		extension.EXPECT().PreTransaction(AtTransaction[any](10, 7), gomock.Any()),
		processor.EXPECT().Process(AtTransaction[any](10, 7), gomock.Any()),
		extension.EXPECT().PostTransaction(AtTransaction[any](10, 7), gomock.Any()),
		extension.EXPECT().PreTransaction(AtTransaction[any](10, 9), gomock.Any()),
		processor.EXPECT().Process(AtTransaction[any](10, 9), gomock.Any()),
		extension.EXPECT().PostTransaction(AtTransaction[any](10, 9), gomock.Any()),
		extension.EXPECT().PostBlock(AtTransaction[any](10, 9), gomock.Any()),

		extension.EXPECT().PreBlock(AtBlock[any](11), gomock.Any()),
		extension.EXPECT().PreTransaction(AtTransaction[any](11, 7), gomock.Any()),
		processor.EXPECT().Process(AtTransaction[any](11, 7), gomock.Any()),
		extension.EXPECT().PostTransaction(AtTransaction[any](11, 7), gomock.Any()),
		extension.EXPECT().PreTransaction(AtTransaction[any](11, 9), gomock.Any()),
		processor.EXPECT().Process(AtTransaction[any](11, 9), gomock.Any()),
		extension.EXPECT().PostTransaction(AtTransaction[any](11, 9), gomock.Any()),
		extension.EXPECT().PostBlock(AtTransaction[any](11, 9), gomock.Any()),

		extension.EXPECT().PostRun(AtBlock[any](12), gomock.Any(), nil),
	)

	executor := NewExecutor[any](substate, "DEBUG")
	if err := executor.Run(Params{From: 10, To: 12, ParallelismGranularity: BlockLevel}, processor, []Extension[any]{extension}); err != nil {
		t.Errorf("execution failed: %v", err)
	}
}

func TestProcessor_FailingProcessorShouldStopExecutionButEndEventsAreDelivered_TransactionLevelParallelism(t *testing.T) {
	ctrl := gomock.NewController(t)
	substate := NewMockProvider[any](ctrl)
	processor := NewMockProcessor[any](ctrl)
	extension := NewMockExtension[any](ctrl)

	substate.EXPECT().
		Run(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(from int, to int, consume Consumer[any]) error {
			for i := from; i < to; i++ {
				if err := consume(TransactionInfo[any]{i, 7, nil}); err != nil {
					return err
				}
			}
			return nil
		})

	stop := fmt.Errorf("stop!")
	gomock.InOrder(
		extension.EXPECT().PreRun(AtBlock[any](10), gomock.Any()),
		extension.EXPECT().PreTransaction(AtTransaction[any](10, 7), gomock.Any()),
		processor.EXPECT().Process(AtTransaction[any](10, 7), gomock.Any()).Return(stop),
		extension.EXPECT().PostRun(AtBlock[any](10), gomock.Any(), WithError(stop)),
	)

	executor := NewExecutor[any](substate, "DEBUG")
	if got, want := executor.Run(Params{From: 10, To: 20, ParallelismGranularity: TransactionLevel}, processor, []Extension[any]{extension}), stop; !errors.Is(got, want) {
		t.Errorf("execution did not fail as expected, wanted %v, got %v", want, got)
	}
}

func TestProcessor_FailingProcessorShouldStopExecutionButEndEventsAreDelivered_BlockLevelParallelism(t *testing.T) {
	ctrl := gomock.NewController(t)
	substate := NewMockProvider[any](ctrl)
	processor := NewMockProcessor[any](ctrl)
	extension := NewMockExtension[any](ctrl)

	substate.EXPECT().
		Run(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(from int, to int, consume Consumer[any]) error {
			for i := from; i < to; i++ {
				if err := consume(TransactionInfo[any]{i, 7, nil}); err != nil {
					return err
				}
			}
			return nil
		})

	stop := fmt.Errorf("stop!")
	gomock.InOrder(
		extension.EXPECT().PreRun(AtBlock[any](10), gomock.Any()),
		extension.EXPECT().PreBlock(AtBlock[any](10), gomock.Any()),
		extension.EXPECT().PreTransaction(AtTransaction[any](10, 7), gomock.Any()),
		processor.EXPECT().Process(AtTransaction[any](10, 7), gomock.Any()).Return(stop),
		extension.EXPECT().PostRun(AtBlock[any](10), gomock.Any(), WithError(stop)),
	)

	executor := NewExecutor[any](substate, "DEBUG")
	if got, want := executor.Run(Params{From: 10, To: 20, ParallelismGranularity: BlockLevel}, processor, []Extension[any]{extension}), stop; !errors.Is(got, want) {
		t.Errorf("execution did not fail as expected, wanted %v, got %v", want, got)
	}
}

func TestProcessor_EmptyIntervalEmitsNoEvents_TransactionLevelParallelism(t *testing.T) {
	ctrl := gomock.NewController(t)
	substate := NewMockProvider[any](ctrl)
	processor := NewMockProcessor[any](ctrl)
	extension := NewMockExtension[any](ctrl)

	substate.EXPECT().Run(10, 10, gomock.Any()).Return(nil)

	gomock.InOrder(
		extension.EXPECT().PreRun(AtBlock[any](10), gomock.Any()),
		extension.EXPECT().PostRun(AtBlock[any](10), gomock.Any(), nil),
	)

	executor := NewExecutor[any](substate, "DEBUG")
	if err := executor.Run(Params{From: 10, To: 10, ParallelismGranularity: TransactionLevel}, processor, []Extension[any]{extension}); err != nil {
		t.Errorf("execution failed: %v", err)
	}
}

func TestProcessor_EmptyIntervalEmitsNoEvents_BlockLevelParallelism(t *testing.T) {
	ctrl := gomock.NewController(t)
	substate := NewMockProvider[any](ctrl)
	processor := NewMockProcessor[any](ctrl)
	extension := NewMockExtension[any](ctrl)

	substate.EXPECT().Run(10, 10, gomock.Any()).Return(nil)

	gomock.InOrder(
		extension.EXPECT().PreRun(AtBlock[any](10), gomock.Any()),
		extension.EXPECT().PostRun(AtBlock[any](10), gomock.Any(), nil),
	)

	executor := NewExecutor[any](substate, "DEBUG")
	if err := executor.Run(Params{From: 10, To: 10, ParallelismGranularity: BlockLevel}, processor, []Extension[any]{extension}); err != nil {
		t.Errorf("execution failed: %v", err)
	}
}

func TestProcessor_MultipleExtensionsGetSignaledInOrder_TransactionLevelParallelism(t *testing.T) {
	ctrl := gomock.NewController(t)
	substate := NewMockProvider[any](ctrl)
	processor := NewMockProcessor[any](ctrl)
	extension1 := NewMockExtension[any](ctrl)
	extension2 := NewMockExtension[any](ctrl)

	substate.EXPECT().
		Run(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(from int, to int, consume Consumer[any]) error {
			// We simulate two transactions per block.
			for i := from; i < to; i++ {
				consume(TransactionInfo[any]{i, 7, nil})
				consume(TransactionInfo[any]{i, 9, nil})
			}
			return nil
		})

	gomock.InOrder(
		extension1.EXPECT().PreRun(AtBlock[any](10), gomock.Any()),
		extension2.EXPECT().PreRun(AtBlock[any](10), gomock.Any()),

		extension1.EXPECT().PreTransaction(AtTransaction[any](10, 7), gomock.Any()),
		extension2.EXPECT().PreTransaction(AtTransaction[any](10, 7), gomock.Any()),
		processor.EXPECT().Process(AtTransaction[any](10, 7), gomock.Any()),
		extension2.EXPECT().PostTransaction(AtTransaction[any](10, 7), gomock.Any()),
		extension1.EXPECT().PostTransaction(AtTransaction[any](10, 7), gomock.Any()),

		extension1.EXPECT().PreTransaction(AtTransaction[any](10, 9), gomock.Any()),
		extension2.EXPECT().PreTransaction(AtTransaction[any](10, 9), gomock.Any()),
		processor.EXPECT().Process(AtTransaction[any](10, 9), gomock.Any()),
		extension2.EXPECT().PostTransaction(AtTransaction[any](10, 9), gomock.Any()),
		extension1.EXPECT().PostTransaction(AtTransaction[any](10, 9), gomock.Any()),

		extension2.EXPECT().PostRun(AtBlock[any](11), gomock.Any(), nil),
		extension1.EXPECT().PostRun(AtBlock[any](11), gomock.Any(), nil),
	)

	executor := NewExecutor[any](substate, "DEBUG")
	if err := executor.Run(Params{From: 10, To: 11, ParallelismGranularity: TransactionLevel}, processor, []Extension[any]{extension1, extension2}); err != nil {
		t.Errorf("execution failed: %v", err)
	}
}

func TestProcessor_MultipleExtensionsGetSignaledInOrder_BlockLevelParallelism(t *testing.T) {
	ctrl := gomock.NewController(t)
	substate := NewMockProvider[any](ctrl)
	processor := NewMockProcessor[any](ctrl)
	extension1 := NewMockExtension[any](ctrl)
	extension2 := NewMockExtension[any](ctrl)

	substate.EXPECT().
		Run(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(from int, to int, consume Consumer[any]) error {
			// We simulate two transactions per block.
			for i := from; i < to; i++ {
				consume(TransactionInfo[any]{i, 7, nil})
				consume(TransactionInfo[any]{i, 9, nil})
			}
			return nil
		})

	gomock.InOrder(
		extension1.EXPECT().PreRun(AtBlock[any](10), gomock.Any()),
		extension2.EXPECT().PreRun(AtBlock[any](10), gomock.Any()),

		extension1.EXPECT().PreBlock(AtBlock[any](10), gomock.Any()),
		extension2.EXPECT().PreBlock(AtBlock[any](10), gomock.Any()),

		extension1.EXPECT().PreTransaction(AtTransaction[any](10, 7), gomock.Any()),
		extension2.EXPECT().PreTransaction(AtTransaction[any](10, 7), gomock.Any()),
		processor.EXPECT().Process(AtTransaction[any](10, 7), gomock.Any()),
		extension2.EXPECT().PostTransaction(AtTransaction[any](10, 7), gomock.Any()),
		extension1.EXPECT().PostTransaction(AtTransaction[any](10, 7), gomock.Any()),

		extension1.EXPECT().PreTransaction(AtTransaction[any](10, 9), gomock.Any()),
		extension2.EXPECT().PreTransaction(AtTransaction[any](10, 9), gomock.Any()),
		processor.EXPECT().Process(AtTransaction[any](10, 9), gomock.Any()),
		extension2.EXPECT().PostTransaction(AtTransaction[any](10, 9), gomock.Any()),
		extension1.EXPECT().PostTransaction(AtTransaction[any](10, 9), gomock.Any()),

		extension2.EXPECT().PostBlock(AtBlock[any](10), gomock.Any()),
		extension1.EXPECT().PostBlock(AtBlock[any](10), gomock.Any()),

		extension2.EXPECT().PostRun(AtBlock[any](11), gomock.Any(), nil),
		extension1.EXPECT().PostRun(AtBlock[any](11), gomock.Any(), nil),
	)

	executor := NewExecutor[any](substate, "DEBUG")
	if err := executor.Run(Params{From: 10, To: 11, ParallelismGranularity: BlockLevel}, processor, []Extension[any]{extension1, extension2}); err != nil {
		t.Errorf("execution failed: %v", err)
	}
}

func TestProcessor_FailingExtensionPreEventCausesExecutionToStop_TransactionLevelParallelism(t *testing.T) {
	ctrl := gomock.NewController(t)
	substate := NewMockProvider[any](ctrl)
	processor := NewMockProcessor[any](ctrl)
	extension1 := NewMockExtension[any](ctrl)
	extension2 := NewMockExtension[any](ctrl)

	stop := fmt.Errorf("stop!")
	resultError := errors.Join(stop)
	gomock.InOrder(
		extension1.EXPECT().PreRun(AtBlock[any](10), gomock.Any()).Return(stop),
		extension2.EXPECT().PreRun(AtBlock[any](10), gomock.Any()),

		extension2.EXPECT().PostRun(AtBlock[any](10), gomock.Any(), resultError),
		extension1.EXPECT().PostRun(AtBlock[any](10), gomock.Any(), resultError),
	)

	executor := NewExecutor[any](substate, "DEBUG")
	if got, want := executor.Run(Params{From: 10, To: 20, ParallelismGranularity: TransactionLevel}, processor, []Extension[any]{extension1, extension2}), resultError; errors.Is(got, want) {
		t.Errorf("execution failed with wrong error, wanted %v, got %v", want, got)
	}
}

func TestProcessor_FailingExtensionPreEventCausesExecutionToStop_BlockLevelParallelism(t *testing.T) {
	ctrl := gomock.NewController(t)
	substate := NewMockProvider[any](ctrl)
	processor := NewMockProcessor[any](ctrl)
	extension1 := NewMockExtension[any](ctrl)
	extension2 := NewMockExtension[any](ctrl)

	stop := fmt.Errorf("stop!")
	resultError := errors.Join(stop)
	gomock.InOrder(
		extension1.EXPECT().PreRun(AtBlock[any](10), gomock.Any()).Return(stop),
		extension2.EXPECT().PreRun(AtBlock[any](10), gomock.Any()),

		extension2.EXPECT().PostRun(AtBlock[any](10), gomock.Any(), resultError),
		extension1.EXPECT().PostRun(AtBlock[any](10), gomock.Any(), resultError),
	)

	executor := NewExecutor[any](substate, "DEBUG")
	if got, want := executor.Run(Params{From: 10, To: 20, ParallelismGranularity: BlockLevel}, processor, []Extension[any]{extension1, extension2}), resultError; errors.Is(got, want) {
		t.Errorf("execution failed with wrong error, wanted %v, got %v", want, got)
	}
}

func TestProcessor_FailingExtensionPostEventCausesExecutionToStop_TransactionLevelParallelism(t *testing.T) {
	ctrl := gomock.NewController(t)
	substate := NewMockProvider[any](ctrl)
	processor := NewMockProcessor[any](ctrl)
	extension1 := NewMockExtension[any](ctrl)
	extension2 := NewMockExtension[any](ctrl)

	substate.EXPECT().
		Run(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(from int, to int, consume Consumer[any]) error {
			for i := from; i < to; i++ {
				if err := consume(TransactionInfo[any]{i, 7, nil}); err != nil {
					return err
				}
			}
			return nil
		})

	processor.EXPECT().Process(gomock.Any(), gomock.Any()).AnyTimes()

	stop := fmt.Errorf("stop!")
	gomock.InOrder(
		extension1.EXPECT().PreRun(AtBlock[any](10), gomock.Any()),
		extension2.EXPECT().PreRun(AtBlock[any](10), gomock.Any()),

		extension1.EXPECT().PreTransaction(AtBlock[any](10), gomock.Any()),
		extension2.EXPECT().PreTransaction(AtBlock[any](10), gomock.Any()),

		extension2.EXPECT().PostTransaction(AtBlock[any](10), gomock.Any()).Return(stop),
		extension1.EXPECT().PostTransaction(AtBlock[any](10), gomock.Any()),

		extension2.EXPECT().PostRun(AtBlock[any](10), gomock.Any(), WithError(stop)),
		extension1.EXPECT().PostRun(AtBlock[any](10), gomock.Any(), WithError(stop)),
	)

	executor := NewExecutor[any](substate, "DEBUG")
	if got, want := executor.Run(Params{From: 10, To: 20, ParallelismGranularity: TransactionLevel}, processor, []Extension[any]{extension1, extension2}), stop; strings.Compare(got.Error(), want.Error()) != 0 {
		t.Errorf("execution failed with wrong error, wanted %v, got %v", want, got)
	}
}

func TestProcessor_FailingExtensionPostEventCausesExecutionToStop_BlockLevelParallelism(t *testing.T) {
	ctrl := gomock.NewController(t)
	substate := NewMockProvider[any](ctrl)
	processor := NewMockProcessor[any](ctrl)
	extension1 := NewMockExtension[any](ctrl)
	extension2 := NewMockExtension[any](ctrl)

	substate.EXPECT().
		Run(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(from int, to int, consume Consumer[any]) error {
			for i := from; i < to; i++ {
				if err := consume(TransactionInfo[any]{i, 7, nil}); err != nil {
					return err
				}
			}
			return nil
		})

	processor.EXPECT().Process(gomock.Any(), gomock.Any()).AnyTimes()

	stop := fmt.Errorf("stop!")
	gomock.InOrder(
		extension1.EXPECT().PreRun(AtBlock[any](10), gomock.Any()),
		extension2.EXPECT().PreRun(AtBlock[any](10), gomock.Any()),

		extension1.EXPECT().PreBlock(AtBlock[any](10), gomock.Any()),
		extension2.EXPECT().PreBlock(AtBlock[any](10), gomock.Any()),

		extension1.EXPECT().PreTransaction(AtBlock[any](10), gomock.Any()),
		extension2.EXPECT().PreTransaction(AtBlock[any](10), gomock.Any()),

		extension2.EXPECT().PostTransaction(AtBlock[any](10), gomock.Any()).Return(stop),
		extension1.EXPECT().PostTransaction(AtBlock[any](10), gomock.Any()),

		extension2.EXPECT().PostRun(AtBlock[any](10), gomock.Any(), WithError(stop)),
		extension1.EXPECT().PostRun(AtBlock[any](10), gomock.Any(), WithError(stop)),
	)

	executor := NewExecutor[any](substate, "DEBUG")
	if got, want := executor.Run(Params{From: 10, To: 20, ParallelismGranularity: BlockLevel}, processor, []Extension[any]{extension1, extension2}), stop; strings.Compare(got.Error(), want.Error()) != 0 {
		t.Errorf("execution failed with wrong error, wanted %v, got %v", want, got)
	}
}

func TestProcessor_StateDbIsPropagatedToTheProcessorAndAllExtensions_TransactionLevelParallelism(t *testing.T) {
	ctrl := gomock.NewController(t)
	substate := NewMockProvider[any](ctrl)
	processor := NewMockProcessor[any](ctrl)
	extension := NewMockExtension[any](ctrl)
	state := state.NewMockStateDB(ctrl)

	substate.EXPECT().
		Run(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(from int, to int, consume Consumer[any]) error {
			// We simulate two transactions per block.
			for i := from; i < to; i++ {
				consume(TransactionInfo[any]{i, 7, nil})
				consume(TransactionInfo[any]{i, 9, nil})
			}
			return nil
		})

	gomock.InOrder(
		extension.EXPECT().PreRun(gomock.Any(), WithState(state)),
		extension.EXPECT().PreTransaction(gomock.Any(), WithState(state)),
		processor.EXPECT().Process(gomock.Any(), WithState(state)),
		extension.EXPECT().PostTransaction(gomock.Any(), WithState(state)),
		extension.EXPECT().PreTransaction(gomock.Any(), WithState(state)),
		processor.EXPECT().Process(gomock.Any(), WithState(state)),
		extension.EXPECT().PostTransaction(gomock.Any(), WithState(state)),
		extension.EXPECT().PostRun(gomock.Any(), WithState(state), nil),
	)

	err := NewExecutor[any](substate, "DEBUG").Run(
		Params{From: 10, To: 11, State: state, ParallelismGranularity: TransactionLevel},
		processor,
		[]Extension[any]{extension},
	)
	if err != nil {
		t.Errorf("execution failed: %v", err)
	}
}

func TestProcessor_StateDbIsPropagatedToTheProcessorAndAllExtensions_BlockLevelParallelism(t *testing.T) {
	ctrl := gomock.NewController(t)
	substate := NewMockProvider[any](ctrl)
	processor := NewMockProcessor[any](ctrl)
	extension := NewMockExtension[any](ctrl)
	state := state.NewMockStateDB(ctrl)

	substate.EXPECT().
		Run(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(from int, to int, consume Consumer[any]) error {
			// We simulate two transactions per block.
			for i := from; i < to; i++ {
				consume(TransactionInfo[any]{i, 7, nil})
				consume(TransactionInfo[any]{i, 9, nil})
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

	err := NewExecutor[any](substate, "DEBUG").Run(
		Params{From: 10, To: 11, State: state, ParallelismGranularity: BlockLevel},
		processor,
		[]Extension[any]{extension},
	)
	if err != nil {
		t.Errorf("execution failed: %v", err)
	}
}

func TestProcessor_StateDbCanBeModifiedByExtensionsAndProcessorInSequentialRun_TransactionLevelParallelism(t *testing.T) {
	ctrl := gomock.NewController(t)
	substate := NewMockProvider[any](ctrl)
	processor := NewMockProcessor[any](ctrl)
	extension := NewMockExtension[any](ctrl)

	stateA := state.NewMockStateDB(ctrl)
	stateB := state.NewMockStateDB(ctrl)
	stateC := state.NewMockStateDB(ctrl)
	stateD := state.NewMockStateDB(ctrl)
	stateE := state.NewMockStateDB(ctrl)

	substate.EXPECT().
		Run(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(from int, to int, consume Consumer[any]) error {
			consume(TransactionInfo[any]{from, 7, nil})
			return nil
		})

	setState := func(state state.StateDB) func(State[any], *Context) {
		return func(_ State[any], ctx *Context) {
			ctx.State = state
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

	err := NewExecutor[any](substate, "DEBUG").Run(
		Params{State: stateA, NumWorkers: 2, ParallelismGranularity: TransactionLevel},
		processor,
		[]Extension[any]{extension},
	)
	if err != nil {
		t.Errorf("execution failed: %v", err)
	}
}

func TestProcessor_StateDbCanBeModifiedByExtensionsAndProcessorInSequentialRun_BlockLevelParallelism(t *testing.T) {
	ctrl := gomock.NewController(t)
	substate := NewMockProvider[any](ctrl)
	processor := NewMockProcessor[any](ctrl)
	extension := NewMockExtension[any](ctrl)

	stateA := state.NewMockStateDB(ctrl)
	stateB := state.NewMockStateDB(ctrl)
	stateC := state.NewMockStateDB(ctrl)
	stateD := state.NewMockStateDB(ctrl)
	stateE := state.NewMockStateDB(ctrl)
	stateF := state.NewMockStateDB(ctrl)
	stateG := state.NewMockStateDB(ctrl)

	substate.EXPECT().
		Run(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(from int, to int, consume Consumer[any]) error {
			consume(TransactionInfo[any]{from, 7, nil})
			return nil
		})

	setState := func(state state.StateDB) func(State[any], *Context) {
		return func(_ State[any], ctx *Context) {
			ctx.State = state
		}
	}

	gomock.InOrder(
		extension.EXPECT().PreRun(gomock.Any(), WithState(stateA)).Do(setState(stateB)),
		extension.EXPECT().PreBlock(gomock.Any(), WithState(stateB)).Do(setState(stateC)),
		extension.EXPECT().PreTransaction(gomock.Any(), WithState(stateC)).Do(setState(stateD)),
		processor.EXPECT().Process(gomock.Any(), WithState(stateD)).Do(setState(stateE)),
		extension.EXPECT().PostTransaction(gomock.Any(), WithState(stateE)).Do(setState(stateF)),
		extension.EXPECT().PostBlock(gomock.Any(), WithState(stateF)).Do(setState(stateG)),
		// the context from a parallel execution is not merged back to the top-level context
		extension.EXPECT().PostRun(gomock.Any(), WithState(stateB), nil),
	)

	err := NewExecutor[any](substate, "DEBUG").Run(
		Params{State: stateA, NumWorkers: 2, ParallelismGranularity: BlockLevel},
		processor,
		[]Extension[any]{extension},
	)
	if err != nil {
		t.Errorf("execution failed: %v", err)
	}
}

func TestProcessor_TransactionsAreProcessedWithMultipleWorkersIfRequested_TransactionLevelParallelism(t *testing.T) {
	ctrl := gomock.NewController(t)
	substate := NewMockProvider[any](ctrl)
	processor := NewMockProcessor[any](ctrl)

	substate.EXPECT().
		Run(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(from int, to int, consume Consumer[any]) error {
			// We simulate two transactions per block.
			for i := from; i < to; i++ {
				consume(TransactionInfo[any]{i, 7, nil})
				consume(TransactionInfo[any]{i, 9, nil})
			}
			return nil
		})

	// Simulate two processors that need to be called in parallel.
	var wg sync.WaitGroup
	wg.Add(2)
	processor.EXPECT().Process(gomock.Any(), gomock.Any()).Times(2).Do(func(State[any], *Context) {
		wg.Done()
		wg.Wait()
	})

	err := NewExecutor[any](substate, "DEBUG").Run(
		Params{From: 10, To: 11, NumWorkers: 2, ParallelismGranularity: TransactionLevel},
		processor,
		nil,
	)
	if err != nil {
		t.Errorf("execution failed: %v", err)
	}
}

func TestProcessor_TransactionsAreProcessedWithMultipleWorkersIfRequested_BlockLevelParallelism(t *testing.T) {
	ctrl := gomock.NewController(t)
	substate := NewMockProvider[any](ctrl)
	processor := NewMockProcessor[any](ctrl)

	substate.EXPECT().
		Run(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(from int, to int, consume Consumer[any]) error {
			// We simulate two transactions per block.
			for i := from; i < to; i++ {
				consume(TransactionInfo[any]{i, 7, nil})
				consume(TransactionInfo[any]{i + 1, 9, nil})
			}
			return nil
		})

	// Simulate two processors that need to be called in parallel.
	var wg sync.WaitGroup
	wg.Add(2)
	processor.EXPECT().Process(gomock.Any(), gomock.Any()).Times(2).Do(func(State[any], *Context) {
		wg.Done()
		wg.Wait()
	})

	err := NewExecutor[any](substate, "DEBUG").Run(
		Params{From: 10, To: 11, NumWorkers: 2, ParallelismGranularity: BlockLevel},
		processor,
		nil,
	)
	if err != nil {
		t.Errorf("execution failed: %v", err)
	}
}

func TestProcessor_SignalsAreDeliveredInConcurrentExecution_TransactionLevelParallelism(t *testing.T) {
	ctrl := gomock.NewController(t)
	substate := NewMockProvider[any](ctrl)
	processor := NewMockProcessor[any](ctrl)
	extension := NewMockExtension[any](ctrl)

	substate.EXPECT().
		Run(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(from int, to int, consume Consumer[any]) error {
			// We simulate two transactions per block.
			for i := from; i < to; i++ {
				consume(TransactionInfo[any]{i, 7, nil})
				consume(TransactionInfo[any]{i, 9, nil})
			}
			return nil
		})

	// For each transaction, PreTransaction, Process, and PostTransaction
	// should happen in order. However, Transactions may be processed
	// out-of-order.
	// Note: In the transaction level parallel context there is no block boundary.
	pre := extension.EXPECT().PreRun(AtBlock[any](10), gomock.Any())
	post := extension.EXPECT().PostRun(AtBlock[any](12), gomock.Any(), nil)

	gomock.InOrder(
		pre,
		extension.EXPECT().PreTransaction(AtTransaction[any](10, 7), gomock.Any()),
		processor.EXPECT().Process(AtTransaction[any](10, 7), gomock.Any()),
		extension.EXPECT().PostTransaction(AtTransaction[any](10, 7), gomock.Any()),
		post,
	)

	gomock.InOrder(
		pre,
		extension.EXPECT().PreTransaction(AtTransaction[any](10, 9), gomock.Any()),
		processor.EXPECT().Process(AtTransaction[any](10, 9), gomock.Any()),
		extension.EXPECT().PostTransaction(AtTransaction[any](10, 9), gomock.Any()),
		post,
	)

	gomock.InOrder(
		pre,
		extension.EXPECT().PreTransaction(AtTransaction[any](11, 7), gomock.Any()),
		processor.EXPECT().Process(AtTransaction[any](11, 7), gomock.Any()),
		extension.EXPECT().PostTransaction(AtTransaction[any](11, 7), gomock.Any()),
		post,
	)

	gomock.InOrder(
		pre,
		extension.EXPECT().PreTransaction(AtTransaction[any](11, 9), gomock.Any()),
		processor.EXPECT().Process(AtTransaction[any](11, 9), gomock.Any()),
		extension.EXPECT().PostTransaction(AtTransaction[any](11, 9), gomock.Any()),
		post,
	)

	err := NewExecutor[any](substate, "DEBUG").Run(
		Params{From: 10, To: 12, NumWorkers: 2, ParallelismGranularity: TransactionLevel},
		processor,
		[]Extension[any]{extension},
	)
	if err != nil {
		t.Errorf("execution failed: %v", err)
	}
}

func TestProcessor_SignalsAreDeliveredInConcurrentExecution_BlockLevelParallelism(t *testing.T) {
	ctrl := gomock.NewController(t)
	substate := NewMockProvider[any](ctrl)
	processor := NewMockProcessor[any](ctrl)
	extension := NewMockExtension[any](ctrl)

	substate.EXPECT().
		Run(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(from int, to int, consume Consumer[any]) error {
			// We simulate two transactions per block.
			for i := from; i < to; i++ {
				consume(TransactionInfo[any]{i, 7, nil})
				consume(TransactionInfo[any]{i, 9, nil})
			}
			return nil
		})

	// For each block PreBlock and PostBlock should happen then,
	// for each transaction, PreTransaction, Process, PostTransaction
	// should happen in order. However, Transactions may be processed
	// out-of-order.
	// Note: In the transaction level parallel context there is no block boundary.
	pre := extension.EXPECT().PreRun(AtBlock[any](10), gomock.Any())
	post := extension.EXPECT().PostRun(AtBlock[any](12), gomock.Any(), nil)

	gomock.InOrder(
		pre,
		extension.EXPECT().PreBlock(AtBlock[any](10), gomock.Any()),
		extension.EXPECT().PreTransaction(AtTransaction[any](10, 7), gomock.Any()),
		processor.EXPECT().Process(AtTransaction[any](10, 7), gomock.Any()),
		extension.EXPECT().PostTransaction(AtTransaction[any](10, 7), gomock.Any()),
		extension.EXPECT().PreTransaction(AtTransaction[any](10, 9), gomock.Any()),
		processor.EXPECT().Process(AtTransaction[any](10, 9), gomock.Any()),
		extension.EXPECT().PostTransaction(AtTransaction[any](10, 9), gomock.Any()),
		extension.EXPECT().PostBlock(AtBlock[any](10), gomock.Any()),
		post,
	)

	gomock.InOrder(
		pre,
		extension.EXPECT().PreBlock(AtBlock[any](11), gomock.Any()),
		extension.EXPECT().PreTransaction(AtTransaction[any](11, 7), gomock.Any()),
		processor.EXPECT().Process(AtTransaction[any](11, 7), gomock.Any()),
		extension.EXPECT().PostTransaction(AtTransaction[any](11, 7), gomock.Any()),
		extension.EXPECT().PreTransaction(AtTransaction[any](11, 9), gomock.Any()),
		processor.EXPECT().Process(AtTransaction[any](11, 9), gomock.Any()),
		extension.EXPECT().PostTransaction(AtTransaction[any](11, 9), gomock.Any()),
		extension.EXPECT().PostBlock(AtBlock[any](11), gomock.Any()),
		post,
	)

	err := NewExecutor[any](substate, "DEBUG").Run(
		Params{From: 10, To: 12, NumWorkers: 2, ParallelismGranularity: BlockLevel},
		processor,
		[]Extension[any]{extension},
	)
	if err != nil {
		t.Errorf("execution failed: %v", err)
	}
}

func TestProcessor_ProcessErrorAbortsProcessing_TransactionLevelParallelism(t *testing.T) {
	ctrl := gomock.NewController(t)
	substate := NewMockProvider[any](ctrl)
	processor := NewMockProcessor[any](ctrl)

	substate.EXPECT().
		Run(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(from int, to int, consume Consumer[any]) error {
			for i := from; i < to; i++ {
				if err := consume(TransactionInfo[any]{i, 7, nil}); err != nil {
					return err
				}
			}
			return nil
		})

	stop := fmt.Errorf("stop!")
	processor.EXPECT().Process(AtBlock[any](4), gomock.Any()).Return(stop)
	processor.EXPECT().Process(gomock.Any(), gomock.Any()).MaxTimes(20)

	err := NewExecutor[any](substate, "DEBUG").Run(
		Params{To: 1000, NumWorkers: 2, ParallelismGranularity: TransactionLevel},
		processor,
		nil,
	)
	if got, want := err, stop; !errors.Is(got, want) {
		t.Errorf("execution did not stop with correct error, wanted %v, got %v", want, got)
	}
}

func TestProcessor_ProcessErrorAbortsProcessing_BlockLevelParallelism(t *testing.T) {
	ctrl := gomock.NewController(t)
	substate := NewMockProvider[any](ctrl)
	processor := NewMockProcessor[any](ctrl)

	substate.EXPECT().
		Run(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(from int, to int, consume Consumer[any]) error {
			for i := from; i < to; i++ {
				if err := consume(TransactionInfo[any]{i, 7, nil}); err != nil {
					return err
				}
			}
			return nil
		})

	stop := fmt.Errorf("stop!")
	processor.EXPECT().Process(AtBlock[any](4), gomock.Any()).Return(stop)
	processor.EXPECT().Process(gomock.Any(), gomock.Any()).MaxTimes(20)

	err := NewExecutor[any](substate, "DEBUG").Run(
		Params{To: 1000, NumWorkers: 2, ParallelismGranularity: BlockLevel},
		processor,
		nil,
	)
	if got, want := err, stop; !errors.Is(got, want) {
		t.Errorf("execution did not stop with correct error, wanted %v, got %v", want, got)
	}
}

func TestProcessor_PreEventErrorAbortsProcessing_TransactionLevelParallelism(t *testing.T) {
	ctrl := gomock.NewController(t)
	substate := NewMockProvider[any](ctrl)
	processor := NewMockProcessor[any](ctrl)
	extension := NewMockExtension[any](ctrl)

	substate.EXPECT().
		Run(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(from int, to int, consume Consumer[any]) error {
			for i := from; i < to; i++ {
				if err := consume(TransactionInfo[any]{i, 7, nil}); err != nil {
					return err
				}
			}
			return nil
		})

	processor.EXPECT().Process(gomock.Any(), gomock.Any()).MaxTimes(20)

	stop := fmt.Errorf("stop!")
	extension.EXPECT().PreTransaction(AtBlock[any](4), gomock.Any()).Return(stop)

	extension.EXPECT().PreRun(gomock.Any(), gomock.Any())
	extension.EXPECT().PreTransaction(gomock.Any(), gomock.Any()).MaxTimes(20)
	extension.EXPECT().PostTransaction(gomock.Any(), gomock.Any()).MaxTimes(20)
	extension.EXPECT().PostRun(gomock.Any(), gomock.Any(), WithError(stop))

	err := NewExecutor[any](substate, "DEBUG").Run(
		Params{To: 1000, NumWorkers: 2, ParallelismGranularity: TransactionLevel},
		processor,
		[]Extension[any]{extension},
	)
	if got, want := err, stop; !errors.Is(got, want) {
		t.Errorf("execution did not stop with correct error, wanted %v, got %v", want, got)
	}
}

func TestProcessor_PreEventErrorAbortsProcessing_BlockLevelParallelism(t *testing.T) {
	ctrl := gomock.NewController(t)
	substate := NewMockProvider[any](ctrl)
	processor := NewMockProcessor[any](ctrl)
	extension := NewMockExtension[any](ctrl)

	substate.EXPECT().
		Run(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(from int, to int, consume Consumer[any]) error {
			for i := from; i < to; i++ {
				if err := consume(TransactionInfo[any]{i, 7, nil}); err != nil {
					return err
				}
			}
			return nil
		})

	processor.EXPECT().Process(gomock.Any(), gomock.Any()).MaxTimes(20)

	stop := fmt.Errorf("stop!")
	extension.EXPECT().PreTransaction(AtBlock[any](4), gomock.Any()).Return(stop)

	extension.EXPECT().PreRun(gomock.Any(), gomock.Any())
	extension.EXPECT().PreBlock(gomock.Any(), gomock.Any()).MaxTimes(20)
	extension.EXPECT().PreTransaction(gomock.Any(), gomock.Any()).MaxTimes(20)
	extension.EXPECT().PostTransaction(gomock.Any(), gomock.Any()).MaxTimes(20)
	extension.EXPECT().PostBlock(gomock.Any(), gomock.Any()).MaxTimes(20)
	extension.EXPECT().PostRun(gomock.Any(), gomock.Any(), WithError(stop))

	err := NewExecutor[any](substate, "DEBUG").Run(
		Params{To: 1000, NumWorkers: 2, ParallelismGranularity: BlockLevel},
		processor,
		[]Extension[any]{extension},
	)
	if got, want := err, stop; !errors.Is(got, want) {
		t.Errorf("execution did not stop with correct error, wanted %v, got %v", want, got)
	}
}

func TestProcessor_PostEventErrorAbortsProcessing_TransactionLevelParallelism(t *testing.T) {
	ctrl := gomock.NewController(t)
	substate := NewMockProvider[any](ctrl)
	processor := NewMockProcessor[any](ctrl)
	extension := NewMockExtension[any](ctrl)

	substate.EXPECT().
		Run(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(from int, to int, consume Consumer[any]) error {
			for i := from; i < to; i++ {
				if err := consume(TransactionInfo[any]{i, 7, nil}); err != nil {
					return err
				}
			}
			return nil
		})

	processor.EXPECT().Process(gomock.Any(), gomock.Any()).MaxTimes(20)

	stop := fmt.Errorf("stop!")
	extension.EXPECT().PostTransaction(AtBlock[any](4), gomock.Any()).Return(stop)

	extension.EXPECT().PreRun(gomock.Any(), gomock.Any())
	extension.EXPECT().PreTransaction(gomock.Any(), gomock.Any()).MaxTimes(20)
	extension.EXPECT().PostTransaction(gomock.Any(), gomock.Any()).MaxTimes(20)
	extension.EXPECT().PostRun(gomock.Any(), gomock.Any(), WithError(stop))

	err := NewExecutor[any](substate, "DEBUG").Run(
		Params{To: 1000, NumWorkers: 2, ParallelismGranularity: TransactionLevel},
		processor,
		[]Extension[any]{extension},
	)
	if got, want := err, stop; !errors.Is(got, want) {
		t.Errorf("execution did not stop with correct error, wanted %v, got %v", want, got)
	}
}

func TestProcessor_PostEventErrorAbortsProcessing_BlockLevelParallelism(t *testing.T) {
	ctrl := gomock.NewController(t)
	substate := NewMockProvider[any](ctrl)
	processor := NewMockProcessor[any](ctrl)
	extension := NewMockExtension[any](ctrl)

	substate.EXPECT().
		Run(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(from int, to int, consume Consumer[any]) error {
			for i := from; i < to; i++ {
				if err := consume(TransactionInfo[any]{i, 7, nil}); err != nil {
					return err
				}
			}
			return nil
		})

	processor.EXPECT().Process(gomock.Any(), gomock.Any()).MaxTimes(20)

	stop := fmt.Errorf("stop!")
	extension.EXPECT().PostTransaction(AtBlock[any](4), gomock.Any()).Return(stop)

	extension.EXPECT().PreRun(gomock.Any(), gomock.Any())
	extension.EXPECT().PreBlock(gomock.Any(), gomock.Any()).MaxTimes(20)
	extension.EXPECT().PreTransaction(gomock.Any(), gomock.Any()).MaxTimes(20)
	extension.EXPECT().PostTransaction(gomock.Any(), gomock.Any()).MaxTimes(20)
	extension.EXPECT().PostBlock(gomock.Any(), gomock.Any()).MaxTimes(20)
	extension.EXPECT().PostRun(gomock.Any(), gomock.Any(), WithError(stop))

	err := NewExecutor[any](substate, "DEBUG").Run(
		Params{To: 1000, NumWorkers: 2, ParallelismGranularity: BlockLevel},
		processor,
		[]Extension[any]{extension},
	)
	if got, want := err, stop; !errors.Is(got, want) {
		t.Errorf("execution did not stop with correct error, wanted %v, got %v", want, got)
	}
}

func TestProcessor_SubstateIsPropagatedToTheProcessorAndAllExtensions_TransactionLevelParallelism(t *testing.T) {
	ctrl := gomock.NewController(t)
	provider := NewMockProvider[*substate.Substate](ctrl)
	processor := NewMockProcessor[*substate.Substate](ctrl)
	extension := NewMockExtension[*substate.Substate](ctrl)

	substateA := &substate.Substate{}
	substateB := &substate.Substate{}

	provider.EXPECT().
		Run(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(from int, to int, consume Consumer[*substate.Substate]) error {
			consume(TransactionInfo[*substate.Substate]{from, 7, substateA})
			consume(TransactionInfo[*substate.Substate]{from, 8, substateB})
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

	err := NewExecutor[*substate.Substate](provider, "DEBUG").Run(
		Params{From: 10, To: 11, NumWorkers: 2, ParallelismGranularity: TransactionLevel},
		processor,
		[]Extension[*substate.Substate]{extension},
	)
	if err != nil {
		t.Errorf("execution failed: %v", err)
	}
}

func TestProcessor_SubstateIsPropagatedToTheProcessorAndAllExtensions_BlockLevelParallelism(t *testing.T) {
	ctrl := gomock.NewController(t)
	provider := NewMockProvider[*substate.Substate](ctrl)
	processor := NewMockProcessor[*substate.Substate](ctrl)
	extension := NewMockExtension[*substate.Substate](ctrl)

	substateA := &substate.Substate{}
	substateB := &substate.Substate{}

	provider.EXPECT().
		Run(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(from int, to int, consume Consumer[*substate.Substate]) error {
			consume(TransactionInfo[*substate.Substate]{from, 7, substateA})
			consume(TransactionInfo[*substate.Substate]{from, 8, substateB})
			return nil
		})

	pre := extension.EXPECT().PreRun(WithSubstate(nil), gomock.Any())
	post := extension.EXPECT().PostRun(WithSubstate(nil), gomock.Any(), nil)

	gomock.InOrder(
		pre,
		extension.EXPECT().PreBlock(WithSubstate(substateA), gomock.Any()),
		extension.EXPECT().PreTransaction(WithSubstate(substateA), gomock.Any()),
		processor.EXPECT().Process(WithSubstate(substateA), gomock.Any()),
		extension.EXPECT().PostTransaction(WithSubstate(substateA), gomock.Any()),
		extension.EXPECT().PreTransaction(WithSubstate(substateB), gomock.Any()),
		processor.EXPECT().Process(WithSubstate(substateB), gomock.Any()),
		extension.EXPECT().PostTransaction(WithSubstate(substateB), gomock.Any()),
		extension.EXPECT().PostBlock(WithSubstate(substateB), gomock.Any()),
		post,
	)
	err := NewExecutor[*substate.Substate](provider, "DEBUG").Run(
		Params{From: 10, To: 11, NumWorkers: 2, ParallelismGranularity: BlockLevel},
		processor,
		[]Extension[*substate.Substate]{extension},
	)
	if err != nil {
		t.Errorf("execution failed: %v", err)
	}
}

func TestProcessor_APanicInAnExecutorSkipsPostRunActions_TransactionLevelParallelism(t *testing.T) {
	ctrl := gomock.NewController(t)
	provider := NewMockProvider[any](ctrl)
	processor := NewMockProcessor[any](ctrl)
	extension := NewMockExtension[any](ctrl)
	log := logger.NewMockLogger(ctrl)

	provider.EXPECT().
		Run(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(from int, to int, consume Consumer[any]) error {
			return consume(TransactionInfo[any]{Block: from, Transaction: 7})
		})

	log.EXPECT().Debugf(gomock.Any(), gomock.Any())
	extension.EXPECT().PreRun(gomock.Any(), gomock.Any())
	extension.EXPECT().PreTransaction(gomock.Any(), gomock.Any())

	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("program must panic")
		}
	}()

	stop := "stop"
	processor.EXPECT().Process(gomock.Any(), gomock.Any()).Do(func(any, any) {
		panic(stop)
	})

	newExecutor[any](provider, log).Run(
		Params{From: 10, To: 11, NumWorkers: 2, ParallelismGranularity: TransactionLevel},
		processor,
		[]Extension[any]{extension},
	)
}

func TestProcessor_APanicInAnExecutorSkipsPostRunActions_BlockLevelParallelism(t *testing.T) {
	ctrl := gomock.NewController(t)
	provider := NewMockProvider[any](ctrl)
	processor := NewMockProcessor[any](ctrl)
	extension := NewMockExtension[any](ctrl)
	log := logger.NewMockLogger(ctrl)

	provider.EXPECT().
		Run(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(from int, to int, consume Consumer[any]) error {
			return consume(TransactionInfo[any]{Block: from, Transaction: 7})
		})

	log.EXPECT().Debugf(gomock.Any(), gomock.Any())
	extension.EXPECT().PreRun(gomock.Any(), gomock.Any())
	extension.EXPECT().PreBlock(gomock.Any(), gomock.Any())
	extension.EXPECT().PreTransaction(gomock.Any(), gomock.Any())

	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("program must panic")
		}
	}()

	stop := "stop"
	processor.EXPECT().Process(gomock.Any(), gomock.Any()).Do(func(any, any) {
		panic(stop)
	})

	newExecutor[any](provider, log).Run(
		Params{From: 10, To: 11, NumWorkers: 2, ParallelismGranularity: BlockLevel},
		processor,
		[]Extension[any]{extension},
	)
}

func TestProcessor_SingleBlockRun_BlockLevelParallelism(t *testing.T) {
	ctrl := gomock.NewController(t)
	substate := NewMockProvider[any](ctrl)
	processor := NewMockProcessor[any](ctrl)
	extension := NewMockExtension[any](ctrl)

	stateA := state.NewMockStateDB(ctrl)
	stateB := state.NewMockStateDB(ctrl)
	stateC := state.NewMockStateDB(ctrl)
	stateD := state.NewMockStateDB(ctrl)
	stateE := state.NewMockStateDB(ctrl)
	stateF := state.NewMockStateDB(ctrl)
	stateG := state.NewMockStateDB(ctrl)

	substate.EXPECT().
		Run(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(from int, to int, consume Consumer[any]) error {
			consume(TransactionInfo[any]{from, 7, nil})
			return nil
		})

	setState := func(state state.StateDB) func(State[any], *Context) {
		return func(_ State[any], ctx *Context) {
			ctx.State = state
		}
	}

	gomock.InOrder(
		extension.EXPECT().PreRun(gomock.Any(), WithState(stateA)).Do(setState(stateB)),
		extension.EXPECT().PreBlock(gomock.Any(), WithState(stateB)).Do(setState(stateC)),
		extension.EXPECT().PreTransaction(gomock.Any(), WithState(stateC)).Do(setState(stateD)),
		processor.EXPECT().Process(gomock.Any(), WithState(stateD)).Do(setState(stateE)),
		extension.EXPECT().PostTransaction(gomock.Any(), WithState(stateE)).Do(setState(stateF)),
		extension.EXPECT().PostBlock(gomock.Any(), WithState(stateF)).Do(setState(stateG)),
		// the context from a parallel block execution is not merged back to the top-level context
		extension.EXPECT().PostRun(gomock.Any(), WithState(stateB), nil),
	)

	err := NewExecutor[any](substate, "DEBUG").Run(
		Params{State: stateA, NumWorkers: 2, ParallelismGranularity: BlockLevel},
		processor,
		[]Extension[any]{extension},
	)
	if err != nil {
		t.Errorf("execution failed: %v", err)
	}
}

func TestProcessor_MultipleBlocksRun_BlockLevelParallelism(t *testing.T) {
	ctrl := gomock.NewController(t)
	substate := NewMockProvider[any](ctrl)
	processor := NewMockProcessor[any](ctrl)
	extension := NewMockExtension[any](ctrl)

	substate.EXPECT().
		Run(10, 12, gomock.Any()).
		DoAndReturn(func(from int, to int, consume Consumer[any]) error {
			// We simulate two transactions per block.
			consume(TransactionInfo[any]{10, 7, nil})
			consume(TransactionInfo[any]{10, 9, nil})
			consume(TransactionInfo[any]{11, 8, nil})
			consume(TransactionInfo[any]{11, 10, nil})
			return nil
		})

	extension.EXPECT().PreRun(AtBlock[any](10), gomock.Any())
	extension.EXPECT().PreBlock(AtBlock[any](10), gomock.Any())
	extension.EXPECT().PreTransaction(AtTransaction[any](10, 7), gomock.Any())
	processor.EXPECT().Process(AtTransaction[any](10, 7), gomock.Any())
	extension.EXPECT().PostTransaction(AtTransaction[any](10, 7), gomock.Any())
	extension.EXPECT().PreTransaction(AtTransaction[any](10, 9), gomock.Any())
	processor.EXPECT().Process(AtTransaction[any](10, 9), gomock.Any())
	extension.EXPECT().PostTransaction(AtTransaction[any](10, 9), gomock.Any())
	extension.EXPECT().PostBlock(AtTransaction[any](10, 9), gomock.Any())

	extension.EXPECT().PreBlock(AtBlock[any](11), gomock.Any())
	extension.EXPECT().PreTransaction(AtTransaction[any](11, 8), gomock.Any())
	processor.EXPECT().Process(AtTransaction[any](11, 8), gomock.Any())
	extension.EXPECT().PostTransaction(AtTransaction[any](11, 8), gomock.Any())
	extension.EXPECT().PreTransaction(AtTransaction[any](11, 10), gomock.Any())
	processor.EXPECT().Process(AtTransaction[any](11, 10), gomock.Any())
	extension.EXPECT().PostTransaction(AtTransaction[any](11, 10), gomock.Any())
	extension.EXPECT().PostBlock(AtTransaction[any](11, 10), gomock.Any())

	extension.EXPECT().PostRun(AtBlock[any](12), gomock.Any(), nil)

	executor := NewExecutor[any](substate, "DEBUG")
	if err := executor.Run(Params{From: 10, To: 12, NumWorkers: 2, ParallelismGranularity: BlockLevel},
		processor,
		[]Extension[any]{extension}); err != nil {
		t.Errorf("execution failed: %v", err)
	}
}

func TestProcessor_PanicCaughtInPreRunIsProperlyLogged_TransactionLevelParallelism(t *testing.T) {
	ctrl := gomock.NewController(t)
	substate := NewMockProvider[any](ctrl)
	processor := NewMockProcessor[any](ctrl)
	extension := NewMockExtension[any](ctrl)
	log := logger.NewMockLogger(ctrl)

	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("program must panic")
		}
	}()

	gomock.InOrder(
		extension.EXPECT().PreRun(gomock.Any(), gomock.Any()).Do(func(any, any) {
			panic("stop")
		}),
	)

	executor := newExecutor[any](substate, log)
	if err := executor.Run(Params{ParallelismGranularity: TransactionLevel},
		processor,
		[]Extension[any]{extension}); err != nil {
		t.Errorf("execution failed: %v", err)
	}
}

func TestProcessor_PanicCaughtInPreRunIsProperlyLogged_BlockLevelParallelism(t *testing.T) {
	ctrl := gomock.NewController(t)
	substate := NewMockProvider[any](ctrl)
	processor := NewMockProcessor[any](ctrl)
	extension := NewMockExtension[any](ctrl)
	log := logger.NewMockLogger(ctrl)

	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("program must panic")
		}
	}()

	gomock.InOrder(
		extension.EXPECT().PreRun(gomock.Any(), gomock.Any()).Do(func(any, any) {
			panic("stop")
		}),
	)

	executor := newExecutor[any](substate, log)
	if err := executor.Run(Params{ParallelismGranularity: BlockLevel},
		processor,
		[]Extension[any]{extension}); err != nil {
		t.Errorf("execution failed: %v", err)
	}
}

func TestProcessor_PanicCaughtInPreBlockIsProperlyLogged_BlockLevelParallelism(t *testing.T) {
	ctrl := gomock.NewController(t)
	substate := NewMockProvider[any](ctrl)
	processor := NewMockProcessor[any](ctrl)
	extension := NewMockExtension[any](ctrl)
	log := logger.NewMockLogger(ctrl)

	substate.EXPECT().
		Run(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(from int, to int, consume Consumer[any]) error {
			for i := from; i < to; i++ {
				if err := consume(TransactionInfo[any]{i, 0, nil}); err != nil {
					return err
				}
			}
			return nil
		})

	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("program must panic")
		}
	}()

	gomock.InOrder(
		extension.EXPECT().PreRun(gomock.Any(), gomock.Any()),
		log.EXPECT().Debugf(gomock.Any(), gomock.Any()),
		extension.EXPECT().PreBlock(gomock.Any(), gomock.Any()).Do(func(any, any) {
			panic("stop")
		}),
	)

	executor := newExecutor[any](substate, log)
	if err := executor.Run(Params{From: 1, To: 2, NumWorkers: 2, ParallelismGranularity: BlockLevel},
		processor,
		[]Extension[any]{extension}); err != nil {
		t.Errorf("execution failed: %v", err)
	}
}

func TestProcessor_PanicCaughtInPreTransactionIsProperlyLogged_TransactionLevelParallelism(t *testing.T) {
	ctrl := gomock.NewController(t)
	substate := NewMockProvider[any](ctrl)
	processor := NewMockProcessor[any](ctrl)
	extension := NewMockExtension[any](ctrl)
	log := logger.NewMockLogger(ctrl)

	substate.EXPECT().
		Run(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(from int, to int, consume Consumer[any]) error {
			for i := from; i < to; i++ {
				if err := consume(TransactionInfo[any]{i, 0, nil}); err != nil {
					return err
				}
			}
			return nil
		})

	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("program must panic")
		}
	}()

	gomock.InOrder(
		extension.EXPECT().PreRun(gomock.Any(), gomock.Any()),
		log.EXPECT().Debugf(gomock.Any(), gomock.Any()),
		extension.EXPECT().PreTransaction(gomock.Any(), gomock.Any()).Do(func(any, any) {
			panic("stop")
		}),
	)

	executor := newExecutor[any](substate, log)
	if err := executor.Run(Params{From: 1, To: 2, NumWorkers: 2, ParallelismGranularity: TransactionLevel},
		processor,
		[]Extension[any]{extension}); err != nil {
		t.Errorf("execution failed: %v", err)
	}
}

func TestProcessor_PanicCaughtInPreTransactionIsProperlyLogged_BlockLevelParallelism(t *testing.T) {
	ctrl := gomock.NewController(t)
	substate := NewMockProvider[any](ctrl)
	processor := NewMockProcessor[any](ctrl)
	extension := NewMockExtension[any](ctrl)
	log := logger.NewMockLogger(ctrl)

	substate.EXPECT().
		Run(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(from int, to int, consume Consumer[any]) error {
			for i := from; i < to; i++ {
				if err := consume(TransactionInfo[any]{i, 0, nil}); err != nil {
					return err
				}
			}
			return nil
		})

	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("program must panic")
		}
	}()

	gomock.InOrder(
		extension.EXPECT().PreRun(gomock.Any(), gomock.Any()),
		log.EXPECT().Debugf(gomock.Any(), gomock.Any()),
		extension.EXPECT().PreBlock(gomock.Any(), gomock.Any()),
		extension.EXPECT().PreTransaction(gomock.Any(), gomock.Any()).Do(func(any, any) {
			panic("stop")
		}),
	)

	executor := newExecutor[any](substate, log)
	if err := executor.Run(Params{From: 1, To: 2, NumWorkers: 2, ParallelismGranularity: BlockLevel},
		processor,
		[]Extension[any]{extension}); err != nil {
		t.Errorf("execution failed: %v", err)
	}
}

func TestProcessor_PanicCaughtInPostTransactionIsProperlyLogged_TransactionLevelParallelism(t *testing.T) {
	ctrl := gomock.NewController(t)
	substate := NewMockProvider[any](ctrl)
	processor := NewMockProcessor[any](ctrl)
	extension := NewMockExtension[any](ctrl)
	log := logger.NewMockLogger(ctrl)

	substate.EXPECT().
		Run(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(from int, to int, consume Consumer[any]) error {
			for i := from; i < to; i++ {
				if err := consume(TransactionInfo[any]{i, 0, nil}); err != nil {
					return err
				}
			}
			return nil
		})

	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("program must panic")
		}
	}()

	gomock.InOrder(
		extension.EXPECT().PreRun(gomock.Any(), gomock.Any()),
		log.EXPECT().Debugf(gomock.Any(), gomock.Any()),
		extension.EXPECT().PreTransaction(gomock.Any(), gomock.Any()),
		processor.EXPECT().Process(gomock.Any(), gomock.Any()),
		extension.EXPECT().PostTransaction(gomock.Any(), gomock.Any()).Do(func(any, any) {
			panic("stop")
		}),
	)

	executor := newExecutor[any](substate, log)
	if err := executor.Run(Params{From: 1, To: 2, NumWorkers: 2, ParallelismGranularity: TransactionLevel},
		processor,
		[]Extension[any]{extension}); err != nil {
		t.Errorf("execution failed: %v", err)
	}
}

func TestProcessor_PanicCaughtInPostTransactionIsProperlyLogged_BlockLevelParallelism(t *testing.T) {
	ctrl := gomock.NewController(t)
	substate := NewMockProvider[any](ctrl)
	processor := NewMockProcessor[any](ctrl)
	extension := NewMockExtension[any](ctrl)
	log := logger.NewMockLogger(ctrl)

	substate.EXPECT().
		Run(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(from int, to int, consume Consumer[any]) error {
			for i := from; i < to; i++ {
				if err := consume(TransactionInfo[any]{i, 0, nil}); err != nil {
					return err
				}
			}
			return nil
		})

	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("program must panic")
		}
	}()

	gomock.InOrder(
		extension.EXPECT().PreRun(gomock.Any(), gomock.Any()),
		log.EXPECT().Debugf(gomock.Any(), gomock.Any()),
		extension.EXPECT().PreBlock(gomock.Any(), gomock.Any()),
		extension.EXPECT().PreTransaction(gomock.Any(), gomock.Any()),
		processor.EXPECT().Process(gomock.Any(), gomock.Any()),
		extension.EXPECT().PostTransaction(gomock.Any(), gomock.Any()).Do(func(any, any) {
			panic("stop")
		}),
	)

	executor := newExecutor[any](substate, log)
	if err := executor.Run(Params{From: 1, To: 2, NumWorkers: 2, ParallelismGranularity: BlockLevel},
		processor,
		[]Extension[any]{extension}); err != nil {
		t.Errorf("execution failed: %v", err)
	}
}

func TestProcessor_PanicCaughtInPostBlockIsProperlyLogged_BlockLevelParallelism(t *testing.T) {
	ctrl := gomock.NewController(t)
	substate := NewMockProvider[any](ctrl)
	processor := NewMockProcessor[any](ctrl)
	extension := NewMockExtension[any](ctrl)
	log := logger.NewMockLogger(ctrl)

	substate.EXPECT().
		Run(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(from int, to int, consume Consumer[any]) error {
			for i := from; i < to; i++ {
				if err := consume(TransactionInfo[any]{i, 0, nil}); err != nil {
					return err
				}
			}
			return nil
		})

	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("program must panic")
		}
	}()

	gomock.InOrder(
		extension.EXPECT().PreRun(gomock.Any(), gomock.Any()),
		log.EXPECT().Debugf(gomock.Any(), gomock.Any()),
		extension.EXPECT().PreBlock(gomock.Any(), gomock.Any()),
		extension.EXPECT().PreTransaction(gomock.Any(), gomock.Any()),
		processor.EXPECT().Process(gomock.Any(), gomock.Any()),
		extension.EXPECT().PostTransaction(gomock.Any(), gomock.Any()),
		extension.EXPECT().PostBlock(gomock.Any(), gomock.Any()).Do(func(any, any) {
			panic("stop")
		}),
	)

	executor := newExecutor[any](substate, log)
	if err := executor.Run(Params{From: 1, To: 2, NumWorkers: 2, ParallelismGranularity: BlockLevel},
		processor,
		[]Extension[any]{extension}); err != nil {
		t.Errorf("execution failed: %v", err)
	}
}

func TestProcessor_PanicCaughtInPostRunIsReturned_TransactionLevelParallelism(t *testing.T) {
	ctrl := gomock.NewController(t)
	substate := NewMockProvider[any](ctrl)
	processor := NewMockProcessor[any](ctrl)
	extension := NewMockExtension[any](ctrl)
	log := logger.NewMockLogger(ctrl)

	substate.EXPECT().
		Run(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(from int, to int, consume Consumer[any]) error {
			for i := from; i < to; i++ {
				if err := consume(TransactionInfo[any]{i, 0, nil}); err != nil {
					return err
				}
			}
			return nil
		})

	gomock.InOrder(
		extension.EXPECT().PreRun(gomock.Any(), gomock.Any()),
		log.EXPECT().Debugf(gomock.Any(), gomock.Any()),
		extension.EXPECT().PreTransaction(gomock.Any(), gomock.Any()),
		processor.EXPECT().Process(gomock.Any(), gomock.Any()),
		extension.EXPECT().PostTransaction(gomock.Any(), gomock.Any()),
		extension.EXPECT().PostRun(gomock.Any(), gomock.Any(), gomock.Any()).Do(func(any, any, any) {
			panic("stop")
		}),
	)

	executor := newExecutor[any](substate, log)
	err := executor.Run(Params{From: 1, To: 2, NumWorkers: 2, ParallelismGranularity: TransactionLevel},
		processor,
		[]Extension[any]{extension})
	if err == nil {
		t.Error("run must return an error")
	}

	wantedErr := "sending forward recovered panic from PostRun; stop"
	if strings.Compare(err.Error(), wantedErr) != 0 {
		t.Errorf("unexpected err \nwant: %v\ngot: %v", wantedErr, err.Error())
	}

}

func TestProcessor_PanicCaughtInPostRunIsReturned_BlockLevelParallelism(t *testing.T) {
	ctrl := gomock.NewController(t)
	substate := NewMockProvider[any](ctrl)
	processor := NewMockProcessor[any](ctrl)
	extension := NewMockExtension[any](ctrl)
	log := logger.NewMockLogger(ctrl)

	substate.EXPECT().
		Run(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(from int, to int, consume Consumer[any]) error {
			for i := from; i < to; i++ {
				if err := consume(TransactionInfo[any]{i, 0, nil}); err != nil {
					return err
				}
			}
			return nil
		})

	gomock.InOrder(
		extension.EXPECT().PreRun(gomock.Any(), gomock.Any()),
		log.EXPECT().Debugf(gomock.Any(), gomock.Any()),
		extension.EXPECT().PreBlock(gomock.Any(), gomock.Any()),
		extension.EXPECT().PreTransaction(gomock.Any(), gomock.Any()),
		processor.EXPECT().Process(gomock.Any(), gomock.Any()),
		extension.EXPECT().PostTransaction(gomock.Any(), gomock.Any()),
		extension.EXPECT().PostBlock(gomock.Any(), gomock.Any()),
		extension.EXPECT().PostRun(gomock.Any(), gomock.Any(), gomock.Any()).Do(func(any, any, any) {
			panic("stop")
		}),
	)

	executor := newExecutor[any](substate, log)
	err := executor.Run(Params{From: 1, To: 2, NumWorkers: 2, ParallelismGranularity: BlockLevel},
		processor,
		[]Extension[any]{extension})
	if err == nil {
		t.Error("run must return an error")
	}

	wantedErr := "sending forward recovered panic from PostRun; stop"
	if strings.Compare(err.Error(), wantedErr) != 0 {
		t.Errorf("unexpected err \nwant: %v\ngot: %v", wantedErr, err.Error())
	}

}
