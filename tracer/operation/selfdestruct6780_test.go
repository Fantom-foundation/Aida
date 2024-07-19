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

func initSelfDestruct6780(t *testing.T) (*context.Replay, *SelfDestruct6780, common.Address) {
	addr := getRandomAddress(t)
	// create context context
	ctx := context.NewReplay()
	contract := ctx.EncodeContract(addr)

	// create new operation
	op := NewSelfDestruct6780(contract)
	if op == nil {
		t.Fatalf("failed to create operation")
	}
	// check id
	if op.GetId() != SelfDestruct6780ID {
		t.Fatalf("wrong ID returned")
	}

	return ctx, op, addr
}

// TestSelfDestruct6780ReadWrite writes a new SelfDestruct6780 object into a buffer, reads from it,
// and checks equality.
func TestSelfDestruct6780ReadWrite(t *testing.T) {
	_, op1, _ := initSelfDestruct6780(t)
	testOperationReadWrite(t, op1, ReadSelfDestruct6780)
}

// TestSelfDestruct6780Debug creates a new SelfDestruct6780 object and checks its Debug message.
func TestSelfDestruct6780Debug(t *testing.T) {
	ctx, op, addr := initSelfDestruct6780(t)
	testOperationDebug(t, ctx, op, fmt.Sprint(addr))
}

// TestSelfDestruct6780Execute
func TestSelfDestruct6780Execute(t *testing.T) {
	ctx, op, addr := initSelfDestruct6780(t)

	// check execution
	mock := NewMockStateDB()
	op.Execute(mock, ctx)

	// check whether methods were correctly called
	expected := []Record{{SelfDestruct6780ID, []any{addr}}}
	mock.compareRecordings(expected, t)
}
