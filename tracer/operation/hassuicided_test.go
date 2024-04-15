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
	"testing"

	"github.com/Fantom-foundation/Aida/tracer/context"
	"github.com/ethereum/go-ethereum/common"
)

func initHasSuicided(t *testing.T) (*context.Replay, *HasSuicided, common.Address) {
	addr := getRandomAddress(t)
	// create context context
	ctx := context.NewReplay()
	contract := ctx.EncodeContract(addr)

	// create new operation
	op := NewHasSuicided(contract)
	if op == nil {
		t.Fatalf("failed to create operation")
	}
	// check id
	if op.GetId() != HasSuicidedID {
		t.Fatalf("wrong ID returned")
	}

	return ctx, op, addr
}

// TestHasSuicidedReadWrite writes a new HasSuicided object into a buffer, reads from it,
// and checks equality.
func TestHasSuicidedReadWrite(t *testing.T) {
	_, op1, _ := initHasSuicided(t)
	testOperationReadWrite(t, op1, ReadHasSuicided)
}

// TestHasSuicidedDebug creates a new HasSuicided object and checks its Debug message.
func TestHasSuicidedDebug(t *testing.T) {
	ctx, op, addr := initHasSuicided(t)
	testOperationDebug(t, ctx, op, fmt.Sprint(addr))
}

// TestHasSuicidedExecute
func TestHasSuicidedExecute(t *testing.T) {
	ctx, op, addr := initHasSuicided(t)

	// check execution
	mock := NewMockStateDB()
	op.Execute(mock, ctx)

	// check whether methods were correctly called
	expected := []Record{{HasSuicidedID, []any{addr}}}
	mock.compareRecordings(expected, t)
}
