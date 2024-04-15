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

func initGetCodeSize(t *testing.T) (*context.Replay, *GetCodeSize, common.Address) {
	addr := getRandomAddress(t)
	// create context context
	ctx := context.NewReplay()
	contract := ctx.EncodeContract(addr)

	// create new operation
	op := NewGetCodeSize(contract)
	if op == nil {
		t.Fatalf("failed to create operation")
	}
	// check id
	if op.GetId() != GetCodeSizeID {
		t.Fatalf("wrong ID returned")
	}

	return ctx, op, addr
}

// TestGetCodeSizeReadWrite writes a new GetCodeSize object into a buffer, reads from it,
// and checks equality.
func TestGetCodeSizeReadWrite(t *testing.T) {
	_, op1, _ := initGetCodeSize(t)
	testOperationReadWrite(t, op1, ReadGetCodeSize)
}

// TestGetCodeSizeDebug creates a new GetCodeSize object and checks its Debug message.
func TestGetCodeSizeDebug(t *testing.T) {
	ctx, op, addr := initGetCodeSize(t)
	testOperationDebug(t, ctx, op, fmt.Sprint(addr))
}

// TestGetCodeSizeExecute creates a new GetCodeSize object and checks its execution signature.
func TestGetCodeSizeExecute(t *testing.T) {
	ctx, op, addr := initGetCodeSize(t)

	// check execution
	mock := NewMockStateDB()
	op.Execute(mock, ctx)

	// check whether methods were correctly called
	expected := []Record{{GetCodeSizeID, []any{addr}}}
	mock.compareRecordings(expected, t)
}
