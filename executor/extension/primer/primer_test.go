package primer

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"testing"

	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/executor/extension"
	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/state"
	"github.com/Fantom-foundation/Aida/txcontext"
	substatecontext "github.com/Fantom-foundation/Aida/txcontext/substate"
	"github.com/Fantom-foundation/Aida/utils"
	"github.com/ethereum/go-ethereum/common"
	"go.uber.org/mock/gomock"
)

func TestStateDbPrimerExtension_NoPrimerIsCreatedIfDisabled(t *testing.T) {
	cfg := &utils.Config{}
	cfg.SkipPriming = true

	ext := MakeStateDbPrimer[any](cfg)
	if _, ok := ext.(extension.NilExtension[any]); !ok {
		t.Errorf("Primer is enabled although not set in configuration")
	}

}

func TestStateDbPrimerExtension_PrimingDExistingStateDbMissingDbInfo(t *testing.T) {
	ctrl := gomock.NewController(t)
	log := logger.NewMockLogger(ctrl)

	cfg := &utils.Config{}
	cfg.IsExistingStateDb = true

	ext := makeStateDbPrimer[any](cfg, log)

	expected := errors.New("cannot read state db info; failed to read statedb_info.json; open statedb_info.json: no such file or directory")

	err := ext.PreRun(executor.State[any]{}, nil)
	if err.Error() != expected.Error() {
		t.Errorf("Priming should fail if db info is missing; got: %v; expected: %v", err, expected)
	}
}

func TestStateDbPrimerExtension_PrimingDoesTriggerForNonExistingStateDb(t *testing.T) {
	ctrl := gomock.NewController(t)
	log := logger.NewMockLogger(ctrl)

	cfg := &utils.Config{}
	cfg.SkipPriming = false
	cfg.StateDbSrc = ""
	cfg.First = 2

	gomock.InOrder(
		log.EXPECT().Infof("Update buffer size: %v bytes", cfg.UpdateBufferSize),
		log.EXPECT().Noticef("Priming from block %v...", uint64(0)),
		log.EXPECT().Noticef("Priming to block %v...", cfg.First-1),
	)

	ext := makeStateDbPrimer[any](cfg, log)

	ext.PreRun(executor.State[any]{}, &executor.Context{})
}

func TestStateDbPrimerExtension_AttemptToPrimeBlockZeroDoesNotFail(t *testing.T) {
	ctrl := gomock.NewController(t)
	log := logger.NewMockLogger(ctrl)

	cfg := &utils.Config{}
	cfg.SkipPriming = false
	cfg.StateDbSrc = ""
	cfg.First = 0

	ext := makeStateDbPrimer[any](cfg, log)

	err := ext.PreRun(executor.State[any]{}, &executor.Context{})
	if err != nil {
		t.Errorf("priming should not happen hence should not fail")
	}
}

