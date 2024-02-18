package statedb

import (
	"testing"

	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/rpc"
	"github.com/Fantom-foundation/Aida/state"
	"go.uber.org/mock/gomock"
)

func TestTemporaryArchivePrepper_PreTransactionCreatesArchiveWithMinusOneBlock(t *testing.T) {
	ctrl := gomock.NewController(t)
	db := state.NewMockStateDB(ctrl)
	st := executor.State[*rpc.RequestAndResults]{
		Block:       1,
		Transaction: 1,
		Data: &rpc.RequestAndResults{
			RequestedBlock: 10,
		},
	}
	ctx := &executor.Context{
		State: db,
	}

	db.EXPECT().GetArchiveState(uint64(st.Data.RequestedBlock) - 1)

	ext := MakeTemporaryArchivePrepper()
	err := ext.PreTransaction(st, ctx)
	if err != nil {
		t.Fatalf("unexpected error; %v", err)
	}

}

func TestTemporaryArchivePrepper_PostTransactionReleasesArchive(t *testing.T) {
	ctrl := gomock.NewController(t)
	db := state.NewMockNonCommittableStateDB(ctrl)

	ctx := &executor.Context{
		Archive: db,
	}

	db.EXPECT().Release()

	ext := MakeTemporaryArchivePrepper()
	err := ext.PostTransaction(executor.State[*rpc.RequestAndResults]{}, ctx)
	if err != nil {
		t.Fatalf("unexpected error; %v", err)
	}

}
