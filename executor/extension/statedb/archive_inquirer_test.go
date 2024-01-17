package statedb

import (
	"math/big"
	"slices"
	"testing"
	"time"

	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/executor/extension"
	"github.com/Fantom-foundation/Aida/executor/transaction"
	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/state"
	"github.com/Fantom-foundation/Aida/utils"
	substate "github.com/Fantom-foundation/Substate"
	"github.com/ethereum/go-ethereum/common"
	"go.uber.org/mock/gomock"
)

func TestArchiveInquirer_DisabledIfNoQueryRateIsGiven(t *testing.T) {
	config := utils.Config{}
	ext := MakeArchiveInquirer(&config)
	if _, ok := ext.(extension.NilExtension[transaction.SubstateData]); !ok {
		t.Errorf("inquirer should not be active by default")
	}
}

func TestArchiveInquirer_ReportsErrorIfNoArchiveIsPresent(t *testing.T) {
	ctrl := gomock.NewController(t)
	log := logger.NewMockLogger(ctrl)
	cfg := utils.Config{}
	cfg.ArchiveQueryRate = 100
	ext := makeArchiveInquirer(&cfg, log)

	state := executor.State[transaction.SubstateData]{}
	if err := ext.PreRun(state, nil); err == nil {
		t.Errorf("expected an error, got nothing")
	}
	if err := ext.PostRun(state, nil, nil); err != nil {
		t.Errorf("failed to shut down gracefully, got %v", err)
	}
}

func TestArchiveInquirer_CanStartUpAndShutdownGracefully(t *testing.T) {
	ctrl := gomock.NewController(t)
	log := logger.NewMockLogger(ctrl)
	db := state.NewMockStateDB(ctrl)

	cfg := utils.Config{}
	cfg.ArchiveMode = true
	cfg.ArchiveQueryRate = 100
	ext := makeArchiveInquirer(&cfg, log)

	state := executor.State[transaction.SubstateData]{}
	context := executor.Context{State: db}

	if err := ext.PreRun(state, &context); err != nil {
		t.Errorf("failed PreRun, got %v", err)
	}
	if err := ext.PostRun(state, nil, nil); err != nil {
		t.Errorf("failed to shut down gracefully, got %v", err)
	}
}

func TestArchiveInquirer_RunsRandomTransactionsInBackground(t *testing.T) {
	ctrl := gomock.NewController(t)
	log := logger.NewMockLogger(ctrl)
	db := state.NewMockStateDB(ctrl)
	archive := state.NewMockNonCommittableStateDB(ctrl)

	cfg := utils.Config{}
	cfg.ArchiveMode = true
	cfg.ArchiveQueryRate = 100
	cfg.ArchiveMaxQueryAge = 100
	cfg.ChainID = utils.TestnetChainID

	state := executor.State[transaction.SubstateData]{}
	context := executor.Context{State: db}

	substate1 := makeValidSubstate()
	substate2 := makeValidSubstate()

	db.EXPECT().GetArchiveBlockHeight().AnyTimes().Return(uint64(14), false, nil)
	db.EXPECT().GetArchiveState(uint64(12)).MinTimes(1).Return(archive, nil)
	db.EXPECT().GetArchiveState(uint64(14)).MinTimes(1).Return(archive, nil)

	archive.EXPECT().BeginTransaction(gomock.Any()).MinTimes(1)
	archive.EXPECT().Prepare(gomock.Any(), gomock.Any()).AnyTimes()
	archive.EXPECT().Snapshot().AnyTimes()
	archive.EXPECT().GetBalance(gomock.Any()).AnyTimes().Return(big.NewInt(1000))
	archive.EXPECT().GetNonce(gomock.Any()).AnyTimes().Return(uint64(0))
	archive.EXPECT().SetNonce(gomock.Any(), gomock.Any()).AnyTimes().Return()
	archive.EXPECT().GetCodeHash(gomock.Any()).AnyTimes().Return(common.Hash{})
	archive.EXPECT().SubBalance(gomock.Any(), gomock.Any()).AnyTimes()
	archive.EXPECT().CreateAccount(gomock.Any()).AnyTimes()
	archive.EXPECT().AddBalance(gomock.Any(), gomock.Any()).AnyTimes()
	archive.EXPECT().SetCode(gomock.Any(), gomock.Any()).AnyTimes()
	archive.EXPECT().GetRefund().AnyTimes()
	archive.EXPECT().RevertToSnapshot(gomock.Any()).AnyTimes()
	archive.EXPECT().EndTransaction().AnyTimes()
	archive.EXPECT().Release().MinTimes(1)

	ext := makeArchiveInquirer(&cfg, log)
	if err := ext.PreRun(state, &context); err != nil {
		t.Errorf("failed PreRun, got %v", err)
	}

	// Add two transaction to the pool
	state.Block = 13
	state.Transaction = 0
	state.Data = substate1
	if err := ext.PostTransaction(state, &context); err != nil {
		t.Fatalf("failed to add transaction to pool: %v", err)
	}

	state.Block = 15
	state.Transaction = 0
	state.Data = substate2
	if err := ext.PostTransaction(state, &context); err != nil {
		t.Fatalf("failed to add transaction to pool: %v", err)
	}

	time.Sleep(time.Second)

	if err := ext.PostRun(state, nil, nil); err != nil {
		t.Errorf("failed to shut down gracefully, got %v", err)
	}
}

