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

package statedb

import (
	"testing"

	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/state"
	"go.uber.org/mock/gomock"
)

func TestTxGeneratorBlockEventEmitter_SingleBlock(t *testing.T) {
	ext := MakeTxGeneratorBlockEventEmitter[any]()

	mockCtrl := gomock.NewController(t)
	mockStateDB := state.NewMockStateDB(mockCtrl)

	gomock.InOrder(
		mockStateDB.EXPECT().BeginBlock(uint64(0)),
		mockStateDB.EXPECT().EndBlock(),
		mockStateDB.EXPECT().BeginBlock(uint64(1)),
		mockStateDB.EXPECT().EndBlock(),
	)

	state := executor.State[any]{
		Block: 0,
	}
	ctx := &executor.Context{
		State: mockStateDB,
	}
	if err := ext.PreTransaction(state, ctx); err != nil {
		t.Fatalf("failed to to run pre-transaction: %v", err)
	}

	// increment the block number to make sure the block is ended
	// and the next block is started
	state.Block = 1
	if err := ext.PreTransaction(state, ctx); err != nil {
		t.Fatalf("failed to to run pre-transaction: %v", err)
	}

	// call post run to end the last block
	if err := ext.PostRun(state, ctx, nil); err != nil {
		t.Fatalf("failed to to run post-run: %v", err)
	}
}
