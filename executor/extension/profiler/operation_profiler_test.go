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

package profiler

import (
	"fmt"
	"math/big"
	"math/rand"
	"testing"
	"time"

	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/executor/extension"
	"github.com/Fantom-foundation/Aida/state"
	"github.com/Fantom-foundation/Aida/tracer/operation"
	"github.com/Fantom-foundation/Aida/utils"
	"github.com/Fantom-foundation/Aida/utils/analytics"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
	gomock "go.uber.org/mock/gomock"
	"golang.org/x/exp/maps"
)

// general helper functions for testing

func assertExactlyEqual[T comparable](t *testing.T, a T, b T) {
	if a != b {
		t.Errorf("%v != %v", a, b)
	}
}

func getTotalOpCount(a *analytics.IncrementalAnalytics) int {
	var count uint64 = 0
	for _, stat := range a.Iterate() {
		count += stat.GetCount()
	}
	return int(count)
}

// This generates exactly one call per operation and test if the following are true:
// - That op profiler correctly proxies any StateDB implementation
// - This is repeated for each depth level -> interval, block and txcontext
// - Call each function exactly once
//   - Make explicit the fact that some StateDB are not proxied (see black list below)

func TestOperationProfiler_WithEachOpOnce(t *testing.T) {
	name := "OperationProfiler EachOpOnce"
	cfg := &utils.Config{
		Profile:         true,
		ProfileDepth:    int(TransactionLevel),
		First:           uint64(1),
		Last:            uint64(1),
		ProfileInterval: uint64(1),
	}

	t.Run(name, func(t *testing.T) {
		ext, ok := MakeOperationProfiler[any](cfg).(*operationProfiler[any])
		if !ok {
			t.Fatalf("Failed to create OperationProfiler despite valid config")
		}

		ac := len(ext.anlts)
		if ac != int(ext.depth)+1 {
			t.Fatalf("Number of Analytics should be equal to depth configured. Configured %d, but there's %d analytics", ext.depth+1, ac)
		}

		ctrl := gomock.NewController(t)
		mockStateDB := state.NewMockStateDB(ctrl)
		mockCtx := executor.Context{State: mockStateDB}
		prepareMockStateDbOnce(mockStateDB)

		// PRE BLOCK
		ext.PreRun(executor.State[any]{}, &mockCtx)
		ext.PreBlock(executor.State[any]{Block: int(cfg.First)}, nil)
		ext.PreTransaction(executor.State[any]{Transaction: int(0)}, nil)

		// call each function once as a single tx in a single block
		funcs := getStateDbFuncs(mockCtx.State)
		for _, f := range funcs {
			f()
		}

		// Check here before the stats are reset by the extension
		totalOpCount := make([]int, int(ext.depth)+1)
		ops := operation.CreateIdLabelMap()

		// These are purposely not implemented, will be blacklisted here
		notImplemented := make([]bool, len(ops))
		for _, a := range []byte{14, 18, 21, 22, 23, 29, 42, 49, 50, 51, 53} {
			notImplemented[a] = true
		}

		for _, op := range maps.Keys(ops) {
			if notImplemented[op] {
				continue
			}

			for depth := IntervalLevel; depth <= ext.depth; depth++ {
				c := ext.anlts[int(depth)].GetCount(op)
				if c != 1 {
					t.Errorf("Op %d:%s occurs %d times, expecting exactly 1", op, ops[op], c)
				}
				totalOpCount[depth] += int(c)
			}
		}

		for depth := IntervalLevel; depth <= ext.depth; depth++ {
			if totalOpCount[int(depth)] != len(funcs) {
				t.Errorf("Seen %d ops even though we have %d", totalOpCount[int(depth)], len(funcs))
			}
		}

		// POST BLOCK
		ext.PostTransaction(executor.State[any]{Transaction: int(0)}, nil)
		ext.PostBlock(executor.State[any]{Block: int(cfg.First)}, nil)
		ext.PostRun(executor.State[any]{}, nil, nil)

	})
}

