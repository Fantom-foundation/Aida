package profiler

import (
	"fmt"
	"math/big"
	"math/rand"
	"os"
	"testing"
	"time"

	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/executor/extension"
	"github.com/Fantom-foundation/Aida/state"
	"github.com/Fantom-foundation/Aida/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	gomock "go.uber.org/mock/gomock"
)

func assertExactlyEqual[T comparable](t *testing.T, a T, b T) {
	if a != b {
		t.Errorf("%v != %v", a, b)
	}
}

func getStateDbFuncs(db state.StateDB) []func() {
	mockAddress := common.HexToAddress("0x00000F1")
	mockHash := common.BigToHash(big.NewInt(0))
	return []func(){
		func() { db.CreateAccount(mockAddress) },
		func() { db.SubBalance(mockAddress, big.NewInt(0)) },
		func() { db.AddBalance(mockAddress, big.NewInt(0)) },
		func() { db.GetBalance(mockAddress) },
		func() { db.GetNonce(mockAddress) },
		func() { db.SetNonce(mockAddress, 0) },
		func() { db.GetCodeHash(mockAddress) },
		func() { db.GetCode(mockAddress) },
		func() { db.SetCode(mockAddress, []byte{0}) },
		func() { db.GetCodeSize(mockAddress) },
		func() { db.AddRefund(0) },
		func() { db.SubRefund(0) },
		func() { db.GetRefund() },
		func() { db.GetCommittedState(mockAddress, mockHash) },
		func() { db.GetState(mockAddress, mockHash) },
		func() { db.SetState(mockAddress, mockHash, mockHash) },
		func() { db.Suicide(mockAddress) },
		func() { db.HasSuicided(mockAddress) },
		func() { db.Exist(mockAddress) },
		func() { db.Empty(mockAddress) },
		func() {
			db.PrepareAccessList(
				mockAddress,
				&mockAddress,
				[]common.Address{},
				[]types.AccessTuple{{mockAddress, []common.Hash{mockHash}}},
			)
		},
		func() { db.AddAddressToAccessList(mockAddress) },
		func() { db.AddressInAccessList(mockAddress) },
		func() { db.SlotInAccessList(mockAddress, mockHash) },
		func() { db.AddSlotToAccessList(mockAddress, mockHash) },
		func() { db.Snapshot() },
		func() { db.RevertToSnapshot(0) },
		func() { db.BeginTransaction(0) },
		func() { db.EndTransaction() },
		func() { db.BeginBlock(0) },
		func() { db.EndBlock() },
		func() { db.BeginSyncPeriod(0) },
		func() { db.EndSyncPeriod() },
		func() { db.AddLog(nil) },
		func() { db.GetLogs(mockHash, mockHash) },
		func() { db.AddPreimage(mockHash, []byte{0}) },
		func() { db.ForEachStorage(mockAddress, func(h1, h2 common.Hash) bool { return false }) },
		func() { db.Prepare(mockHash, 0) },
		func() { db.Finalise(false) },
		func() { db.IntermediateRoot(false) },
		func() { db.Commit(false) },
		func() { db.Close() },
	}
}

