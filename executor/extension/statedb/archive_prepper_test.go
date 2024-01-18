package statedb

import (
	"testing"

	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/state"
	"github.com/Fantom-foundation/Aida/txcontext"
	"go.uber.org/mock/gomock"
)

func TestArchivePrepper_ArchiveGetsReleasedInPostBlock(t *testing.T) {
	ext := MakeArchivePrepper()

	ctrl := gomock.NewController(t)
	db := state.NewMockStateDB(ctrl)
	archive := state.NewMockNonCommittableStateDB(ctrl)

	gomock.InOrder(
		db.EXPECT().GetArchiveState(uint64(1)).Return(archive, nil),
		archive.EXPECT().Release(),
	)

	state := executor.State[txcontext.WithValidation]{
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
