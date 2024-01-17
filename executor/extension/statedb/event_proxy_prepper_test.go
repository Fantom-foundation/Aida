package statedb

import (
	"testing"

	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/state"
	"github.com/Fantom-foundation/Aida/stochastic"
	"github.com/Fantom-foundation/Aida/utils"
	"go.uber.org/mock/gomock"
)

func TestEventProxyPrepper_PreTransactionCreatesEventProxyIfNotAlready(t *testing.T) {
	ctrl := gomock.NewController(t)
	db := state.NewMockStateDB(ctrl)

	path := t.TempDir() + "test_trace"
	cfg := &utils.Config{}
	cfg.Output = path
	cfg.SyncPeriodLength = 1

	ext := MakeEventProxyPrepper[any](cfg)

	ctx := &executor.Context{}
	ctx.State = db

	err := ext.PreTransaction(executor.State[any]{}, ctx)
	if err != nil {
		t.Fatalf("unexpected error; %v", err)
	}

	_, ok := ctx.State.(*stochastic.EventProxy)
	if !ok {
		t.Fatalf("state is not a recorder proxy")
	}
}

func TestEventProxyPrepper_PreTransactionDoesNotCreateEventProxyBecauseItIsAlready(t *testing.T) {
	ctrl := gomock.NewController(t)
	db := state.NewMockStateDB(ctrl)

	path := t.TempDir() + "test_trace"
	cfg := &utils.Config{}
	cfg.Output = path
	cfg.SyncPeriodLength = 1

	ext := MakeEventProxyPrepper[any](cfg)

	ctx := &executor.Context{}
	ctx.State = db

	err := ext.PreRun(executor.State[any]{}, ctx)
	if err != nil {
		t.Fatalf("unexpected error; %v", err)
	}

	originalDb := ctx.State

	err = ext.PreTransaction(executor.State[any]{}, ctx)
	if err != nil {
		t.Fatalf("unexpected error; %v", err)
	}

	if originalDb != ctx.State {
		t.Fatal("pre-transaction must not create new proxy")
	}
}