// TestStatedb_PrimeStateDB tests priming fresh state DB with randomized world state data
func TestPrime_PrimeStateDB(t *testing.T) {
	log := logger.NewLogger("Warning", "TestPrimeStateDB")
	for _, tc := range utils.GetStateDbTestCases() {
		t.Run(fmt.Sprintf("DB variant: %s; shadowImpl: %s; archive variant: %s", tc.Variant, tc.ShadowImpl, tc.ArchiveVariant), func(t *testing.T) {
			cfg := utils.MakeTestConfig(tc)

			// Initialization of state DB
			sDB, sDbDir, err := utils.PrepareStateDB(cfg)
			defer os.RemoveAll(sDbDir)

			if err != nil {
				t.Fatalf("failed to create state DB: %v", err)
			}

			// Closing of state DB
			defer func(sDB state.StateDB) {
				err = state.CloseCarmenDbTestContext(sDB)
				if err != nil {
					t.Fatalf("cannot close carmen test context; %v", err)
				}
			}(sDB)

			// Generating randomized world state
			alloc, _ := utils.MakeWorldState(t)
			ws := substatecontext.NewWorldState(alloc)

			pc := utils.NewPrimeContext(cfg, sDB, 0, log)
			// Priming state DB
			err = pc.PrimeStateDB(ws, sDB)
			if err != nil {
				t.Fatal(err)
			}

			err = state.BeginCarmenDbTestContext(sDB)
			if err != nil {
				t.Fatal(err)
			}

			// Checks if state DB was primed correctly
			ws.ForEachAccount(func(addr common.Address, acc txcontext.Account) {

				if sDB.GetBalance(addr).Cmp(acc.GetBalance()) != 0 {
					t.Fatalf("failed to prime account balance; Is: %v; Should be: %v", sDB.GetBalance(addr), acc.GetBalance())
				}

				if sDB.GetNonce(addr) != acc.GetNonce() {
					t.Fatalf("failed to prime account nonce; Is: %v; Should be: %v", sDB.GetNonce(addr), acc.GetNonce())
				}

				if bytes.Compare(sDB.GetCode(addr), acc.GetCode()) != 0 {
					t.Fatalf("failed to prime account code; Is: %v; Should be: %v", sDB.GetCode(addr), acc.GetCode())
				}

				acc.ForEachStorage(func(keyHash common.Hash, valueHash common.Hash) {
					if sDB.GetState(addr, keyHash) != valueHash {
						t.Fatalf("failed to prime account storage; Is: %v; Should be: %v", sDB.GetState(addr, keyHash), valueHash)
					}
				})

			})

		})
	}
}

func TestStateDbPrimerExtension_UserIsInformedAboutRandomPriming(t *testing.T) {
	ctrl := gomock.NewController(t)
	log := logger.NewMockLogger(ctrl)

	cfg := &utils.Config{}
	cfg.SkipPriming = false
	cfg.StateDbSrc = ""
	cfg.First = 10
	cfg.PrimeRandom = true
	cfg.RandomSeed = 111
	cfg.PrimeThreshold = 10
	cfg.UpdateBufferSize = 1024

	ext := makeStateDbPrimer[any](cfg, log)

	gomock.InOrder(
		log.EXPECT().Infof("Randomized Priming enabled; Seed: %v, threshold: %v", int64(111), 10),
		log.EXPECT().Infof("Update buffer size: %v bytes", uint64(1024)),
		log.EXPECT().Noticef("Priming from block %v...", uint64(0)),
		log.EXPECT().Noticef("Priming to block %v...", uint64(9)),
	)

	ext.PreRun(executor.State[any]{}, &executor.Context{})
}

