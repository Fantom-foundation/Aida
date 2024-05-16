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

func TestArchivePrepper_ArchiveGetsReleasedInPostBlock(t *testing.T) {
	ext := MakeArchivePrepper[any]()

	ctrl := gomock.NewController(t)
	db := state.NewMockStateDB(ctrl)
	archive := state.NewMockNonCommittableStateDB(ctrl)

	gomock.InOrder(
		db.EXPECT().GetArchiveState(uint64(1)).Return(archive, nil),
		archive.EXPECT().Release(),
	)

	state := executor.State[any]{
		Block: 2,
	}
	ctx := &executor.Context{
		State: db,
	}
	if err := ext.PreBlock(state, ctx); err != nil {
		t.Fatalf("failed to to run pre-block: %v", err)
	}
	if err := ext.PostBlock(state, ctx); err != nil {
		t.Fatalf("failed to to run post-block: %v", err)
	}
}
