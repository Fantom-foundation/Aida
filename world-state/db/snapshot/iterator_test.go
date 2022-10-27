// Package snapshot implements database interfaces for the world state manager.
package snapshot

import (
	"context"
	"github.com/Fantom-foundation/Aida/world-state/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/crypto"
	"math/big"
	"math/rand"
	"testing"
	"time"
)

// testDBSize is the number of account we put into the test DB
const testDBSize = 250
const testAccountStorageSize = 10

// makeTestDB primes test DB for the other tests using randomized account data.
func makeTestDB(t *testing.T) (*StateDB, map[common.Hash]types.Account, map[common.Hash]common.Address, map[common.Hash]common.Hash) {
	// create in-memory database
	db, err := OpenStateDB("")
	if err != nil {
		t.Fatalf("failed test data build; could not create test DB; %s", err.Error())
	}

	var hashing = crypto.NewKeccakState()

	var ta = make(map[common.Hash]types.Account, testDBSize)
	var adh = make(map[common.Hash]common.Address, testDBSize)
	var sdh = make(map[common.Hash]common.Hash, testDBSize*testAccountStorageSize)

	// TestAccounts represents the test set for accounts.
	for i := 0; i < testDBSize; i++ {
		pk, err := crypto.GenerateKey()
		if err != nil {
			t.Fatalf("failed test data build; could not create random keys; %s", err.Error())
		}

		addr := crypto.PubkeyToAddress(pk.PublicKey)
		hash := crypto.HashData(hashing, addr.Bytes())
		adh[hash] = addr

		// write the mapping into the test DB
		err = db.PutHashToAccountAddress(hash, addr)
		if err != nil {
			t.Fatalf("failed test data build; could not write hash to address mapping; %s", err.Error())
		}

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
		for j := 0; j < testAccountStorageSize; j++ {
			_, err = rand.Read(buffer)
			if err != nil {
				t.Fatalf("failed test data build; can not generate random code; %s", err.Error())
			}
			h := common.BytesToHash(buffer)
			k := crypto.HashData(hashing, h.Bytes())

			sdh[k] = h
			// write the mapping into the test DB
			err = db.PutHashToStorage(k, h)
			if err != nil {
				t.Fatalf("failed test data build; could not write storage hash to hash mapping; %s", err.Error())
			}

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

	return db, ta, adh, sdh
}

// makeIncompleteTestDB on top of TestDB adds accounts and storages without original addresses in indexing tables
// adding one account to db and nodes without inserting record to a2h
// adding one storage to one random existing account  without inserting record to s2h
func makeIncompleteTestDB(t *testing.T) (*StateDB, map[common.Hash]types.Account, map[common.Hash]common.Address, map[common.Hash]common.Hash, int) {
	// prep source DB
	db, nodes, a2h, s2h := makeTestDB(t)

	// number of records missing in a2h and s2h combined
	missing := 0

	rand.Seed(time.Now().UnixNano())

	// add storage to existing account
	acc := getRandomAccount(t, db, a2h)
	var hashing = crypto.NewKeccakState()
	// fill the storage map
	buffer := make([]byte, 32)
	_, err := rand.Read(buffer)
	if err != nil {
		t.Fatalf("failed test data build; can not generate random code; %s", err.Error())
	}

	h := common.BytesToHash(buffer)
	k := crypto.HashData(hashing, h.Bytes())
	acc.Storage[k] = h
	nodes[acc.Hash] = *acc
	err = db.PutAccount(acc)
	if err != nil {
		t.Fatalf("unable to update account in database; %s", err.Error())
	}
	missing++

	//put new account
	_, err = rand.Read(buffer)
	if err != nil {
		t.Fatalf("failed test data build; can not generate random code; %s", err.Error())
	}
	hash := common.BytesToHash(buffer)
	newAcc := types.Account{
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
	nodes[hash] = newAcc
	err = db.PutAccount(&newAcc)
	if err != nil {
		t.Fatalf("unable to insert account in database; %s", err.Error())
	}
	missing++

	return db, nodes, a2h, s2h, missing
}

func TestStateDB_NewAccountIterator(t *testing.T) {
	dl, ok := t.Deadline()
	if !ok {
		t.Fatalf("test deadline exceeded")
	}

	// compare the DB to itself
	ctx, cancel := context.WithDeadline(context.Background(), dl)
	defer cancel()

	db, ta, _, _ := makeTestDB(t)
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
