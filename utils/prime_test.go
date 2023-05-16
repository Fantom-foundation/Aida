package utils

import (
	"bytes"
	"fmt"
	"os"
	"testing"

	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/state"
)

// TestStatedb_PrimeStateDB tests priming fresh state DB with randomized world state data
func TestPrime_PrimeStateDB(t *testing.T) {
	log := logger.NewLogger("Warning", "TestPrimeStateDB")
	for _, tc := range getStatedbTestCases() {
		t.Run(fmt.Sprintf("DB variant: %s; shadowImpl: %s; archive variant: %s", tc.variant, tc.shadowImpl, tc.archiveVariant), func(t *testing.T) {
			cfg := makeTestConfig(tc)

			// Initialization of state DB
			sDB, sDbDir, err := PrepareStateDB(cfg)
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
			ws, _ := makeWorldState(t)

			pc := NewPrimeContext(cfg, log)
			// Priming state DB
			pc.PrimeStateDB(ws, sDB)

			// Checks if state DB was primed correctly
			for key, account := range ws {
				if sDB.GetBalance(key).Cmp(account.Balance) != 0 {
					t.Fatalf("failed to prime account balance; Is: %v; Should be: %v", sDB.GetBalance(key), account.Balance)
				}

				if sDB.GetNonce(key) != account.Nonce {
					t.Fatalf("failed to prime account nonce; Is: %v; Should be: %v", sDB.GetNonce(key), account.Nonce)
				}

				if bytes.Compare(sDB.GetCode(key), account.Code) != 0 {
					t.Fatalf("failed to prime account code; Is: %v; Should be: %v", sDB.GetCode(key), account.Code)
				}

				for sKey, sValue := range account.Storage {
					if sDB.GetState(key, sKey) != sValue {
						t.Fatalf("failed to prime account storage; Is: %v; Should be: %v", sDB.GetState(key, sKey), sValue)
					}
				}
			}
		})
	}
}