// This generate random amount of operations call per block and test if the following are true:
// - That profiler correctly proxies any StateDB implementation
// - If analytics is properly reset when it should
// - If interval is correct (and that it's 0-index), and is updated when it should
// - That the amount of operation generated are logged on analytics agrees
func TestOperationProfiler_WithRandomInput(t *testing.T) {
	type argument struct {
		name           string
		seed           int64 // -1 = don't use my seed, random something for me
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
		{args: argument{"50-100/block, interval=1", 258, 50, 100, 1, 100, 1}},
		{args: argument{"50-100/block, several intervals", 258, 50, 100, 1, 100, 10}},
		{args: argument{"50-100/block, 1 interval", 258, 50, 100, 1, 100, 10000}},
		{args: argument{"50-100/block, first=last", 258, 50, 100, 100, 100, 12}},
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
			if test.args.seed != 0 && test.args.seed != -1 {
				r = rand.New(rand.NewSource(test.args.seed))
			} else {
				r = rand.New(rand.NewSource(time.Now().UnixNano()))
			}

			ext, ok := MakeOperationProfiler[any](cfg).(*operationProfiler[any])
			if !ok {
				t.Fatalf("Failed to create Operation Profiler despite valid config")
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
					totalSeenOpCount += getTotalOpCount(ext.anlts[0])
					intervalGeneratedOpCount = 0
				}

				ext.PreBlock(executor.State[any]{Block: int(b)}, nil)
				if b > intervalEnd {
					// make sure that the stats is reset
					if getTotalOpCount(ext.anlts[0]) != 0 {
						t.Errorf("Analytics should have been reset but found %d ops", getTotalOpCount(ext.anlts[0]))
					}
				}

				// ensure 0 index
				if ext.interval.Start() != cfg.First && ext.interval.Start()%cfg.ProfileInterval != 0 {
					t.Fatalf("Interval is not using 0-index, found %d", ext.interval.Start()%cfg.ProfileInterval)
				}

				gap := test.args.maxOpsPerBlock - test.args.minOpsPerBlock
				generatedOpCount := test.args.minOpsPerBlock + r.Intn(gap)

				for o := 0; o < generatedOpCount; o++ {
					getRandomStateDbFunc(mockCtx.State, r)()
					totalGeneratedOpCount++
					intervalGeneratedOpCount++
				}

				ext.PostBlock(executor.State[any]{Block: int(b)}, nil)

				// check that amount of ops seen eqals to amount of ops generated within this interval
				if getTotalOpCount(ext.anlts[0]) != intervalGeneratedOpCount {
					t.Errorf("[Interval] Seen %d ops, but generated %d ops", getTotalOpCount(ext.anlts[0]), intervalGeneratedOpCount)
				}
			}

			// check that amount of ops seen equals to amount of ops generated
			totalSeenOpCount += getTotalOpCount(ext.anlts[0])
			if totalSeenOpCount != totalGeneratedOpCount {
				t.Errorf("[Total] Seen %d ops, but generated %d ops", totalSeenOpCount, totalGeneratedOpCount)
			}

			ext.PostRun(executor.State[any]{}, nil, nil)
		})
	}
}

// Originally this would test if interval <= 0 or if last < first.
// This is deemed unneccessary since there is a separate validation for config, so they are never malformed.
// Since we need to test disabled profiling, this is retained.
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
			t.Fatalf("OperationProfiler is enabled although configuration not set or malformed")
		}
	}
}

////////////////////////////////////////////////////////////////////////////////////////

// HELPER FUNCTIONS

