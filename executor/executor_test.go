package executor

import (
	"errors"
	"fmt"
	"testing"

	"github.com/Fantom-foundation/Aida/state"
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
				consume(i, 0, nil)
				consume(i, 1, nil)
			}
			return nil
		})

	gomock.InOrder(
		processor.EXPECT().Process(AtTransaction(10, 0)),
		processor.EXPECT().Process(AtTransaction(10, 1)),
		processor.EXPECT().Process(AtTransaction(11, 0)),
		processor.EXPECT().Process(AtTransaction(11, 1)),
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
				if err := consume(i, 0, nil); err != nil {
					return err
				}
			}
			return nil
		})

	stop := fmt.Errorf("stop!")
	gomock.InOrder(
		processor.EXPECT().Process(gomock.Any()).Times(3),
		processor.EXPECT().Process(gomock.Any()).Return(stop),
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
				consume(i, 7, nil)
				consume(i, 9, nil)
			}
			return nil
		})

	gomock.InOrder(
		extension.EXPECT().PreRun(AtBlock(10)),

		extension.EXPECT().PreBlock(AtBlock(10)),
		extension.EXPECT().PreTransaction(AtTransaction(10, 7)),
		processor.EXPECT().Process(AtTransaction(10, 7)),
		extension.EXPECT().PostTransaction(AtTransaction(10, 7)),
		extension.EXPECT().PreTransaction(AtTransaction(10, 9)),
		processor.EXPECT().Process(AtTransaction(10, 9)),
		extension.EXPECT().PostTransaction(AtTransaction(10, 9)),
		extension.EXPECT().PostBlock(AtTransaction(10, 9)),

		extension.EXPECT().PreBlock(AtBlock(11)),
		extension.EXPECT().PreTransaction(AtTransaction(11, 7)),
		processor.EXPECT().Process(AtTransaction(11, 7)),
		extension.EXPECT().PostTransaction(AtTransaction(11, 7)),
		extension.EXPECT().PreTransaction(AtTransaction(11, 9)),
		processor.EXPECT().Process(AtTransaction(11, 9)),
		extension.EXPECT().PostTransaction(AtTransaction(11, 9)),
		extension.EXPECT().PostBlock(AtTransaction(11, 9)),

		extension.EXPECT().PostRun(AtBlock(12), nil),
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
				if err := consume(i, 7, nil); err != nil {
					return err
				}
			}
			return nil
		})

	stop := fmt.Errorf("stop!")
	gomock.InOrder(
		extension.EXPECT().PreRun(AtBlock(10)),
		extension.EXPECT().PreBlock(AtBlock(10)),
		extension.EXPECT().PreTransaction(AtTransaction(10, 7)),
		processor.EXPECT().Process(AtTransaction(10, 7)).Return(stop),
		extension.EXPECT().PostRun(AtTransaction(10, 7), stop),
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
		extension.EXPECT().PreRun(AtBlock(10)),
		extension.EXPECT().PostRun(AtBlock(10), nil),
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
				consume(i, 7, nil)
				consume(i, 9, nil)
			}
			return nil
		})

	gomock.InOrder(
		extension1.EXPECT().PreRun(AtBlock(10)),
		extension2.EXPECT().PreRun(AtBlock(10)),

		extension1.EXPECT().PreBlock(AtBlock(10)),
		extension2.EXPECT().PreBlock(AtBlock(10)),

		extension1.EXPECT().PreTransaction(AtTransaction(10, 7)),
		extension2.EXPECT().PreTransaction(AtTransaction(10, 7)),
		processor.EXPECT().Process(AtTransaction(10, 7)),
		extension2.EXPECT().PostTransaction(AtTransaction(10, 7)),
		extension1.EXPECT().PostTransaction(AtTransaction(10, 7)),

		extension1.EXPECT().PreTransaction(AtTransaction(10, 9)),
		extension2.EXPECT().PreTransaction(AtTransaction(10, 9)),
		processor.EXPECT().Process(AtTransaction(10, 9)),
		extension2.EXPECT().PostTransaction(AtTransaction(10, 9)),
		extension1.EXPECT().PostTransaction(AtTransaction(10, 9)),

		extension2.EXPECT().PostBlock(AtBlock(10)),
		extension1.EXPECT().PostBlock(AtBlock(10)),

		extension2.EXPECT().PostRun(AtBlock(11), nil),
		extension1.EXPECT().PostRun(AtBlock(11), nil),
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
		extension1.EXPECT().PreRun(AtBlock(10)).Return(stop),
		extension2.EXPECT().PreRun(AtBlock(10)),

		extension2.EXPECT().PostRun(AtBlock(10), resultError),
		extension1.EXPECT().PostRun(AtBlock(10), resultError),
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
				if err := consume(i, 7, nil); err != nil {
					return err
				}
			}
			return nil
		})

	processor.EXPECT().Process(gomock.Any()).AnyTimes()

	stop := fmt.Errorf("stop!")
	resultError := errors.Join(stop)
	gomock.InOrder(
		extension1.EXPECT().PreRun(AtBlock(10)),
		extension2.EXPECT().PreRun(AtBlock(10)),

		extension1.EXPECT().PreBlock(AtBlock(10)),
		extension2.EXPECT().PreBlock(AtBlock(10)),

		extension1.EXPECT().PreTransaction(AtBlock(10)),
		extension2.EXPECT().PreTransaction(AtBlock(10)),

		extension2.EXPECT().PostTransaction(AtBlock(10)).Return(stop),
		extension1.EXPECT().PostTransaction(AtBlock(10)),

		extension2.EXPECT().PostRun(AtBlock(10), resultError),
		extension1.EXPECT().PostRun(AtBlock(10), resultError),
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
				consume(i, 7, nil)
				consume(i, 9, nil)
			}
			return nil
		})

	gomock.InOrder(
		extension.EXPECT().PreRun(WithState(state)),
		extension.EXPECT().PreBlock(WithState(state)),
		extension.EXPECT().PreTransaction(WithState(state)),
		processor.EXPECT().Process(WithState(state)),
		extension.EXPECT().PostTransaction(WithState(state)),
		extension.EXPECT().PreTransaction(WithState(state)),
		processor.EXPECT().Process(WithState(state)),
		extension.EXPECT().PostTransaction(WithState(state)),
		extension.EXPECT().PostBlock(WithState(state)),
		extension.EXPECT().PostRun(WithState(state), nil),
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

// ----------------------------------------------------------------------------
//                                   Matcher
// ----------------------------------------------------------------------------

func AtBlock(block int) gomock.Matcher {
	return atBlock{block}
}

type atBlock struct {
	expectedBlock int
}

func (m atBlock) Matches(value any) bool {
	state, ok := value.(State)
	return ok && state.Block == m.expectedBlock
}

func (m atBlock) String() string {
	return fmt.Sprintf("should be at block %d", m.expectedBlock)
}

func AtTransaction(block int, transaction int) gomock.Matcher {
	return atTransaction{block, transaction}
}

type atTransaction struct {
	expectedBlock       int
	expectedTransaction int
}

func (m atTransaction) Matches(value any) bool {
	state, ok := value.(State)
	return ok && state.Block == m.expectedBlock && state.Transaction == m.expectedTransaction
}

func (m atTransaction) String() string {
	return fmt.Sprintf("should be at transaction %d/%d", m.expectedBlock, m.expectedTransaction)
}

func WithState(state state.StateDB) gomock.Matcher {
	return withState{state}
}

type withState struct {
	state state.StateDB
}

func (m withState) Matches(value any) bool {
	state, ok := value.(State)
	return ok && state.State == m.state
}

func (m withState) String() string {
	return fmt.Sprintf("should reference state %p", m.state)
}