func makeValidSubstate() transaction.SubstateData {
	// This Substate is a minimal substate that can be successfully processed.
	sub := &substate.Substate{
		Env: &substate.SubstateEnv{
			GasLimit: 100_000_000,
		},
		Message: &substate.SubstateMessage{
			Gas:      100_000,
			GasPrice: big.NewInt(0),
			Value:    big.NewInt(0),
		},
		Result: &substate.SubstateResult{
			GasUsed: 1,
		},
	}
	return transaction.NewOldSubstateData(sub)
}

func TestCircularBuffer_EnforcesSize(t *testing.T) {
	for _, size := range []int{0, 1, 2, 10, 50} {
		buffer := newBuffer[int](size)
		for i := 0; i < 100; i++ {
			want := i
			if i > size {
				want = size
			}
			if got := buffer.Size(); want != got {
				t.Errorf("expected size, wanted %d, got %d", want, got)
			}
			buffer.Add(0)
		}
	}
}

func TestCircularBuffer_GetReturnsValueAtPosition(t *testing.T) {
	buffer := newBuffer[int](3)
	buffer.Add(1)
	buffer.Add(2)
	buffer.Add(3)
	for i := 0; i < buffer.Size(); i++ {
		if want, got := i+1, buffer.Get(i); want != got {
			t.Errorf("unexpected element at position %d: want %d, got %d", i, want, got)
		}
	}
}

func TestCircularBuffer_CyclesThroughContent(t *testing.T) {
	buffer := newBuffer[int](3)
	if want, got := []int{}, buffer.data; !slices.Equal(want, got) {
		t.Errorf("unexpected content, wanted %v, got %v", want, got)
	}

	buffer.Add(1)
	if want, got := []int{1}, buffer.data; !slices.Equal(want, got) {
		t.Errorf("unexpected content, wanted %v, got %v", want, got)
	}
	buffer.Add(2)
	if want, got := []int{1, 2}, buffer.data; !slices.Equal(want, got) {
		t.Errorf("unexpected content, wanted %v, got %v", want, got)
	}
	buffer.Add(3)
	if want, got := []int{1, 2, 3}, buffer.data; !slices.Equal(want, got) {
		t.Errorf("unexpected content, wanted %v, got %v", want, got)
	}
	buffer.Add(4)
	if want, got := []int{4, 2, 3}, buffer.data; !slices.Equal(want, got) {
		t.Errorf("unexpected content, wanted %v, got %v", want, got)
	}
	buffer.Add(5)
	if want, got := []int{4, 5, 3}, buffer.data; !slices.Equal(want, got) {
		t.Errorf("unexpected content, wanted %v, got %v", want, got)
	}
}

func TestThrottler_ProducesEventsInExpectedRate(t *testing.T) {
	const testPeriod = 500 * time.Millisecond
	for _, rate := range []int{5, 10, 100, 1000} {
		throttler := *newThrottler(rate)

		count := 0
		start := time.Now()
		for time.Since(start) < testPeriod {
			if throttler.shouldRunNow() {
				count++
			}
		}

		expected := float64(rate) * float64(testPeriod) / float64(time.Second)
		diff := float64(count) - expected
		if diff > 2 || diff < -2 {
			t.Errorf("failed to reproduce rate %d, did %d events in %v", rate, count, testPeriod)
		}
	}
}
