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
	"math/big"
	"math/rand"
	"testing"
	"time"

	"github.com/Fantom-foundation/Aida/tracer/context"
	"github.com/ethereum/go-ethereum/common"
)

func initSubBalance(t *testing.T) (*context.Replay, *SubBalance, common.Address, *big.Int) {
	rand.Seed(time.Now().UnixNano())
	addr := getRandomAddress(t)
	value := big.NewInt(rand.Int63n(100000))
	// create context context
	ctx := context.NewReplay()
	contract := ctx.EncodeContract(addr)

	// create new operation
	op := NewSubBalance(contract, value)
	if op == nil {
		t.Fatalf("failed to create operation")
	}
	// check id
	if op.GetId() != SubBalanceID {
		t.Fatalf("wrong ID returned")
	}
	return ctx, op, addr, value
}

// TestSubBalanceReadWrite writes a new SubBalance object into a buffer, reads from it,
// and checks equality.
func TestSubBalanceReadWrite(t *testing.T) {
	_, op1, _, _ := initSubBalance(t)
	testOperationReadWrite(t, op1, ReadSubBalance)
}

// TestSubBalanceDebug creates a new SubBalance object and checks its Debug message.
func TestSubBalanceDebug(t *testing.T) {
	ctx, op, addr, value := initSubBalance(t)
	testOperationDebug(t, ctx, op, fmt.Sprint(addr, value))
}

// TestSubBalanceExecute
func TestSubBalanceExecute(t *testing.T) {
	ctx, op, addr, value := initSubBalance(t)

	// check execution
	mock := NewMockStateDB()
	op.Execute(mock, ctx)

	// check whether methods were correctly called
	expected := []Record{{SubBalanceID, []any{addr, value}}}
	mock.compareRecordings(expected, t)
}