// contains a list of all possible mocked operations to be tested.
func getStateDbFuncs(db state.StateDB) []func() {
	mockAddress := common.HexToAddress("0x00000F1")
	mockHash := common.BigToHash(big.NewInt(0))
	return []func(){
		func() { db.CreateAccount(mockAddress) },
		func() { db.SubBalance(mockAddress, uint256.NewInt(0), tracing.BalanceChangeUnspecified) },
		func() { db.AddBalance(mockAddress, uint256.NewInt(0), tracing.BalanceChangeUnspecified) },
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
		func() { db.GetTransientState(mockAddress, mockHash) },
		func() { db.SetState(mockAddress, mockHash, mockHash) },
		func() { db.SetTransientState(mockAddress, mockHash, mockHash) },
		func() { db.SelfDestruct(mockAddress) },
		func() { db.HasSelfDestructed(mockAddress) },
		func() { db.Exist(mockAddress) },
		func() { db.Empty(mockAddress) },
		func() {
			db.Prepare(
				params.Rules{},
				mockAddress,
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
		func() { db.GetLogs(mockHash, uint64(0), mockHash) },
		func() { db.AddPreimage(mockHash, []byte{0}) },
		func() { db.SetTxContext(mockHash, 0) },
		func() { db.Finalise(false) },
		func() { db.IntermediateRoot(false) },
		func() { db.Commit(uint64(0), false) },
		func() { db.Close() },
	}
}

// MockStateDB must be prepared before used (it needs to know how many time each function will be called).
// This functions tell MockStateDB to expect any number of calls (0 or more) to each of the functions (for randomized test)
func prepareMockStateDb(m *state.MockStateDB) {
	m.EXPECT().CreateAccount(gomock.Any()).AnyTimes()
	m.EXPECT().SubBalance(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	m.EXPECT().AddBalance(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
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
	m.EXPECT().GetTransientState(gomock.Any(), gomock.Any()).AnyTimes()
	m.EXPECT().SetState(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	m.EXPECT().SetTransientState(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	m.EXPECT().SelfDestruct(gomock.Any()).AnyTimes()
	m.EXPECT().HasSelfDestructed(gomock.Any()).AnyTimes()
	m.EXPECT().Exist(gomock.Any()).AnyTimes()
	m.EXPECT().Empty(gomock.Any()).AnyTimes()
	m.EXPECT().Prepare(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
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
	m.EXPECT().GetLogs(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	m.EXPECT().AddPreimage(gomock.Any(), gomock.Any()).AnyTimes()
	m.EXPECT().SetTxContext(gomock.Any(), gomock.Any()).AnyTimes()
	m.EXPECT().Finalise(gomock.Any()).AnyTimes()
	m.EXPECT().IntermediateRoot(gomock.Any()).AnyTimes()
	m.EXPECT().Commit(gomock.Any(), gomock.Any()).AnyTimes()
	m.EXPECT().Close().AnyTimes()
}

// MockStateDB must be prepared before used (it needs to know how many time each function will be called.
// This functions tell MockStateDB to expect exactly one call to each of the possible functions
func prepareMockStateDbOnce(m *state.MockStateDB) {
	m.EXPECT().CreateAccount(gomock.Any())
	m.EXPECT().SubBalance(gomock.Any(), gomock.Any(), gomock.Any())
	m.EXPECT().AddBalance(gomock.Any(), gomock.Any(), gomock.Any())
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
	m.EXPECT().GetTransientState(gomock.Any(), gomock.Any())
	m.EXPECT().SetState(gomock.Any(), gomock.Any(), gomock.Any())
	m.EXPECT().SetTransientState(gomock.Any(), gomock.Any(), gomock.Any())
	m.EXPECT().SelfDestruct(gomock.Any())
	m.EXPECT().HasSelfDestructed(gomock.Any())
	m.EXPECT().Exist(gomock.Any())
	m.EXPECT().Empty(gomock.Any())
	m.EXPECT().Prepare(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any())
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
	m.EXPECT().GetLogs(gomock.Any(), gomock.Any(), gomock.Any())
	m.EXPECT().AddPreimage(gomock.Any(), gomock.Any())
	m.EXPECT().SetTxContext(gomock.Any(), gomock.Any())
	m.EXPECT().Finalise(gomock.Any())
	m.EXPECT().IntermediateRoot(gomock.Any())
	m.EXPECT().Commit(gomock.Any(), gomock.Any())
	m.EXPECT().Close()
}

// Helper function to randomize an operation to be called
func getRandomStateDbFunc(db state.StateDB, r *rand.Rand) func() {
	funcs := getStateDbFuncs(db)
	funcCount := len(funcs)
	return funcs[r.Intn(funcCount)]
}
