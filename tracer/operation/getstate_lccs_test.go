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
	"github.com/ethereum/go-ethereum/common"
)

func initGetStateLccs(t *testing.T) (*context.Replay, *GetStateLccs, common.Address, common.Hash, common.Hash) {
	rand.Seed(time.Now().UnixNano())
	pos := 0

	// create context context
	ctx := context.NewReplay()

	// create new operation
	op := NewGetStateLccs(pos)
	if op == nil {
		t.Fatalf("failed to create operation")
	}
	// check id
	if op.GetId() != GetStateLccsID {
		t.Fatalf("wrong ID returned")
	}

	addr := getRandomAddress(t)
	ctx.EncodeContract(addr)

	storage := getRandomHash(t)
	ctx.EncodeKey(storage)

	storage2 := getRandomHash(t)

	return ctx, op, addr, storage, storage2
}

// TestGetStateLccsReadWrite writes a new GetStateLccs object into a buffer, reads from it,
// and checks equality.
func TestGetStateLccsReadWrite(t *testing.T) {
	_, op1, _, _, _ := initGetStateLccs(t)
	testOperationReadWrite(t, op1, ReadGetStateLccs)
}

// TestGetStateLccsDebug creates a new GetStateLccs object and checks its Debug message.
func TestGetStateLccsDebug(t *testing.T) {
	ctx, op, addr, storage, _ := initGetStateLccs(t)
	testOperationDebug(t, ctx, op, fmt.Sprint(addr, storage))
}

// TestGetStateLccsExecute
func TestGetStateLccsExecute(t *testing.T) {
	ctx, op, addr, storage, storage2 := initGetStateLccs(t)

	// check execution
	mock := NewMockStateDB()
	op.Execute(mock, ctx)

	ctx.EncodeKey(storage2)

	op.Execute(mock, ctx)

	// check whether methods were correctly called
	expected := []Record{{GetStateID, []any{addr, storage}}, {GetStateID, []any{addr, storage2}}}
	mock.compareRecordings(expected, t)
}
