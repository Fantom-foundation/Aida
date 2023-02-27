package snapshot

import (
	"bytes"
	"context"
	"fmt"
	"math/rand"
	"strconv"
	"testing"
	"time"

	substate "github.com/Fantom-foundation/Substate"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

// mockIteratorSize represents the number of accounts created in the iterator
const mockIteratorSize = 50

type mockSubstate struct {
	addr    common.Address
	storage map[common.Hash]common.Hash
}

type mockSubstateIterator struct {
	list    []mockSubstate
	hashing crypto.KeccakState
	current int
}

func (i *mockSubstateIterator) Next() bool {
	i.current++
	return i.current < len(i.list)
}

func (i *mockSubstateIterator) Address() common.Address {
	if i.current >= 0 && i.current < len(i.list) {
		return i.list[i.current].addr
	}
	return common.Address{}
}

func (i *mockSubstateIterator) Storage() map[common.Hash]common.Hash {
	if i.current >= 0 && i.current < len(i.list) {
		return i.list[i.current].storage
	}
	return map[common.Hash]common.Hash{}
}

func (i *mockSubstateIterator) Value() *substate.Transaction {
	if i.current < 0 || i.current >= len(i.list) {
		panic(fmt.Errorf("invalid iterator position"))
	}

	// we mention each address as many times as we can (6x)
	st := substate.Transaction{
		Block:       uint64(len(i.list)),
		Transaction: i.current,
		Substate: &substate.Substate{
			InputAlloc: map[common.Address]*substate.SubstateAccount{
				i.list[i.current].addr: {Storage: i.list[i.current].storage},
			},
			OutputAlloc: map[common.Address]*substate.SubstateAccount{
				i.list[i.current].addr: {Storage: i.list[i.current].storage},
			},
			Env: &substate.SubstateEnv{
				Coinbase: i.list[i.current].addr,
			},
			Message: &substate.SubstateMessage{
				From: i.list[i.current].addr,
				To:   &i.list[i.current].addr,
			},
			Result: &substate.SubstateResult{
				ContractAddress: i.list[i.current].addr,
			},
		},
	}
	return &st
}

func (i *mockSubstateIterator) Release() {
	i.current = -1 // we do not point to any valid account after the release
}

func (i *mockSubstateIterator) IsKnown(a common.Address) bool {
	for j := 0; j < mockIteratorSize; j++ {
		if bytes.Equal(a.Bytes(), i.list[j].addr.Bytes()) {
			return true
		}
	}
	return false
}

func (i *mockSubstateIterator) IsStored(a common.Hash) bool {
	for j := 0; j < mockIteratorSize; j++ {
		for hash := range i.list[j].storage {
			if bytes.Equal(a.Bytes(), hash.Bytes()) {
				return true
			}
		}

	}
	return false
}

// makeMockIterator creates an instance of mock substate iterator for testing.
func makeMockIterator(t *testing.T) *mockSubstateIterator {
	iter := mockSubstateIterator{
		list:    make([]mockSubstate, mockIteratorSize),
		hashing: crypto.NewKeccakState(),
		current: -1, // we do not point to any valid account at the beginning
	}

	for i := 0; i < mockIteratorSize; i++ {
		pk, err := crypto.GenerateKey()
		if err != nil {
			t.Fatalf("failed test data build; could not create random keys; %s", err.Error())
		}

		rand.Seed(time.Now().UnixNano())
		storageSize := rand.Intn(4)
		ms := mockSubstate{addr: crypto.PubkeyToAddress(pk.PublicKey), storage: make(map[common.Hash]common.Hash, storageSize)}
		for k := 0; k < storageSize; k++ {
			addrHash := crypto.HashData(iter.hashing, []byte(strconv.Itoa(k)))
			ms.storage[addrHash] = addrHash
		}
		iter.list[i] = ms
	}

	return &iter
}

func TestCollectAccounts(t *testing.T) {
	// prep mock iterator
	ti := makeMockIterator(t)

	// collect accounts from the iterator
	visited := make(map[common.Address]int, len(ti.list))
	ac, storage := CollectAccounts(context.Background(), ti, 0, 5)

	//draining storages to avoid deadlock
	go func() {
		for {
			_, ok := <-storage
			if !ok {
				return
			}
		}
	}()

	for {
		ac, open := <-ac
		if !open {
			break
		}

		acc, ok := ac.(common.Address)
		if !ok {
			t.Fatalf("account %s has invalid type %T", acc.String(), ac)
		}

		// increase visited counter to verify number of appearances
		visited[acc]++

		// check if the address is in the mock iterator
		if !ti.IsKnown(acc) {
			t.Fatalf("failed account check; expected to know %s, iterator reported FALSE", acc.String())
		}
	}

	// check if each account has been visited expected number of times
	for a, c := range visited {
		if c != 6 {
			t.Errorf("failed account frequency check; expected 6x account %s, got %dx", a.String(), c)
		}
	}
}

func TestCollectStorages(t *testing.T) {
	// prep mock iterator
	ti := makeMockIterator(t)

	// collect total number of storages
	totalCount := 0
	ac, storage := CollectAccounts(context.Background(), ti, 0, 5)

	//draining accounts to avoid deadlock
	go func() {
		for {
			_, ok := <-ac
			if !ok {
				return
			}
		}
	}()

	for {
		st, open := <-storage
		if !open {
			break
		}

		hash, ok := st.(common.Hash)
		if !ok {
			t.Fatalf("account %s has invalid type %T", hash.String(), st)
		}

		totalCount++

		// check if the hash is in the mock iterator
		if !ti.IsStored(hash) {
			t.Fatalf("failed storage check; expected to know %s, iterator reported FALSE", hash.String())
		}
	}

	count := 0
	for _, mockSS := range ti.list {
		// 2 times count since storage is mentioned in both inInputAlloc andOutputAlloc
		count += 2 * len(mockSS.storage)
	}

	if totalCount != count {
		t.Errorf("number of extracted storages %d doesnt match the originally inserted storages %d", totalCount, count)
	}
}

func TestFilterUniqueAccounts(t *testing.T) {
	// prep mock iterator
	ti := makeMockIterator(t)

	ac, storage := CollectAccounts(context.Background(), ti, 0, 5)

	//draining storages to avoid deadlock
	go func() {
		for {
			_, ok := <-storage
			if !ok {
				return
			}
		}
	}()

	visited := make(map[string]bool, len(ti.list))
	unique := make(chan any, cap(ac))
	go FilterUnique(context.Background(), ac, unique)

	var count int
	for {
		ac, open := <-unique
		if !open {
			break
		}

		acc, ok := ac.(common.Address)
		if !ok {
			t.Fatalf("account %s has invalid type %T", acc.String(), ac)
		}

		// check if the address is in the mock iterator
		if !ti.IsKnown(acc) {
			t.Fatalf("failed account check; expected to know %s, iterator reported FALSE", acc.String())
		}

		_, found := visited[acc.Hex()]
		if found {
			t.Fatalf("failed unique account check; expected to see %s only once, got repeated occurence", acc.String())
		}

		count++
	}

	// do the unique number corresponds with expected
	if count != len(ti.list) {
		t.Errorf("failed account list size check; expected to see %d unique accounts, found %d", len(ti.list), count)
	}
}

func TestWriteAccountAddresses(t *testing.T) {
	// create in-memory database
	db, err := OpenStateDB("")
	if err != nil {
		t.Fatalf("failed test data build; could not create test DB; %s", err.Error())
	}

	// prep mock iterator
	ti := makeMockIterator(t)

	addr, storage := CollectAccounts(context.Background(), ti, uint64(len(ti.list)), 5)

	errAcc := WriteAccounts(context.Background(), addr, db)
	errStorage := WriteAccounts(context.Background(), storage, db)

	err, ok := <-errAcc
	if ok && err != nil {
		t.Fatalf("failed to write accounts; expected no error, got %s", err.Error())
	}

	err, ok = <-errStorage
	if ok && err != nil {
		t.Fatalf("failed to write storage hashes; expected no error, got %s", err.Error())
	}

	// loop all accounts and check their mapping exists in the database
	ti.Release()
	for ti.Next() {
		h := db.AccountAddressToHash(ti.Address())

		adr, err := db.HashToAccountAddress(h)
		if err != nil {
			t.Fatalf("failed hash to address; expected to find hash %s, got error %s", h.String(), err.Error())
		}

		if !bytes.Equal(ti.Address().Bytes(), adr.Bytes()) {
			t.Errorf("failed hash address check; expected to get %s, got %s", ti.Address().String(), adr.String())
		}

		for hash := range ti.Storage() {
			s := db.StorageToHash(hash)

			sh, err := db.HashToStorage(s)
			if err != nil {
				t.Fatalf("failed hash to storage; expected to find hash %s, got error %s", sh.String(), err.Error())
			}

			if !bytes.Equal(hash.Bytes(), sh.Bytes()) {
				t.Errorf("failed hash storage check; expected to get %s, got %s", hash.String(), sh.String())
			}
		}

	}
}
