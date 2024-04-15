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

func initGetCodeHashLc(t *testing.T) (*context.Replay, *GetCodeHashLc, common.Address) {
	// create context context
	ctx := context.NewReplay()

	addr := getRandomAddress(t)
	ctx.EncodeContract(addr)

	// create new operation
	op := NewGetCodeHashLc()
	if op == nil {
		t.Fatalf("failed to create operation")
	}
	// check id
	if op.GetId() != GetCodeHashLcID {
		t.Fatalf("wrong ID returned")
	}

	return ctx, op, addr
}

// TestGetCodeHashLcReadWrite writes a new GetCodeHashLc object into a buffer, reads from it,
// and checks equality.
func TestGetCodeHashLcReadWrite(t *testing.T) {
	_, op1, _ := initGetCodeHashLc(t)
	testOperationReadWrite(t, op1, ReadGetCodeHashLc)
}

// TestGetCodeHashLcDebug creates a new GetCodeHashLc object and checks its Debug message.
func TestGetCodeHashLcDebug(t *testing.T) {
	ctx, op, addr := initGetCodeHashLc(t)
	testOperationDebug(t, ctx, op, fmt.Sprint(addr))
}

// TestGetCodeHashLcExecute
func TestGetCodeHashLcExecute(t *testing.T) {
	ctx, op, addr := initGetCodeHashLc(t)

	// check execution
	mock := NewMockStateDB()
	op.Execute(mock, ctx)

	// check whether methods were correctly called
	expected := []Record{{GetCodeHashID, []any{addr}}}
	mock.compareRecordings(expected, t)
}
