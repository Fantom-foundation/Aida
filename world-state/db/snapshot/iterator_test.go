// Package snapshot implements database interfaces for the world state manager.
package snapshot

import (
	"context"
	"github.com/Fantom-foundation/Aida-Testing/world-state/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/crypto"
	"math/big"
	"math/rand"
	"testing"
)

// makeTestDB primes test DB for the other tests using randomized account data.
func makeTestDB(t *testing.T) (*StateDB, map[common.Hash]types.Account) {
	// create in-memory database
	db, err := OpenStateDB("")
	if err != nil {
		t.Fatalf("failed test data build; could not create test DB; %s", err.Error())
	}

	var hashing = crypto.NewKeccakState()

	// TestAccounts represents the test set for accounts.
	var ta = map[common.Hash]types.Account{}
	for i := 0; i < 5; i++ {
		pk, err := crypto.GenerateKey()
		if err != nil {
			t.Fatalf("failed test data build; could not create random keys; %s", err.Error())
		}

		addr := crypto.PubkeyToAddress(pk.PublicKey)
		hash := crypto.HashData(hashing, addr.Bytes())

		// add this account to the map
		acc := types.Account{
			Hash:    hash,
			Storage: map[common.Hash]common.Hash{},
			Code:    make([]byte, rand.Intn(2048)),
			Account: state.Account{
				Nonce:    uint64(rand.Int63()),
				Balance:  big.NewInt(rand.Int63()),
				Root:     common.Hash{},
				CodeHash: common.Hash{}.Bytes(),
			},
		}

		// fill the code with random data
		_, err = rand.Read(acc.Code)
		if err != nil {
			t.Fatalf("failed test data build; can not generate random code; %s", err.Error())
		}
		acc.CodeHash = crypto.HashData(hashing, acc.Code).Bytes()

		// fill the storage map
		buffer := make([]byte, 32)
		for j := 0; j < 10; j++ {
			_, err = rand.Read(buffer)
			if err != nil {
				t.Fatalf("failed test data build; can not generate random code; %s", err.Error())
			}
			k := crypto.HashData(hashing, buffer)

			_, err = rand.Read(buffer)
			if err != nil {
				t.Fatalf("failed test data build; can not generate random code; %s", err.Error())
			}

			acc.Storage[k] = crypto.HashData(hashing, buffer)
		}

		// add this account to the DB
		ta[hash] = acc
		err = db.PutAccount(&acc)
		if err != nil {
			t.Fatalf("failed test data build; could not add account; %s", err.Error())
		}
	}

	return db, ta
}

func TestStateDB_NewAccountIterator(t *testing.T) {
	dl, ok := t.Deadline()
	if !ok {
		t.Fatalf("test deadline exceeded")
	}

	// compare the DB to itself
	ctx, cancel := context.WithDeadline(context.Background(), dl)
	defer cancel()

	db, ta := makeTestDB(t)
	iter := db.NewAccountIterator(ctx)
	defer iter.Release()

	var count int
	for iter.Next() {
		if iter.Error() != nil {
			t.Errorf("iterator failed; expected no error, got %s", iter.Error().Error())
			break
		}

		count++
		key := common.Hash{}
		key.SetBytes(iter.Key())

		sa, ok := ta[key]
		if !ok {
			t.Errorf("failed account key; expected to find account key, key not found")
			break
		}

		sb := iter.Value()
		if !sa.IsIdentical(sb) {
			err := sa.IsDifferent(sb)
			t.Errorf("failed account check; expected identical account, found %s", err.Error())
			break
		}
	}

	// the number of iterations should match the number of items in the account set
	if count != len(ta) {
		t.Errorf("failed accounts iterated; expected to iterate %dx, iterated %dx", len(ta), count)
	}

	// we release the iterator; the subsequent deferred iterator release should not fail later
	iter.Release()
}
