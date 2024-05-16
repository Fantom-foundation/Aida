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

func initGetStateLcls(t *testing.T) (*context.Replay, *GetStateLcls, common.Address, common.Hash) {
	// create context context
	ctx := context.NewReplay()

	// create new operation
	op := NewGetStateLcls()
	if op == nil {
		t.Fatalf("failed to create operation")
	}
	// check id
	if op.GetId() != GetStateLclsID {
		t.Fatalf("wrong ID returned")
	}

	addr := getRandomAddress(t)
	ctx.EncodeContract(addr)

	storage := getRandomAddress(t).Hash()
	ctx.EncodeKey(storage)

	return ctx, op, addr, storage
}

// TestGetStateLclsReadWrite writes a new GetStateLcls object into a buffer, reads from it,
// and checks equality.
func TestGetStateLclsReadWrite(t *testing.T) {
	_, op1, _, _ := initGetStateLcls(t)
	testOperationReadWrite(t, op1, ReadGetStateLcls)
}

// TestGetStateLclsDebug creates a new GetStateLcls object and checks its Debug message.
func TestGetStateLclsDebug(t *testing.T) {
	ctx, op, addr, storage := initGetStateLcls(t)
	testOperationDebug(t, ctx, op, fmt.Sprint(addr, storage))
}

// TestGetStateLclsExecute
func TestGetStateLclsExecute(t *testing.T) {
	ctx, op, addr, storage := initGetStateLcls(t)

	// check execution
	mock := NewMockStateDB()
	op.Execute(mock, ctx)

	// check whether methods were correctly called
	expected := []Record{{GetStateID, []any{addr, storage}}}
	mock.compareRecordings(expected, t)
}
