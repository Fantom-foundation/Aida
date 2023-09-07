package executor

import (
	"errors"
	"fmt"

	"github.com/Fantom-foundation/Aida/state"
	substate "github.com/Fantom-foundation/Substate"
	gomock "go.uber.org/mock/gomock"
)

// ----------------------------------------------------------------------------
//                                   Matcher
// ----------------------------------------------------------------------------

// AtBlock matches executor.State instances with the given block.
func AtBlock(block int) gomock.Matcher {
	return atBlock{block}
}

// AtBlock matches executor.State instances with the given block and
// transaction number.
func AtTransaction(block int, transaction int) gomock.Matcher {
	return atTransaction{block, transaction}
}

// WithState matches executor.State instances with the given state.
func WithState(state state.StateDB) gomock.Matcher {
	return withState{state}
}

// WithSubstate matches executor.State instances with the given substate.
func WithSubstate(substate *substate.Substate) gomock.Matcher {
	return withSubstate{substate}
}

// ----------------------------------------------------------------------------

type atBlock struct {
	expectedBlock int
}

func (m atBlock) Matches(value any) bool {
	state, ok := value.(State)
	return ok && state.Block == m.expectedBlock
}

func (m atBlock) String() string {
	return fmt.Sprintf("at block %d", m.expectedBlock)
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
	return fmt.Sprintf("at transaction %d/%d", m.expectedBlock, m.expectedTransaction)
}

type withState struct {
	state state.StateDB
}

func (m withState) Matches(value any) bool {
	state, ok := value.(State)
	return ok && state.State == m.state
}

func (m withState) String() string {
	return fmt.Sprintf("with state %p", m.state)
}

func WithError(err error) gomock.Matcher {
	return withError{err}
}

type withError struct {
	err error
}

func (m withError) Matches(value any) bool {
	err, ok := value.(error)
	return ok && errors.Is(err, m.err)
}

func (m withError) String() string {
	return fmt.Sprintf("with error %v", m.err)
}

type withSubstate struct {
	substate *substate.Substate
}

func (m withSubstate) Matches(value any) bool {
	state, ok := value.(State)
	return ok && state.Substate == m.substate
}

func (m withSubstate) String() string {
	return fmt.Sprintf("with substate %p", m.substate)
}
