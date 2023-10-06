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

// Lt matches every value less than the given limit.
func Lt(limit float64) gomock.Matcher {
	return lt{limit}
}

// Gt matches every value greater than the given limit.
func Gt(limit float64) gomock.Matcher {
	return gt{limit}
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
	if context, ok := value.(Context); ok {
		return context.State == m.state
	}
	if context, ok := value.(*Context); ok {
		return context.State == m.state
	}
	return false
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

type lt struct {
	limit float64
}

func (m lt) Matches(value any) bool {
	v, ok := value.(float64)
	return ok && v < m.limit
}

func (m lt) String() string {
	return fmt.Sprintf("less than %v", m.limit)
}

type gt struct {
	limit float64
}

func (m gt) Matches(value any) bool {
	v, ok := value.(float64)
	return ok && v > m.limit
}

func (m gt) String() string {
	return fmt.Sprintf("greater than %v", m.limit)
}
