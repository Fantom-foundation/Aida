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

func initExist(t *testing.T) (*context.Replay, *Exist, common.Address) {
	addr := getRandomAddress(t)
	// create context context
	ctx := context.NewReplay()
	contract := ctx.EncodeContract(addr)

	// create new operation
	op := NewExist(contract)
	if op == nil {
		t.Fatalf("failed to create operation")
	}
	// check id
	if op.GetId() != ExistID {
		t.Fatalf("wrong ID returned")
	}

	return ctx, op, addr
}

// TestExistReadWrite writes a new Exist object into a buffer, reads from it,
// and checks equality.
func TestExistReadWrite(t *testing.T) {
	_, op1, _ := initExist(t)
	testOperationReadWrite(t, op1, ReadExist)
}

// TestExistDebug creates a new Exist object and checks its Debug message.
func TestExistDebug(t *testing.T) {
	ctx, op, addr := initExist(t)
	testOperationDebug(t, ctx, op, fmt.Sprint(addr))
}

// TestExistExecute
func TestExistExecute(t *testing.T) {
	ctx, op, addr := initExist(t)

	// check execution
	mock := NewMockStateDB()
	op.Execute(mock, ctx)

	// check whether methods were correctly called
	expected := []Record{{ExistID, []any{addr}}}
	mock.compareRecordings(expected, t)
}