// make sure that the stateDb contains data from both the first and the second priming
func TestStateDbPrimerExtension_ContinuousPrimingFromExistingDb(t *testing.T) {
	log := logger.NewLogger("Warning", "TestPrimeStateDB")
	for _, tc := range utils.GetStateDbTestCases() {
		t.Run(fmt.Sprintf("DB variant: %s; shadowImpl: %s; archive variant: %s", tc.Variant, tc.ShadowImpl, tc.ArchiveVariant), func(t *testing.T) {
			cfg := utils.MakeTestConfig(tc)

			// Initialization of state DB
			sDB, sDbDir, err := utils.PrepareStateDB(cfg)
			defer os.RemoveAll(sDbDir)

			if err != nil {
				t.Fatalf("failed to create state DB: %v", err)
			}

			// Generating randomized world state
			alloc, _ := utils.MakeWorldState(t)
			ws := substatecontext.NewWorldState(alloc)

			pc := utils.NewPrimeContext(cfg, sDB, 0, log)
			// Priming state DB
			err = pc.PrimeStateDB(ws, sDB)
			if err != nil {
				t.Fatalf("failed to prime state DB: %v", err)
			}

			// Checks if state DB was primed correctly
			ws.ForEachAccount(func(addr common.Address, acc txcontext.Account) {
				if sDB.GetBalance(addr).Cmp(acc.GetBalance()) != 0 {
					t.Fatalf("failed to prime account balance; Is: %v; Should be: %v", sDB.GetBalance(addr), acc.GetBalance())
				}

				if sDB.GetNonce(addr) != acc.GetNonce() {
					t.Fatalf("failed to prime account nonce; Is: %v; Should be: %v", sDB.GetNonce(addr), acc.GetNonce())
				}

				if bytes.Compare(sDB.GetCode(addr), acc.GetCode()) != 0 {
					t.Fatalf("failed to prime account code; Is: %v; Should be: %v", sDB.GetCode(addr), acc.GetCode())
				}

				acc.ForEachStorage(func(keyHash common.Hash, valueHash common.Hash) {
					if sDB.GetState(addr, keyHash) != valueHash {
						t.Fatalf("failed to prime account storage; Is: %v; Should be: %v", sDB.GetState(addr, keyHash), valueHash)
					}
				})
			})

			rootHash := sDB.GetHash()
			// Closing of state DB
			err = sDB.Close()
			if err != nil {
				t.Fatalf("failed to close state DB: %v", err)
			}

			cfg.StateDbSrc = sDbDir
			// Call for json creation and writing into it
			err = utils.WriteStateDbInfo(cfg.StateDbSrc, cfg, 2, rootHash)
			if err != nil {
				t.Fatalf("failed to write into DB info json file: %v", err)
			}

			// Initialization of state DB
			sDB2, sDbDir2, err := utils.PrepareStateDB(cfg)
			defer os.RemoveAll(sDbDir2)
			if err != nil {
				t.Fatalf("failed to create state DB2: %v", err)
			}

			defer func() {
				// Closing of state DB
				err = sDB2.Close()
				if err != nil {
					t.Fatalf("failed to close state DB: %v", err)
				}
			}()

			// Checks if state DB was primed correctly
			ws.ForEachAccount(func(addr common.Address, acc txcontext.Account) {
				if sDB2.GetBalance(addr).Cmp(acc.GetBalance()) != 0 {
					t.Fatalf("failed to prime account balance; Is: %v; Should be: %v", sDB2.GetBalance(addr), acc.GetBalance())
				}

				if sDB2.GetNonce(addr) != acc.GetNonce() {
					t.Fatalf("failed to prime account nonce; Is: %v; Should be: %v", sDB2.GetNonce(addr), acc.GetNonce())
				}

				if bytes.Compare(sDB2.GetCode(addr), acc.GetCode()) != 0 {
					t.Fatalf("failed to prime account code; Is: %v; Should be: %v", sDB2.GetCode(addr), acc.GetCode())
				}

				acc.ForEachStorage(func(keyHash common.Hash, valueHash common.Hash) {
					if sDB2.GetState(addr, keyHash) != valueHash {
						t.Fatalf("failed to prime account storage; Is: %v; Should be: %v", sDB2.GetState(addr, keyHash), valueHash)
					}
				})
			})

			// Generating randomized world state
			alloc2, _ := utils.MakeWorldState(t)
			ws2 := substatecontext.NewWorldState(alloc2)

			pc2 := utils.NewPrimeContext(cfg, sDB2, 1, log)
			// Priming state DB
			err = pc2.PrimeStateDB(ws2, sDB2)
			if err != nil {
				t.Fatalf("failed to prime state DB2: %v", err)
			}

			// Checks if state DB was primed correctly
			ws2.ForEachAccount(func(addr common.Address, acc txcontext.Account) {
				if sDB2.GetBalance(addr).Cmp(acc.GetBalance()) != 0 {
					t.Fatalf("failed to prime account balance; Is: %v; Should be: %v", sDB2.GetBalance(addr), acc.GetBalance())
				}

				if sDB2.GetNonce(addr) != acc.GetNonce() {
					t.Fatalf("failed to prime account nonce; Is: %v; Should be: %v", sDB2.GetNonce(addr), acc.GetNonce())
				}

				if bytes.Compare(sDB2.GetCode(addr), acc.GetCode()) != 0 {
					t.Fatalf("failed to prime account code; Is: %v; Should be: %v", sDB2.GetCode(addr), acc.GetCode())
				}

				acc.ForEachStorage(func(keyHash common.Hash, valueHash common.Hash) {
					if sDB2.GetState(addr, keyHash) != valueHash {
						t.Fatalf("failed to prime account storage; Is: %v; Should be: %v", sDB2.GetState(addr, keyHash), valueHash)
					}
				})
			})
		})
	}
}
