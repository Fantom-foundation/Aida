package snapshot

import (
	"bytes"
	"context"
	"github.com/Fantom-foundation/Aida/world-state/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/substate"
	"testing"
)

// TestStateDB_Substate check whether generated substatAlloc is identical to the original database
func TestStateDB_Substate(t *testing.T) {
	// prep source DB
	db, nodes, a2h, s2h := makeTestDB(t)
	defer MustCloseStateDB(db)

	ssDB, err := db.SubstateAlloc(context.Background())
	if err != nil {
		t.Fatalf("failed substate test; expected no error, got %s", err.Error())
	}

	for hash, account := range nodes {
		for h, address := range a2h {
			if h == hash {
				compare(t, account, address, ssDB, s2h)
				break
			}
		}
	}

}

// compare compares one world state account to one in substateAlloc
func compare(t *testing.T, account types.Account, address common.Address, ssDB substate.SubstateAlloc, s2h map[common.Hash]common.Hash) {
	ss, found := ssDB[address]
	if !found {
		t.Fatalf("failed to find account %s in substate", address)
	}

	if account.Nonce != ss.Nonce {
		t.Fatalf("failed account %s in substate has different nonce", address)
	}

	if account.Balance.Cmp(ss.Balance) != 0 {
		t.Fatalf("failed account %s in substate has different balance", address)
	}

	if bytes.Compare(account.Code, ss.Code) != 0 {
		t.Fatalf("failed account %s in substate has different code", address)
	}

	if len(account.Storage) != len(ss.Storage) {
		t.Fatalf("storage sizes did not match; %s - expected %d received %d", address, len(account.Storage), len(ss.Storage))
	}

	for hash, v := range account.Storage {
		us, f := s2h[hash]
		if !f {
			t.Fatalf("incorrect translation table for storage hashes")
		}

		v2, f2 := ss.Storage[us]
		if !f2 {
			t.Fatalf("incorrect substate storage data")
		}

		if v != v2 {
			t.Fatalf("storage values not matching")
		}
	}
}
