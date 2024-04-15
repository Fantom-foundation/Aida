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

package operation

import (
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/Fantom-foundation/Aida/tracer/context"
)

func initBeginBlock(t *testing.T) (*context.Replay, *BeginBlock, uint64) {
	rand.Seed(time.Now().UnixNano())
	blId := rand.Uint64()

	// create context context
	ctx := context.NewReplay()

	// create new operation
	op := NewBeginBlock(blId)
	if op == nil {
		t.Fatalf("failed to create operation")
	}
	// check id
	if op.GetId() != BeginBlockID {
		t.Fatalf("wrong ID returned")
	}

	return ctx, op, blId
}

// TestBeginBlockReadWrite writes a new BeginBlock object into a buffer, reads from it,
// and checks equality.
func TestBeginBlockReadWrite(t *testing.T) {
	_, op1, _ := initBeginBlock(t)
	testOperationReadWrite(t, op1, ReadBeginBlock)
}

// TestBeginBlockDebug creates a new BeginBlock object and checks its Debug message.
func TestBeginBlockDebug(t *testing.T) {
	ctx, op, value := initBeginBlock(t)
	testOperationDebug(t, ctx, op, fmt.Sprint(value))
}

// TestBeginBlockExecute
func TestBeginBlockExecute(t *testing.T) {
	ctx, op, _ := initBeginBlock(t)

	// check execution
	mock := NewMockStateDB()
	op.Execute(mock, ctx)

	// check whether methods were correctly called
	expected := []Record{{BeginBlockID, []any{op.BlockNumber}}}
	mock.compareRecordings(expected, t)
}
