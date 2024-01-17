package primer

import (
	"bytes"
	"fmt"
	"os"
	"testing"

	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/executor/extension"
	"github.com/Fantom-foundation/Aida/executor/transaction"
	"github.com/Fantom-foundation/Aida/executor/transaction/substate_transaction"
	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/state"
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

func TestStateDbPrimerExtension_PrimingDoesNotTriggerForExistingStateDb(t *testing.T) {
	ctrl := gomock.NewController(t)
	log := logger.NewMockLogger(ctrl)

	cfg := &utils.Config{}
	cfg.SkipPriming = false
	cfg.IsExistingStateDb = true

	log.EXPECT().Warning("Skipping priming due to usage of pre-existing StateDb")

	ext := makeStateDbPrimer[any](cfg, log)

	ext.PreRun(executor.State[any]{}, nil)

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
				err = sDB.Close()
				if err != nil {
					t.Fatalf("failed to close state DB: %v", err)
				}
			}(sDB)

			// Generating randomized world state
			alloc, _ := utils.MakeWorldState(t)
			ws := substate_transaction.NewOldSubstateAlloc(alloc)

			pc := utils.NewPrimeContext(cfg, sDB, log)
			// Priming state DB
			pc.PrimeStateDB(ws, sDB)

			// Checks if state DB was primed correctly
			ws.ForEachAccount(func(addr common.Address, acc transaction.Account) {
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
		log.EXPECT().Noticef("Priming to block %v...", uint64(9)),
	)

	ext.PreRun(executor.State[any]{}, &executor.Context{})
}
