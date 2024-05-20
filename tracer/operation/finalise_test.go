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
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/Fantom-foundation/Aida/tracer/context"
)

func initFinalise(t *testing.T) (*context.Replay, *Finalise, bool) {
	rand.Seed(time.Now().UnixNano())
	deleteEmpty := rand.Intn(2) == 1
	// create context context
	ctx := context.NewReplay()

	// create new operation
	op := NewFinalise(deleteEmpty)
	if op == nil {
		t.Fatalf("failed to create operation")
	}
	// check id
	if op.GetId() != FinaliseID {
		t.Fatalf("wrong ID returned")
	}

	return ctx, op, deleteEmpty
}

// TestFinaliseReadWrite writes a new Finalise object into a buffer, reads from it,
// and checks equality.
func TestFinaliseReadWrite(t *testing.T) {
	_, op1, _ := initFinalise(t)
	testOperationReadWrite(t, op1, ReadFinalise)
}

// TestFinaliseDebug creates a new Finalise object and checks its Debug message.
func TestFinaliseDebug(t *testing.T) {
	ctx, op, deleteEmpty := initFinalise(t)
	testOperationDebug(t, ctx, op, fmt.Sprint(deleteEmpty))
}

// TestFinaliseExecute
func TestFinaliseExecute(t *testing.T) {
	ctx, op, deleteEmpty := initFinalise(t)

	// check execution
	mock := NewMockStateDB()
	op.Execute(mock, ctx)

	// check whether methods were correctly called
	expected := []Record{{FinaliseID, []any{deleteEmpty}}}
	mock.compareRecordings(expected, t)
}
