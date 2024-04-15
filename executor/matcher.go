// Copyright 2024 Fantom Foundation
// This file is part of Aida Testing Infrastructure for Sonic
//
// Aida is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Aida is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with Aida. If not, see <http://www.gnu.org/licenses/>.

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
func AtBlock[T any](block int) gomock.Matcher {
	return atBlock[T]{block}
}

// AtBlock matches executor.State instances with the given block and
// transaction number.
func AtTransaction[T any](block int, transaction int) gomock.Matcher {
	return atTransaction[T]{block, transaction}
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

type atBlock[T any] struct {
	expectedBlock int
}

func (m atBlock[T]) Matches(value any) bool {
	state, ok := value.(State[T])
	return ok && state.Block == m.expectedBlock
}

func (m atBlock[T]) String() string {
	return fmt.Sprintf("at block %d", m.expectedBlock)
}

type atTransaction[T any] struct {
	expectedBlock       int
	expectedTransaction int
}

func (m atTransaction[T]) Matches(value any) bool {
	state, ok := value.(State[T])
	return ok && state.Block == m.expectedBlock && state.Transaction == m.expectedTransaction
}

func (m atTransaction[T]) String() string {
	return fmt.Sprintf("at transaction %d/%d", m.expectedBlock, m.expectedTransaction)
}

type withState struct {
	state state.StateDB
}

func (m withState) Matches(value any) bool {
	if ctx, ok := value.(Context); ok {
		return ctx.State == m.state
	}
	if ctx, ok := value.(*Context); ok {
		return ctx.State == m.state
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
	state, ok := value.(State[*substate.Substate])
	return ok && state.Data == m.substate
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

// ----------------------------------------------------------------------------

func MatchRate(constraint gomock.Matcher, name string) gomock.Matcher {
	return matchRate{constraint, name}
}

type matchRate struct {
	constraint gomock.Matcher
	name       string
}

func (m matchRate) Matches(value any) bool {
	txRate, ok := value.(float64)
	return ok && m.constraint.Matches(txRate)
}

func (m matchRate) String() string {
	return fmt.Sprintf("log should have a %v that is %v", m.name, m.constraint)
}
