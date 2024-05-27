// Copyright 2024 Fantom Foundation
// This file is part of Aida Testing Infrastructure for Sonic
//
// Aida is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Aida is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with Aida. If not, see <http://www.gnu.org/licenses/>.

package operation

import (
	"testing"

	"github.com/Fantom-foundation/Aida/tracer/context"
)

func initEndBlock(t *testing.T) (*context.Replay, *EndBlock) {
	// create context context
	ctx := context.NewReplay()

	// create new operation
	op := NewEndBlock()
	if op == nil {
		t.Fatalf("failed to create operation")
	}
	// check id
	if op.GetId() != EndBlockID {
		t.Fatalf("wrong ID returned")
	}

	return ctx, op
}

// TestEndBlockReadWrite writes a new EndBlock object into a buffer, reads from it,
// and checks equality.
func TestEndBlockReadWrite(t *testing.T) {
	_, op1 := initEndBlock(t)
	testOperationReadWrite(t, op1, ReadEndBlock)
}

// TestEndBlockDebug creates a new EndBlock object and checks its Debug message.
func TestEndBlockDebug(t *testing.T) {
	ctx, op := initEndBlock(t)
	testOperationDebug(t, ctx, op, "")
}

// TestEndBlockExecute
func TestEndBlockExecute(t *testing.T) {
	ctx, op := initEndBlock(t)

	// check execution
	mock := NewMockStateDB()
	op.Execute(mock, ctx)

	// check whether methods were correctly called
	expected := []Record{{EndBlockID, []any{}}}
	mock.compareRecordings(expected, t)
}
