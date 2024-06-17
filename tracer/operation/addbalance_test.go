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
	"github.com/holiman/uint256"
)

func initAddBalance(t *testing.T) (*context.Replay, *AddBalance, common.Address, *uint256.Int) {
	rand.Seed(time.Now().UnixNano())
	addr := getRandomAddress(t)
	value := uint256.NewInt(uint64(rand.Int63n(100000)))
	// create context context
	ctx := context.NewReplay()
	contract := ctx.EncodeContract(addr)

	// create new operation
	op := NewAddBalance(contract, value)
	if op == nil {
		t.Fatalf("failed to create operation")
	}
	// check id
	if op.GetId() != AddBalanceID {
		t.Fatalf("wrong ID returned")
	}
	return ctx, op, addr, value
}

// TestAddBalanceReadWrite writes a new AddBalance object into a buffer, reads from it,
// and checks equality.
func TestAddBalanceReadWrite(t *testing.T) {
	_, op1, _, _ := initAddBalance(t)
	testOperationReadWrite(t, op1, ReadAddBalance)
}

// TestAddBalanceDebug creates a new AddBalance object and checks its Debug message.
func TestAddBalanceDebug(t *testing.T) {
	ctx, op, addr, value := initAddBalance(t)
	testOperationDebug(t, ctx, op, fmt.Sprint(addr, value))
}

// TestAddBalanceExecute
func TestAddBalanceExecute(t *testing.T) {
	ctx, op, addr, value := initAddBalance(t)

	// check execution
	mock := NewMockStateDB()
	op.Execute(mock, ctx)

	// check whether methods were correctly called
	expected := []Record{{AddBalanceID, []any{addr, value}}}
	mock.compareRecordings(expected, t)
}
