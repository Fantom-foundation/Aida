package statedb

import (
	"os"
	"testing"

	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/state"
	"github.com/Fantom-foundation/Aida/state/proxy"
	"github.com/Fantom-foundation/Aida/utils"
	substate "github.com/Fantom-foundation/Substate"
	"go.uber.org/mock/gomock"
)

func TestTemporaryProxyRecorderPrepper_PreTransactionCreatesRecorderProxy(t *testing.T) {
	path := t.TempDir() + "test_trace"
	cfg := &utils.Config{}
	cfg.TraceFile = path
	cfg.SyncPeriodLength = 1

	p := MakeTemporaryProxyRecorderPrepper(cfg)

	ctx := &executor.Context{}

	err := p.PreRun(executor.State[*substate.Substate]{}, ctx)
	if err != nil {
		t.Fatalf("unexpected error; %v", err)
	}

	err = p.PreTransaction(executor.State[*substate.Substate]{}, ctx)
	if err != nil {
		t.Fatalf("unexpected error; %v", err)
	}

	_, ok := ctx.State.(*proxy.RecorderProxy)
	if !ok {
		t.Fatalf("state is not a recorder proxy")
	}

	// close the file gracefully
	err = p.PostRun(executor.State[*substate.Substate]{}, ctx, nil)
	if err != nil {
		t.Fatalf("unexpected error; %v", err)
	}
}

func TestProxyRecorderPrepper_PreBlockWritesABeginBlockOperation(t *testing.T) {
	path := t.TempDir() + "test_trace"
	cfg := &utils.Config{}
	cfg.TraceFile = path
	cfg.SyncPeriodLength = 1

	p := makeTemporaryProxyRecorderPrepper(cfg)

	ctx := &executor.Context{}

	err := p.PreRun(executor.State[*substate.Substate]{}, ctx)
	if err != nil {
		t.Fatalf("unexpected error; %v", err)
	}

	err = p.PreTransaction(executor.State[*substate.Substate]{}, ctx)
	if err != nil {
		t.Fatalf("unexpected error; %v", err)
	}

	err = p.PreBlock(executor.State[*substate.Substate]{}, ctx)
	if err != nil {
		t.Fatalf("unexpected error; %v", err)
	}

	// close the file gracefully
	p.rCtx.Close()

	stats, err := os.Stat(path)
	if err != nil {
		t.Fatalf("unexpected error; %v", err)
	}

	if stats.Size() <= 0 {
		t.Fatalf("size of trace file is 0")
	}

}

func TestProxyRecorderPrepper_PostBlockWritesAnEndBlockOperation(t *testing.T) {
	path := t.TempDir() + "test_trace"
	cfg := &utils.Config{}
	cfg.TraceFile = path
	cfg.SyncPeriodLength = 1

	p := makeTemporaryProxyRecorderPrepper(cfg)

	ctx := &executor.Context{}

	err := p.PreRun(executor.State[*substate.Substate]{}, ctx)
	if err != nil {
		t.Fatalf("unexpected error; %v", err)
	}

	err = p.PreTransaction(executor.State[*substate.Substate]{}, ctx)
	if err != nil {
		t.Fatalf("unexpected error; %v", err)
	}

	err = p.PostBlock(executor.State[*substate.Substate]{}, ctx)
	if err != nil {
		t.Fatalf("unexpected error; %v", err)
	}

	// close the file gracefully
	p.rCtx.Close()

	stats, err := os.Stat(path)
	if err != nil {
		t.Fatalf("unexpected error; %v", err)
	}

	if stats.Size() <= 0 {
		t.Fatalf("size of trace file is 0")
	}

}

func TestProxyRecorderPrepper_PostRunWritesAnEndSynchPeriodOperation(t *testing.T) {
	path := t.TempDir() + "test_trace"
	cfg := &utils.Config{}
	cfg.TraceFile = path
	cfg.SyncPeriodLength = 1

	p := MakeTemporaryProxyRecorderPrepper(cfg)

	ctx := &executor.Context{}

	err := p.PreRun(executor.State[*substate.Substate]{}, ctx)
	if err != nil {
		t.Fatalf("unexpected error; %v", err)
	}

	// close the file gracefully
	err = p.PostRun(executor.State[*substate.Substate]{}, ctx, nil)
	if err != nil {
		t.Fatalf("unexpected error; %v", err)
	}

	stats, err := os.Stat(path)
	if err != nil {
		t.Fatalf("unexpected error; %v", err)
	}

	if stats.Size() <= 0 {
		t.Fatalf("size of trace file is 0")
	}

}

func TestProxyRecorderPrepper_PreTransactionCreatesNewLoggerProxy(t *testing.T) {
	ctrl := gomock.NewController(t)
	db := state.NewMockStateDB(ctrl)

	cfg := &utils.Config{}
	cfg.LogLevel = "critical"

	ctx := new(executor.Context)
	ctx.State = db

	ext := MakeTemporaryProxyRecorderPrepper(cfg)

	// ctx.State is not yet a RecorderProxy hence PreTransaction assigns it
	err := ext.PreTransaction(executor.State[*substate.Substate]{}, ctx)
	if err != nil {
		t.Fatalf("pre-transaction failed; %v", err)
	}

	if _, ok := ctx.State.(*proxy.RecorderProxy); !ok {
		t.Fatal("db must be of type RecorderProxy!")
	}
}

func TestProxyRecorderPrepper_PreTransactionDoesNotCreateNewLoggerProxy(t *testing.T) {
	ctrl := gomock.NewController(t)
	db := state.NewMockStateDB(ctrl)

	cfg := &utils.Config{}
	cfg.TraceFile = t.TempDir() + "test_trace"
	cfg.LogLevel = "critical"
	cfg.SyncPeriodLength = 1

	ctx := new(executor.Context)
	ctx.State = db

	ext := MakeTemporaryProxyRecorderPrepper(cfg)

	// first call PreTransaction to assign the proxy
	err := ext.PreTransaction(executor.State[*substate.Substate]{}, ctx)
	if err != nil {
		t.Fatalf("pre-transaction failed; %v", err)
	}

	// save original state to make sure next call to PreTransaction will not have changed the ctx.State
	originalDb := ctx.State

	// then make sure it is not re-assigned again
	err = ext.PreTransaction(executor.State[*substate.Substate]{}, ctx)
	if err != nil {
		t.Fatalf("pre-transaction failed; %v", err)
	}

	if originalDb != ctx.State {
		t.Fatal("db must not be be changed!")
	}
}
