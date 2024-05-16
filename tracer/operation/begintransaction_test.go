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

func initBeginTransaction(t *testing.T) (*context.Replay, *BeginTransaction) {
	rand.Seed(time.Now().UnixNano())
	num := rand.Uint32()

	// create context context
	ctx := context.NewReplay()

	// create new operation
	op := NewBeginTransaction(num)
	if op == nil {
		t.Fatalf("failed to create operation")
	}
	// check id
	if op.GetId() != BeginTransactionID {
		t.Fatalf("wrong ID returned")
	}

	return ctx, op
}

// TestBeginTransactionReadWrite writes a new BeginTransaction object into a buffer, reads from it,
// and checks equality.
func TestBeginTransactionReadWrite(t *testing.T) {
	_, op1 := initBeginTransaction(t)
	testOperationReadWrite(t, op1, ReadBeginTransaction)
}

// TestBeginTransactionDebug creates a new BeginTransaction object and checks its Debug message.
func TestBeginTransactionDebug(t *testing.T) {
	ctx, op := initBeginTransaction(t)
	testOperationDebug(t, ctx, op, fmt.Sprintf("%v", op.TransactionNumber))
}

// TestBeginTransactionExecute
func TestBeginTransactionExecute(t *testing.T) {
	ctx, op := initBeginTransaction(t)

	// check execution
	mock := NewMockStateDB()
	op.Execute(mock, ctx)

	// check whether methods were correctly called
	expected := []Record{{BeginTransactionID, []any{op.TransactionNumber}}}
	mock.compareRecordings(expected, t)
}
