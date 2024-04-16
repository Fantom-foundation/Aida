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
	"testing"

	"github.com/Fantom-foundation/Aida/tracer/context"
	"github.com/ethereum/go-ethereum/common"
)

func initSetState(t *testing.T) (*context.Replay, *SetState, common.Address, common.Hash, common.Hash) {
	addr := getRandomAddress(t)
	storage := getRandomAddress(t).Hash()
	value := getRandomAddress(t).Hash()

	// create context context
	ctx := context.NewReplay()
	contract := ctx.EncodeContract(addr)
	sIdx, _ := ctx.EncodeKey(storage)

	// create new operation
	op := NewSetState(contract, sIdx, value)
	if op == nil {
		t.Fatalf("failed to create operation")
	}
	// check id
	if op.GetId() != SetStateID {
		t.Fatalf("wrong ID returned")
	}

	return ctx, op, addr, storage, value
}

// TestSetStateReadWrite writes a new SetState object into a buffer, reads from it,
// and checks equality.
func TestSetStateReadWrite(t *testing.T) {
	_, op1, _, _, _ := initSetState(t)
	testOperationReadWrite(t, op1, ReadSetState)
}

// TestSetStateDebug creates a new SetState object and checks its Debug message.
func TestSetStateDebug(t *testing.T) {
	ctx, op, addr, storage, value := initSetState(t)
	testOperationDebug(t, ctx, op, fmt.Sprint(addr, storage, value))
}

// TestSetStateExecute
func TestSetStateExecute(t *testing.T) {
	ctx, op, addr, storage, value := initSetState(t)

	// check execution
	mock := NewMockStateDB()
	op.Execute(mock, ctx)

	// check whether methods were correctly called
	expected := []Record{{SetStateID, []any{addr, storage, value}}}
	mock.compareRecordings(expected, t)
}
