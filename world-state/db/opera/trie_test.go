package opera

import (
	"context"
	"fmt"
	"github.com/Fantom-foundation/Aida/world-state/types"
	"github.com/Fantom-foundation/lachesis-base/kvdb"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
	"math/big"
	"math/rand"
	"os"
	"testing"
	"time"
)

// testDBSize defines how many test accounts will be injected into the test trie
const (
	testDBSize      = 500
	numberOfWorkers = 20
)

// makeTestTrie creates a testing MPT trie filled with random accounts
// Returns state DB, root hash of state trie, map of testing accounts and map of account hashes -> account addresses
func makeTestTrie(t *testing.T, store kvdb.Store) (state.Database, common.Hash, map[common.Hash]types.Account, map[common.Hash]common.Address) {
	// open the source trie DB
	stateDB := OpenStateDB(store)

	// try to create empty MPT
	stateTrie, err := stateDB.OpenTrie(common.Hash{})
	opened := stateTrie != nil && err == nil
	if !opened {
		t.Fatalf("failed test data build; could not open empty stateTrie; %s", err.Error())
	}

	// create test accounts
	ta, h2a := makeTestAccounts(t, stateDB)

	// create state trie
	sth := buildTrie(t, ta, h2a, stateTrie)

	// returns structure representing new MPT and state trie DB
	return stateDB, sth, ta, h2a
}

// makeTestAccounts method creates randomized testing accounts
// returns generated accounts with randomized data and one account which is not a contract
// and map of account hashes -> account addresses
func makeTestAccounts(t *testing.T, stateDB state.Database) (map[common.Hash]types.Account, map[common.Hash]common.Address) {
	var ta = make(map[common.Hash]types.Account, testDBSize)
	var hash2addr = make(map[common.Hash]common.Address, testDBSize)
	var hashing = crypto.NewKeccakState()

	// create randomized accounts
	for i := 0; i < testDBSize; i++ {
		// generate account public key
		pk, err := crypto.GenerateKey()
		if err != nil {
			t.Fatalf("failed test data build; could not create random keys; %s", err.Error())
		}

		// generate account address
		addr := crypto.PubkeyToAddress(pk.PublicKey)
		hash := crypto.HashData(hashing, addr.Bytes())
		hash2addr[hash] = addr

		// create account
		acc := types.Account{
			Hash: hash,
			Code: []byte{},
		}
		acc.Nonce = rand.Uint64()
		acc.Balance = big.NewInt(rand.Int63())
		acc.Root = ethTypes.EmptyRootHash
		acc.CodeHash = types.EmptyCodeHash.Bytes()

		// quarter of the accounts are going to represent as contracts
		if i%4 == 0 {
			// Initialize empty storage
			acc.Storage = map[common.Hash]common.Hash{}

			// generate account code
			ch := makeAccountCode(t, &acc, stateDB)
			acc.CodeHash = ch

			// generate account storage
			srh := makeAccountStorage(t, &acc, stateDB)
			acc.Root = srh
		}

		// add the account to the map
		ta[hash] = acc
	}

	return ta, hash2addr
}

// makeAccountCode method creates account code, fills it with randomized data and store it into db
// returns bytes slice representing code hash
func makeAccountCode(t *testing.T, account *types.Account, stateDB state.Database) []byte {
	hashing := crypto.NewKeccakState()

	// make code container
	account.Code = make([]byte, rand.Intn(2048))

	// fill the code with random data
	_, err := rand.Read(account.Code)
	if err != nil {
		t.Fatalf("failed test data build; can not generate randomized code; %s", err.Error())
	}

	// create code hash
	ch := crypto.HashData(hashing, account.Code)

	// store code into db
	rawdb.WriteCode(stateDB.TrieDB().DiskDB(), ch, account.Code)

	// return code hash represented by bytes slice
	return ch.Bytes()
}