func prepareMockStateDb(m *state.MockStateDB) {
	m.EXPECT().CreateAccount(gomock.Any()).AnyTimes()
	m.EXPECT().SubBalance(gomock.Any(), gomock.Any()).AnyTimes()
	m.EXPECT().AddBalance(gomock.Any(), gomock.Any()).AnyTimes()
	m.EXPECT().GetBalance(gomock.Any()).AnyTimes()
	m.EXPECT().GetNonce(gomock.Any()).AnyTimes()
	m.EXPECT().SetNonce(gomock.Any(), gomock.Any()).AnyTimes()
	m.EXPECT().GetCodeHash(gomock.Any()).AnyTimes()
	m.EXPECT().GetCode(gomock.Any()).AnyTimes()
	m.EXPECT().SetCode(gomock.Any(), gomock.Any()).AnyTimes()
	m.EXPECT().GetCodeSize(gomock.Any()).AnyTimes()
	m.EXPECT().AddRefund(gomock.Any()).AnyTimes()
	m.EXPECT().SubRefund(gomock.Any()).AnyTimes()
	m.EXPECT().GetRefund().AnyTimes()
	m.EXPECT().GetCommittedState(gomock.Any(), gomock.Any()).AnyTimes()
	m.EXPECT().GetState(gomock.Any(), gomock.Any()).AnyTimes()
	m.EXPECT().SetState(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	m.EXPECT().Suicide(gomock.Any()).AnyTimes()
	m.EXPECT().HasSuicided(gomock.Any()).AnyTimes()
	m.EXPECT().Exist(gomock.Any()).AnyTimes()
	m.EXPECT().Empty(gomock.Any()).AnyTimes()
	m.EXPECT().PrepareAccessList(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	m.EXPECT().AddAddressToAccessList(gomock.Any()).AnyTimes()
	m.EXPECT().AddressInAccessList(gomock.Any()).AnyTimes()
	m.EXPECT().SlotInAccessList(gomock.Any(), gomock.Any()).AnyTimes()
	m.EXPECT().AddSlotToAccessList(gomock.Any(), gomock.Any()).AnyTimes()
	m.EXPECT().Snapshot().AnyTimes()
	m.EXPECT().RevertToSnapshot(gomock.Any()).AnyTimes()
	m.EXPECT().BeginTransaction(gomock.Any()).AnyTimes()
	m.EXPECT().EndTransaction().AnyTimes()
	m.EXPECT().BeginBlock(gomock.Any()).AnyTimes()
	m.EXPECT().EndBlock().AnyTimes()
	m.EXPECT().BeginSyncPeriod(gomock.Any()).AnyTimes()
	m.EXPECT().EndSyncPeriod().AnyTimes()
	m.EXPECT().AddLog(gomock.Any()).AnyTimes()
	m.EXPECT().GetLogs(gomock.Any(), gomock.Any()).AnyTimes()
	m.EXPECT().AddPreimage(gomock.Any(), gomock.Any()).AnyTimes()
	m.EXPECT().ForEachStorage(gomock.Any(), gomock.Any()).AnyTimes()
	m.EXPECT().Prepare(gomock.Any(), gomock.Any()).AnyTimes()
	m.EXPECT().Finalise(gomock.Any()).AnyTimes()
	m.EXPECT().IntermediateRoot(gomock.Any()).AnyTimes()
	m.EXPECT().Commit(gomock.Any()).AnyTimes()
	m.EXPECT().Close().AnyTimes()
}

func prepareMockStateDbOnce(m *state.MockStateDB) {
	m.EXPECT().CreateAccount(gomock.Any())
	m.EXPECT().SubBalance(gomock.Any(), gomock.Any())
	m.EXPECT().AddBalance(gomock.Any(), gomock.Any())
	m.EXPECT().GetBalance(gomock.Any())
	m.EXPECT().GetNonce(gomock.Any())
	m.EXPECT().SetNonce(gomock.Any(), gomock.Any())
	m.EXPECT().GetCodeHash(gomock.Any())
	m.EXPECT().GetCode(gomock.Any())
	m.EXPECT().SetCode(gomock.Any(), gomock.Any())
	m.EXPECT().GetCodeSize(gomock.Any())
	m.EXPECT().AddRefund(gomock.Any())
	m.EXPECT().SubRefund(gomock.Any())
	m.EXPECT().GetRefund()
	m.EXPECT().GetCommittedState(gomock.Any(), gomock.Any())
	m.EXPECT().GetState(gomock.Any(), gomock.Any())
	m.EXPECT().SetState(gomock.Any(), gomock.Any(), gomock.Any())
	m.EXPECT().Suicide(gomock.Any())
	m.EXPECT().HasSuicided(gomock.Any())
	m.EXPECT().Exist(gomock.Any())
	m.EXPECT().Empty(gomock.Any())
	m.EXPECT().PrepareAccessList(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any())
	m.EXPECT().AddAddressToAccessList(gomock.Any())
	m.EXPECT().AddressInAccessList(gomock.Any())
	m.EXPECT().SlotInAccessList(gomock.Any(), gomock.Any())
	m.EXPECT().AddSlotToAccessList(gomock.Any(), gomock.Any())
	m.EXPECT().Snapshot()
	m.EXPECT().RevertToSnapshot(gomock.Any())
	m.EXPECT().BeginTransaction(gomock.Any())
	m.EXPECT().EndTransaction()
	m.EXPECT().BeginBlock(gomock.Any())
	m.EXPECT().EndBlock()
	m.EXPECT().BeginSyncPeriod(gomock.Any())
	m.EXPECT().EndSyncPeriod()
	m.EXPECT().AddLog(gomock.Any())
	m.EXPECT().GetLogs(gomock.Any(), gomock.Any())
	m.EXPECT().AddPreimage(gomock.Any(), gomock.Any())
	m.EXPECT().ForEachStorage(gomock.Any(), gomock.Any())
	m.EXPECT().Prepare(gomock.Any(), gomock.Any())
	m.EXPECT().Finalise(gomock.Any())
	m.EXPECT().IntermediateRoot(gomock.Any())
	m.EXPECT().Commit(gomock.Any())
	m.EXPECT().Close()
}

func getRandomStateDbFunc(db state.StateDB, r *rand.Rand) func() {
	funcs := getStateDbFuncs(db)
	funcCount := len(funcs)
	return funcs[r.Intn(funcCount)]
}

func suppressStdout(f func()) {
	tmp := os.Stdout
	os.Stdout = nil
	f()
	os.Stdout = tmp
}

func TestOperationProfiler_WithEachOpOnce(t *testing.T) {
	name := "OperationProfiler EachOpOnce"
	cfg := &utils.Config{
		Profile:         true,
		First:           uint64(1),
		Last:            uint64(1),
		ProfileInterval: uint64(1),
	}

	t.Run(name, func(t *testing.T) {
		ext, ok := MakeOperationProfiler[any](cfg).(*operationProfiler[any])
		if !ok {
			t.Fatalf("Fail to create Operation Profiler despite valid config")
		}

		ctrl := gomock.NewController(t)
		mockStateDB := state.NewMockStateDB(ctrl)
		mockCtx := executor.Context{State: mockStateDB}
		prepareMockStateDbOnce(mockStateDB)

		ext.PreRun(executor.State[any]{}, &mockCtx)
		funcs := getStateDbFuncs(mockCtx.State)
		for b := int(cfg.First); b <= int(cfg.Last); b += 1 + rand.Intn(3) {
			suppressStdout(func() {
				ext.PreBlock(executor.State[any]{Block: int(b)}, nil)
			})
			for _, f := range funcs {
				f()
			}
			ext.PostBlock(executor.State[any]{Block: int(b)}, nil)

		}
		suppressStdout(func() {
			ext.PostRun(executor.State[any]{}, nil, nil)
		})

		totalOpCount := 0
		ops := ext.stats.GetOpOrder()

		// These are purposely not implemented, will be blacklisted here
		notImplemented := make([]bool, len(ops))
		for _, a := range []byte{14, 18, 21, 22, 23, 29} {
			notImplemented[a] = true
		}

		for _, op := range ops {
			if notImplemented[op] {
				continue
			}

			s := ext.stats.GetStatByOpId(op)
			if s.Frequency != 1 {
				t.Errorf("op %s occurs %d times, expecting exactly 1", s.Label, s.Frequency)
			}
			totalOpCount += int(s.Frequency)
		}
		if totalOpCount != len(funcs) {
			t.Errorf("Seen %d ops even though we have %d", totalOpCount, len(funcs))
		}

	})
}

func TestOperationProfiler_WithRandomInput(t *testing.T) {
	type argument struct {
		name           string
		seed           int64
		minOpsPerBlock int
		maxOpsPerBlock int
		first          int
		last           int
		interval       int
	}
	type result struct{}
	type testcase struct {
		args argument
		want result
	}

	tests := []testcase{
		{args: argument{"50-100/block, interval=1", -1, 50, 100, 1, 100, 1}},
		{args: argument{"50-100/block, several intervals", -1, 50, 100, 1, 100, 10}},
		{args: argument{"50-100/block, seeded several intervals", 258, 50, 100, 4000, 8000, 1000}},
		{args: argument{"50-100/block, 1 interval", -1, 50, 100, 1, 100, 10000}},
		{args: argument{"50-100/block, first=last", -1, 50, 100, 100, 100, 12}},
	}

	for _, test := range tests {
		name := fmt.Sprintf("OperationProfiler Random [%s]", test.args.name)
		cfg := &utils.Config{
			Profile:         true,
			First:           uint64(test.args.first),
			Last:            uint64(test.args.last),
			ProfileInterval: uint64(test.args.interval),
		}

		t.Run(name, func(t *testing.T) {
			// initialize rng with seed
			var r *rand.Rand
			if test.args.seed != 0 && test.args.seed != 1 {
				r = rand.New(rand.NewSource(test.args.seed))
			} else {
				r = rand.New(rand.NewSource(time.Now().UnixNano()))
			}

			ext, ok := MakeOperationProfiler[any](cfg).(*operationProfiler[any])
			if !ok {
				t.Fatalf("Fail to create Operation Profiler despite valid config")
			}

			ctrl := gomock.NewController(t)
			mockStateDB := state.NewMockStateDB(ctrl)
			mockCtx := executor.Context{State: mockStateDB}
			prepareMockStateDb(mockStateDB)

			totalSeenOpCount, totalGeneratedOpCount := 0, 0
			intervalGeneratedOpCount := 0

			intervalStart := test.args.first - (test.args.first % test.args.interval)
			intervalEnd := intervalStart + test.args.interval - 1

			ext.PreRun(executor.State[any]{}, &mockCtx)
			for b := test.args.first; b <= test.args.last; b += 1 + r.Intn(3) {

				if b > intervalEnd {
					intervalStart = intervalEnd + 1
					intervalEnd += test.args.interval
					totalSeenOpCount += ext.stats.GetTotalOpFreq()
					intervalGeneratedOpCount = 0
				}

				suppressStdout(func() {
					ext.PreBlock(executor.State[any]{Block: int(b)}, nil)
				})

				if b > intervalEnd {
					// make sure that the stats is reset
					if ext.stats.GetTotalOpFreq() != 0 {
						t.Fatalf("Should be reset but found %d ops", ext.stats.GetTotalOpFreq())
					}

					// ensure 0 index (skips the initial interval where first is intervalStart)
					if ext.interval.Start()%cfg.ProfileInterval != 0 {
						t.Fatalf("interval is not using 0-index, found %d", ext.interval.Start()%cfg.ProfileInterval)
					}
				}

				gap := test.args.maxOpsPerBlock - test.args.minOpsPerBlock
				generatedOpCount := test.args.minOpsPerBlock + r.Intn(gap)

				for o := 0; o < generatedOpCount; o++ {
					getRandomStateDbFunc(mockCtx.State, r)()
					totalGeneratedOpCount++
					intervalGeneratedOpCount++
				}

				ext.PostBlock(executor.State[any]{Block: int(b)}, nil)

				// check that ext tracks last seen block number correctly
				if ext.lastProcessedBlock != uint64(b) {
					t.Fatalf("Last seen block number was %d, actual last seen block %d", ext.lastProcessedBlock, uint64(b))
				}
				// check that amount of ops seen eqals to amount of ops generated within this interval
				if ext.stats.GetTotalOpFreq() != intervalGeneratedOpCount {
					t.Fatalf("[Interval] Seen %d ops, but generated %d ops", ext.stats.GetTotalOpFreq(), intervalGeneratedOpCount)
				}
			}

			suppressStdout(func() {
				ext.PostRun(executor.State[any]{}, nil, nil)
			})

			// check that last seen block number is within boundary
			if ext.lastProcessedBlock > uint64(test.args.last) {
				t.Errorf("Last seen block number was %d, more than last boundary %d.", ext.lastProcessedBlock, test.args.last)
			}
			// check that amount of ops seen equals to amount of ops generated
			totalSeenOpCount += ext.stats.GetTotalOpFreq()
			if totalSeenOpCount != totalGeneratedOpCount {
				t.Errorf("[Total] Seen %d ops, but generated %d ops", totalSeenOpCount, totalGeneratedOpCount)
			}
		})
	}
}

// Originally this would test if interval <= 0 or if last < first.
// This is deemed unneccessary since there is a separate validation for config, so they are never malformed.
func TestOperationProfiler_WithMalformedConfig(t *testing.T) {
	type argument struct {
		profile  bool
		first    uint64
		last     uint64
		interval uint64
	}

	type testcase struct {
		args argument
	}

	tests := []testcase{
		{args: argument{false, 0, 1000, 100}},
	}

	for _, test := range tests {
		ext := MakeOperationProfiler[any](&utils.Config{
			Profile:         test.args.profile,
			First:           test.args.first,
			Last:            test.args.last,
			ProfileInterval: test.args.interval,
		})

		if _, ok := ext.(extension.NilExtension[any]); !ok {
			t.Errorf("profiler is enabled although configuration not set or malformed")
		}
	}
}
