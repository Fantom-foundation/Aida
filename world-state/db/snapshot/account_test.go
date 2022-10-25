package snapshot

import (
	"bytes"
	"context"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/substate"
	"testing"
)

// mockIteratorSize represents the number of accounts created in the iterator
const mockIteratorSize = 50

type mockSubstateIterator struct {
	list    []common.Address
	hashing crypto.KeccakState
	current int
}

func (i *mockSubstateIterator) Next() bool {
	i.current++
	return i.current < len(i.list)
}

func (i *mockSubstateIterator) Address() common.Address {
	if i.current >= 0 && i.current < len(i.list) {
		return i.list[i.current]
	}
	return common.Address{}
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
				i.list[i.current]: {},
			},
			OutputAlloc: map[common.Address]*substate.SubstateAccount{
				i.list[i.current]: {},
			},
			Env: &substate.SubstateEnv{
				Coinbase: i.list[i.current],
			},
			Message: &substate.SubstateMessage{
				From: i.list[i.current],
				To:   &i.list[i.current],
			},
			Result: &substate.SubstateResult{
				ContractAddress: i.list[i.current],
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
		if bytes.Equal(a.Bytes(), i.list[j].Bytes()) {
			return true
		}
	}
	return false
}

// makeMockIterator creates an instance of mock substate iterator for testing.
func makeMockIterator(t *testing.T) *mockSubstateIterator {
	iter := mockSubstateIterator{
		list:    make([]common.Address, mockIteratorSize),
		hashing: crypto.NewKeccakState(),
		current: -1, // we do not point to any valid account at the beginning
	}

	for i := 0; i < mockIteratorSize; i++ {
		pk, err := crypto.GenerateKey()
		if err != nil {
			t.Fatalf("failed test data build; could not create random keys; %s", err.Error())
		}

		iter.list[i] = crypto.PubkeyToAddress(pk.PublicKey)
	}

	return &iter
}

func TestCollectAccounts(t *testing.T) {
	// prep mock iterator
	ti := makeMockIterator(t)

	// collect accounts from the iterator
	visited := make(map[common.Address]int, len(ti.list))
	ac := CollectAccounts(context.Background(), ti, 0, 5)

	for {
		ac, open := <-ac
		if !open {
			break
		}

		// increase visited counter to verify number of appearances
		visited[ac]++

		// check if the address is in the mock iterator
		if !ti.IsKnown(ac) {
			t.Fatalf("failed account check; expected to know %s, iterator reported FALSE", ac.String())
		}
	}

	// check if each account has been visited expected number of times
	for a, c := range visited {
		if c != 6 {
			t.Errorf("failed account frequency check; expected 6x account %s, got %dx", a.String(), c)
		}
	}
}

func TestFilterUniqueAccounts(t *testing.T) {
	// prep mock iterator
	ti := makeMockIterator(t)

	ac := CollectAccounts(context.Background(), ti, 0, 5)

	visited := make(map[common.Address]bool, len(ti.list))
	unique := make(chan common.Address, cap(ac))
	go FilterUniqueAccounts(context.Background(), ac, unique)

	var count int
	for {
		ac, open := <-unique
		if !open {
			break
		}

		// check if the address is in the mock iterator
		if !ti.IsKnown(ac) {
			t.Fatalf("failed account check; expected to know %s, iterator reported FALSE", ac.String())
		}

		_, found := visited[ac]
		if found {
			t.Fatalf("failed unique account check; expected to see %s only once, got repeated occurence", ac.String())
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

	err = WriteAccountAddresses(context.Background(), CollectAccounts(context.Background(), ti, uint64(len(ti.list)), 5), db)
	if err != nil {
		t.Fatalf("failed to write accounts; expected no error, got %s", err.Error())
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
	}
}
