package opera

import (
	"github.com/Fantom-foundation/Aida/world-state/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
	"math/big"
	"math/rand"
	"testing"
)

// testDBSize defines how many test accounts will be injected into the test trie
const testDBSize = 500

// makeTestTrie creates a testing MPT trie filled with random accounts
// State DB, map of testing accounts, and map of account hashes -> account addresses are returned.
func makeTestTrie(t *testing.T) (state.Database, common.Hash, map[common.Hash]types.Account, map[common.Hash]common.Address) {
	// open the source trie DB
	store, _ := Connect("ldb", "", "test")

	// try to create empty MPT
	stateDB := OpenStateDB(store)
	stateTrie, err := stateDB.OpenTrie(common.Hash{})
	found := stateTrie != nil && err == nil
	if !found {
		t.Fatalf("failed test data build; could not open stateTrie; %s", err.Error())
	}

	ta, adh := makeTestAccounts(t, stateDB)

	// iterate over slice of accounts
	for hash, account := range ta {
		// encode account hash and all the account data
		accHash := hash.Bytes()
		acc, err := rlp.EncodeToBytes(account.Account)
		if err != nil {
			t.Fatalf("failed test data build; could not encode account; %s", err.Error())
		}

		// try to update trie with encoded data
		err = stateTrie.TryUpdate(accHash, acc)
		if err != nil {
			t.Fatalf("failed test data build; could not update trie; %s", err.Error())
		}
	}

	// try to commit all the changes
	var sth common.Hash
	sth, err = stateTrie.Commit(nil)
	if err != nil {
		t.Fatalf("failed test data build; could not commit trie; %s", err.Error())
	}

	// returns structure representing new MPT and state trie DB
	return stateDB, sth, ta, adh
}

// makeTestAccounts method creates randomized testing accounts
// returns generated accounts with randomized data and one account which is not a contract
// and map of account hashes -> account addresses
func makeTestAccounts(t *testing.T, stateDB state.Database) (map[common.Hash]types.Account, map[common.Hash]common.Address) {
	var ta = make(map[common.Hash]types.Account, testDBSize)
	var hash2addr = make(map[common.Hash]common.Address, testDBSize)
	var hashing = crypto.NewKeccakState()

	// create contract accounts
	for i := 0; i < testDBSize; i++ {
		pk, err := crypto.GenerateKey()
		if err != nil {
			t.Fatalf("failed test data build; could not create random keys; %s", err.Error())
		}

		addr := crypto.PubkeyToAddress(pk.PublicKey)
		hash := crypto.HashData(hashing, addr.Bytes())
		hash2addr[hash] = addr

		acc := types.Account{
			Hash:    hash,
			Storage: map[common.Hash]common.Hash{},
		}
		acc.Nonce = rand.Uint64()
		acc.Balance = big.NewInt(rand.Int63())
		acc.Root = ethTypes.EmptyRootHash
		acc.CodeHash = types.EmptyCodeHash.Bytes()

		// quarter of the accounts are going to represent as contracts
		if i%4 == 0 {
			// make code container
			acc.Code = make([]byte, rand.Intn(2048))

			// fill the code with random data
			_, err = rand.Read(acc.Code)
			if err != nil {
				t.Fatalf("failed test data build; can not generate random code; %s", err.Error())
			}
			acc.CodeHash = crypto.HashData(hashing, acc.Code).Bytes()

			// fill the storage map
			srh, err := fillAccountStorage(t, &acc, stateDB)
			if err != nil {
				t.Fatalf("failed test data build; could not generate randomized storage; %s", err.Error())
			}
			acc.Root = srh
		}

		// add this account to the map
		ta[hash] = acc
	}

	return ta, hash2addr
}

// fillAccountStorage method creates account storage Trie and fills it with data
// returns root hash of newly created account storage trie
func fillAccountStorage(t *testing.T, account *types.Account, stateDB state.Database) (common.Hash, error) {
	hashing := crypto.NewKeccakState()

	// try to open account storage trie
	st, err := stateDB.OpenStorageTrie(account.Hash, common.Hash{})
	if err != nil {
		return common.Hash{}, err
	}

	var rh common.Hash
	buffer := make([]byte, 32)

	// generate randomized data
	for j := 0; j < 10; j++ {
		_, err = rand.Read(buffer)
		if err != nil {
			t.Fatalf("failed test data build; can not generate random storage key; %s", err.Error())
		}
		k := crypto.HashData(hashing, buffer)

		_, err = rand.Read(buffer)
		if err != nil {
			t.Fatalf("failed test data build; can not generate random storage value; %s", err.Error())
		}

		// encode buffer
		var encoded []byte
		encoded, err = rlp.EncodeToBytes(buffer)
		if err != nil {
			t.Fatalf("failed test data build; can not encode buffer value; %s", err.Error())
		}

		// try to update storage
		account.Storage[k] = crypto.HashData(hashing, buffer)
		err = st.TryUpdate(k.Bytes(), encoded)
		if err != nil {
			t.Fatalf("failed test data build; can not update storage trie; %s", err.Error())
		}

		// try to commit all the changes
		rh, err = st.Commit(nil)
		if err != nil {
			return common.Hash{}, err
		}
	}

	// returns root hash of committed records
	return rh, nil
}