// makeAccountStorage method creates account storage Trie and fills it with data
// returns root hash of newly created account storage trie
func makeAccountStorage(t *testing.T, account *types.Account, stateDB state.Database) common.Hash {
	hashing := crypto.NewKeccakState()

	// try to open account storage trie
	st, err := stateDB.OpenStorageTrie(account.Hash, common.Hash{})
	if err != nil {
		t.Fatalf("failed test data build; could not open empty storage trie; %s", err.Error())
	}

	var rh common.Hash
	buffer := make([]byte, 32)

	// generate randomized data
	for j := 0; j < 10; j++ {
		// generate key
		_, err = rand.Read(buffer)
		if err != nil {
			t.Fatalf("failed test data build; can not generate random storage key; %s", err.Error())
		}
		k := crypto.HashData(hashing, buffer)

		// generate randomized storage value
		var sv []byte
		sv, err = generateStorageValue()

		if err != nil {
			t.Fatalf("failed test data build; can not generate random storage value; %s", err.Error())
		}

		// store storage value without prefix
		account.Storage[k] = common.BytesToHash(sv[1:])

		// try to update storage trie
		err = st.TryUpdate(buffer, sv)
		if err != nil {
			t.Fatalf("failed test data build; can not update storage trie; %s", err.Error())
		}

		// try to commit all the changes
		rh, err = st.Commit(nil)
		if err != nil {
			if err != nil {
				t.Fatalf("failed test data build; can not commit storage changes; %s", err.Error())
			}
		}
	}

	// returns root hash of committed records
	return rh
}

// generateStorageValue creates randomized byte slice representing storage value
func generateStorageValue() ([]byte, error) {
	// seed the PRNG
	rand.Seed(time.Now().UnixNano())

	//get randomized storage value length
	rangeLower := 5
	rangeUpper := 20
	l := rangeLower + rand.Intn(rangeUpper-rangeLower+1)

	// create byte array and set first index containing information about length
	var b = make([]byte, 32)
	b[0] = byte(0x80 + l)

	// randomize content
	_, err := rand.Read(b[1 : l+1])
	if err != nil {
		return nil, err
	}

	// return slice from byte array
	return b[0 : l+1], nil
}

// buildTrie constructs state trie
// returns root hash of trie
func buildTrie(t *testing.T, ta map[common.Hash]types.Account, adh map[common.Hash]common.Address, stateTrie state.Trie) common.Hash {
	// iterate over slice of accounts
	for hash, account := range ta {
		// encode account hash and all the account data
		address, f := adh[hash]
		if !f {
			t.Fatalf("not found")
		}
		acc, err := rlp.EncodeToBytes(account.Account)
		if err != nil {
			t.Fatalf("failed test data build; could not encode account; %s", err.Error())
		}

		// try to update trie with encoded data
		err = stateTrie.TryUpdate(address.Bytes(), acc)
		if err != nil {
			t.Fatalf("failed test data build; could not update trie; %s", err.Error())
		}
	}

	// try to commit all the changes and get root hash
	rh, err := stateTrie.Commit(nil)
	if err != nil {
		t.Fatalf("failed test data build; could not commit trie; %s", err.Error())
	}

	return rh
}

func TestLoadedAccountsValidity(t *testing.T) {
	store, _ := Connect("ldb", os.TempDir(), "test")
	defer func(store kvdb.Store) {
		err := store.Close()
		if err != nil {
			t.Fatalf("failed test data build; could not close store; %s", err.Error())
		}
	}(store)

	stateDB, sth, ta, _ := makeTestTrie(t, store)
	inAccounts, inErrors := LoadAccounts(context.Background(), stateDB, sth, numberOfWorkers)
	counter := 0

	for {
		account, ok := <-inAccounts

		if !ok {
			break
		}

		counter++
		acc, f := ta[account.Hash]
		if !f {
			t.Fatalf("failed to load account with hash: %s", account.Hash)
		}

		err := acc.IsDifferent(&account)

		if err != nil {
			t.Fatalf("accounts are NOT indetical: %s", err.Error())
		}
	}

	err, _ := <-inErrors
	if err != nil {
		t.Fatal(err.Error())
	}

	if counter != testDBSize {
		t.Fatalf("failed to load all the accounts from testing db, db size: %d, loaded accounts: %d", testDBSize, counter)
	}
}

func TestLoadAccountsWithRandomRootHash(t *testing.T) {
	var randomRootHash = make([]byte, 32)
	_, err := rand.Read(randomRootHash)
	if err != nil {
		t.Fatalf("could not generate randomized byte array; %s", err.Error())
	}

	store, _ := Connect("ldb", os.TempDir(), "test")
	defer func(store kvdb.Store) {
		err := store.Close()
		if err != nil {
			t.Fatalf("failed test data build; could not close store; %s", err.Error())
		}
	}(store)

	stateDB, _, _, _ := makeTestTrie(t, store)
	_, inErrors := LoadAccounts(context.Background(), stateDB, common.BytesToHash(randomRootHash), numberOfWorkers)

	err, _ = <-inErrors
	if err == nil || err.Error() != fmt.Sprintf("root hash %s not found", common.BytesToHash(randomRootHash)) {
		t.Fatalf("should NOT be able open trie with invalid root hash: %s", err.Error())
	}
}
