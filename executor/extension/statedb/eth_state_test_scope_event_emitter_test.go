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

	"github.com/Fantom-foundation/Aida/ethtest"
	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/state"
	"github.com/Fantom-foundation/Aida/txcontext"
	"go.uber.org/mock/gomock"
)

func TestEthStateScopeEventEmitter_PreTransactionCallsBeginBlockAndBeginTransaction(t *testing.T) {
	ctrl := gomock.NewController(t)
	db := state.NewMockStateDB(ctrl)

	ext := ethStateScopeEventEmitter{}

	db.EXPECT().BeginBlock(uint64(2))
	db.EXPECT().BeginTransaction(uint32(1))

	st := executor.State[txcontext.TxContext]{Block: 1, Transaction: 1, Data: ethtest.CreateTestTransaction(t)}
	ctx := &executor.Context{State: db}
	err := ext.PreTransaction(st, ctx)
	if err != nil {
		t.Fatalf("unexpected err; %v", err)
	}
}

func TestEthStateScopeEventEmitter_PostTransactionCallsEndBlockAndEndTransaction(t *testing.T) {
	ctrl := gomock.NewController(t)
	db := state.NewMockStateDB(ctrl)

	ext := ethStateScopeEventEmitter{}

	db.EXPECT().EndTransaction()
	db.EXPECT().EndBlock()

	st := executor.State[txcontext.TxContext]{Block: 1, Transaction: 1, Data: ethtest.CreateTestTransaction(t)}
	ctx := &executor.Context{State: db}
	err := ext.PostTransaction(st, ctx)
	if err != nil {
		t.Fatalf("unexpected err; %v", err)
	}
}
