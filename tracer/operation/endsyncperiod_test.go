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

func initEndSyncPeriod(t *testing.T) (*context.Replay, *EndSyncPeriod) {
	// create context context
	ctx := context.NewReplay()

	// create new operation
	op := NewEndSyncPeriod()
	if op == nil {
		t.Fatalf("failed to create operation")
	}
	// check id
	if op.GetId() != EndSyncPeriodID {
		t.Fatalf("wrong ID returned")
	}

	return ctx, op
}

// TestEndSyncPeriodReadWrite writes a new EndSyncPeriod object into a buffer, reads from it,
// and checks equality.
func TestEndSyncPeriodReadWrite(t *testing.T) {
	_, op1 := initEndSyncPeriod(t)
	testOperationReadWrite(t, op1, ReadEndSyncPeriod)
}

// TestEndSyncPeriodDebug creates a new EndSyncPeriod object and checks its Debug message.
func TestEndSyncPeriodDebug(t *testing.T) {
	ctx, op := initEndSyncPeriod(t)
	testOperationDebug(t, ctx, op, "")
}

// TestEndSyncPeriodExecute
func TestEndSyncPeriodExecute(t *testing.T) {
	ctx, op := initEndSyncPeriod(t)

	// check execution
	mock := NewMockStateDB()
	op.Execute(mock, ctx)

	// check whether methods were correctly called
	expected := []Record{{EndSyncPeriodID, []any{}}}
	mock.compareRecordings(expected, t)
}
