package snapshot

import (
	"bytes"
	"context"
	"fmt"
	"math/rand"
	"reflect"
	"testing"

	"github.com/Fantom-foundation/Aida/world-state/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/substate"
)

// TestStateDB_Substate check whether generated substatAlloc is identical to the original database
func TestStateDB_Substate(t *testing.T) {
	// prep source DB
	db, nodes, a2h, s2h := makeTestDB(t)
	defer MustCloseStateDB(db)

	ssDB, err := db.ToSubstateAlloc(context.Background())
	if err != nil {
		t.Fatalf("failed substate test; expected no error, got %s", err.Error())
	}

	for hash, account := range nodes {
		for h, address := range a2h {
			if h == hash {
				err = compare(account, address, ssDB, s2h)
				if err != nil {
					t.Fatalf("%v", err)
				}
				break
			}
		}
	}
}

// TestStateDB_Substate_CtxFail tests ToSubstateAlloc function in the context expiration state
func TestStateDB_Substate_CtxFail(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), cCtxNoTime)
	defer cancel()

	// prep source DB
	db, _, _, _ := makeTestDB(t)
	defer MustCloseStateDB(db)

	_, err := db.ToSubstateAlloc(ctx)
	if err != context.DeadlineExceeded {
		t.Errorf("failed substate test on context expiration; expected DeadlineExceeded error, got %s", errorStr(err))
	}
}

// compare checks one world state account to one in SubstateAlloc
func compare(account types.Account, address common.Address, ssDB substate.SubstateAlloc, s2h map[common.Hash]common.Hash) error {
	ss, found := ssDB[address]
	if !found {
		return fmt.Errorf("failed to find account %s in substate", address)
	}

	if account.Nonce != ss.Nonce {
		return fmt.Errorf("failed account %s in substate has different nonce", address)
	}

	if account.Balance.Cmp(ss.Balance) != 0 {
		return fmt.Errorf("failed account %s in substate has different balance", address)
	}

	if bytes.Compare(account.Code, ss.Code) != 0 {
		return fmt.Errorf("failed account %s in substate has different code", address)
	}

	if len(account.Storage) != len(ss.Storage) {
		return fmt.Errorf("storage sizes did not match; %s - expected %d received %d", address, len(account.Storage), len(ss.Storage))
	}

	for hash, v := range account.Storage {
		us, f := s2h[hash]
		if !f {
			return fmt.Errorf("incorrect translation table for storage hashes")
		}

		v2, f2 := ss.Storage[us]
		if !f2 {
			return fmt.Errorf("incorrect substate storage data")
		}

		if v != v2 {
			return fmt.Errorf("storage values not matching")
		}
	}
	return nil
}

// TestStateDB_Substate_Skipping check whether addresses with missing translations are omitted
func TestStateDB_Substate_Skipping(t *testing.T) {
	// prep source DB
	db, nodes, a2h, s2h, errExpected := makeIncompleteTestDB(t)
	defer MustCloseStateDB(db)

	ssDB, err := db.ToSubstateAlloc(context.Background())
	if err != nil {
		t.Fatalf("failed substate test; expected no error, got %s", err.Error())
	}

	errorCount := 0
	for hash, account := range nodes {
		for h, address := range a2h {
			if h == hash {
				err = compare(account, address, ssDB, s2h)
				if err != nil {
					errorCount++
				}
				break
			}
		}
	}

	// count accounts which were not inserted into ssDB because of missing a2h
	errorCount += len(nodes) - len(ssDB)

	if errorCount != errExpected {
		t.Fatalf("number of expected errors %d did not match actual amount of errors %d", errExpected, errorCount)
	}
}

func getRandomAccount(t *testing.T, db *StateDB, a2h map[common.Hash]common.Address) *types.Account {
	a2hKeys := reflect.ValueOf(a2h).MapKeys()
	keyToRandomAccount := a2hKeys[rand.Intn(len(a2hKeys))].Interface().(common.Hash)
	acc, err := db.AccountByHash(keyToRandomAccount)
	if err != nil {
		t.Fatalf("unable to retrieve random account from dummy database; %s", err)
	}
	return acc
}
