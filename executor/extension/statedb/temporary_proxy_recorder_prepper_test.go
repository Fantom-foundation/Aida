package statedb

import (
	"os"
	"testing"

	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/state/proxy"
	"github.com/Fantom-foundation/Aida/utils"
	substate "github.com/Fantom-foundation/Substate"
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
