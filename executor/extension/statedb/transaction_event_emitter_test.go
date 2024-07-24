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

package statedb

import (
	"testing"

	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/state"
	"go.uber.org/mock/gomock"
)

func TestTransactionEventEmitter_PreTransactionCallsBeginTx(t *testing.T) {
	ext := MakeTransactionEventEmitter[any]()

	mockCtrl := gomock.NewController(t)
	mockStateDB := state.NewMockStateDB(mockCtrl)

	mockStateDB.EXPECT().BeginTransaction(uint32(0))

	state := executor.State[any]{
		Transaction: 0,
	}
	ctx := &executor.Context{
		State: mockStateDB,
	}
	if err := ext.PreTransaction(state, ctx); err != nil {
		t.Fatalf("failed to to run pre-block: %v", err)
	}
}

func TestTransactionEventEmitter_PostTransactionCallsEndTx(t *testing.T) {
	ext := MakeTransactionEventEmitter[any]()

	mockCtrl := gomock.NewController(t)
	mockStateDB := state.NewMockStateDB(mockCtrl)

	mockStateDB.EXPECT().EndTransaction()

	state := executor.State[any]{
		Block:       0,
		Transaction: 0,
	}
	ctx := &executor.Context{
		State: mockStateDB,
	}

	if err := ext.PostTransaction(state, ctx); err != nil {
		t.Fatalf("failed to to run post-block: %v", err)
	}

}
