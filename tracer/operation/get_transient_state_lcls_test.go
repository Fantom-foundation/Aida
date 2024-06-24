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

func initGetTransientStateLcls(t *testing.T) (*context.Replay, *GetTransientStateLcls, common.Address, common.Hash) {
	// create context context
	ctx := context.NewReplay()

	// create new operation
	op := NewGetTransientStateLcls()
	if op == nil {
		t.Fatalf("failed to create operation")
	}
	// check id
	if op.GetId() != GetTransientStateLclsID {
		t.Fatalf("wrong ID returned")
	}

	addr := getRandomAddress(t)
	ctx.EncodeContract(addr)

	storage := common.BytesToHash(getRandomAddress(t).Bytes())
	ctx.EncodeKey(storage)

	return ctx, op, addr, storage
}

// TestGetTransientStateLclsReadWrite writes a new GetTransientStateLcls object into a buffer, reads from it,
// and checks equality.
func TestGetTransientStateLclsReadWrite(t *testing.T) {
	_, op1, _, _ := initGetTransientStateLcls(t)
	testOperationReadWrite(t, op1, ReadGetTransientStateLcls)
}

// TestGetTransientStateLclsDebug creates a new GetTransientStateLcls object and checks its Debug message.
func TestGetTransientStateLclsDebug(t *testing.T) {
	ctx, op, addr, storage := initGetTransientStateLcls(t)
	testOperationDebug(t, ctx, op, fmt.Sprint(addr, storage))
}

// TestGetTransientStateLclsExecute
func TestGetTransientStateLclsExecute(t *testing.T) {
	ctx, op, addr, storage := initGetTransientStateLcls(t)

	// check execution
	mock := NewMockStateDB()
	op.Execute(mock, ctx)

	// check whether methods were correctly called
	expected := []Record{{GetStateID, []any{addr, storage}}}
	mock.compareRecordings(expected, t)
}
