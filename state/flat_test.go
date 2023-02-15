package state

import (
	"bytes"
	"math/big"
	"math/rand"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

// makeAccountAddress creates randomly generated address
func makeAccountAddress(t *testing.T) common.Address {
	// generate public key
	pk, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("failed test data; could not create random key; %s", err.Error())
	}

	// generate account address
	return crypto.PubkeyToAddress(pk.PublicKey)
}

// makeRandomHash creates randomly generated 32-byte hash
func makeRandomHash(t *testing.T) common.Hash {
	hashing := crypto.NewKeccakState()

	// make byte slice
	buffer := make([]byte, 32)

	// fill the slice with random data
	_, err := rand.Read(buffer)
	if err != nil {
		t.Fatalf("failed test data; can not generate random state hash; %s", err.Error())
	}

	// hash buffer data
	return crypto.HashData(hashing, buffer)
}

// makeAccountCode creates randomly generated code byte slice
func makeAccountCode(t *testing.T) []byte {
	hashing := crypto.NewKeccakState()

	// make code container
	code := make([]byte, rand.Intn(2048))

	// fill the code with random data
	_, err := rand.Read(code)
	if err != nil {
		t.Fatalf("failed test data; can not generate randomized code; %s", err.Error())
	}

	// create code hash
	ch := crypto.HashData(hashing, code)

	// return code hash represented by bytes slice
	return ch.Bytes()
}

// TestMakeFlatStateDBMemory tests creation of in-memory flat state DB
func TestMakeFlatStateDBMemory(t *testing.T) {
	_, err := MakeFlatStateDB("", "go-memory", common.Hash{})
	if err != nil {
		t.Fatalf("failed to create flat state DB: %v", err)
	}
}

// TestMakeFlatStateDBLevelDB tests creation of flat state DB based on levelDB
func TestMakeFlatStateDBLevelDB(t *testing.T) {
	_, err := MakeFlatStateDB("testDB", "go-ldb", common.Hash{})
	if err != nil {
		t.Fatalf("failed to create flat state DB: %v", err)
	}
}

// TestMakeFlatStateDBDefault tests creation of flat state DB without specifying a variant
// should create in-memory DB
func TestMakeFlatStateDBDefault(t *testing.T) {
	_, err := MakeFlatStateDB("", "", common.Hash{})
	if err == nil {
		t.Fatalf("failed to throw error whale creating flat DB")
	}
}

// TestCloseFlatDB tests closing flat state DB
func TestCloseFlatDB(t *testing.T) {
	fsDB, err := MakeFlatStateDB("", "go-memory", common.Hash{})
	if err != nil {
		t.Fatalf("failed to create flat state DB: %v", err)
	}

	err = fsDB.Close()
	if err != nil {
		t.Fatalf("failed to close flat state DB: %v", err)
	}
}

// TestBeginBlockApply tests if starting block apply will run successfully
func TestBeginBlockApply(t *testing.T) {
	fsDB, err := MakeFlatStateDB("", "go-memory", common.Hash{})
	if err != nil {
		t.Fatalf("failed to create flat state DB: %v", err)
	}

	err = fsDB.BeginBlockApply()
	if err != nil {
		t.Fatalf("failed to begin block apply: %v", err)
	}
}

// TestStartBulkLoadAndClose tests starting and immediately closing bulk load without any operations
func TestStartBulkLoadAndClose(t *testing.T) {
	fsDB, err := MakeFlatStateDB("", "go-memory", common.Hash{})
	if err != nil {
		t.Fatalf("failed to create flat state DB: %v", err)
	}

	fbl := fsDB.StartBulkLoad()

	err = fbl.Close()
	if err != nil {
		t.Fatalf("failed to close bulk load: %v", err)
	}
}

// TestSetBalance tests setting random balance to an account in bulk load
func TestSetBalance(t *testing.T) {
	fsDB, err := MakeFlatStateDB("", "go-memory", common.Hash{})
	if err != nil {
		t.Fatalf("failed to create flat state DB: %v", err)
	}

	fbl := fsDB.StartBulkLoad()

	addr := makeAccountAddress(t)

	fbl.CreateAccount(addr)

	// seed the PRNG
	rand.Seed(time.Now().UnixNano())

	// get randomized balance
	rangeLower := 1
	rangeUpper := 1000 * 5000
	randomInt := rangeLower + rand.Intn(rangeUpper-rangeLower+1)

	newBalance := big.NewInt(int64(randomInt))
	fbl.SetBalance(addr, newBalance)

	if fsDB.GetBalance(addr).Cmp(newBalance) != 0 {
		t.Fatal("failed to update account balance")
	}
}

// TestSetNonce tests setting random nonce to an account in bulk load
func TestSetNonce(t *testing.T) {
	fsDB, err := MakeFlatStateDB("", "go-memory", common.Hash{})
	if err != nil {
		t.Fatalf("failed to create flat state DB: %v", err)
	}

	fbl := fsDB.StartBulkLoad()

	addr := makeAccountAddress(t)

	fbl.CreateAccount(addr)

	// seed the PRNG
	rand.Seed(time.Now().UnixNano())

	// get randomized nonce
	rangeLower := 1
	rangeUpper := 1000 * 5000
	randomInt := rangeLower + rand.Intn(rangeUpper-rangeLower+1)

	newNonce := uint64(randomInt)

	fbl.SetNonce(addr, newNonce)

	if fsDB.GetNonce(addr) != newNonce {
		t.Fatal("failed to update account nonce")
	}
}

// TestSetState tests setting randomly generated state to an account in bulk load
func TestSetState(t *testing.T) {
	fsDB, err := MakeFlatStateDB("", "go-memory", common.Hash{})
	if err != nil {
		t.Fatalf("failed to create flat state DB: %v", err)
	}

	fbl := fsDB.StartBulkLoad()

	addr := makeAccountAddress(t)

	fbl.CreateAccount(addr)

	// generate state key and value
	key := makeRandomHash(t)
	value := makeRandomHash(t)

	fbl.SetState(addr, key, value)

	if fsDB.GetState(addr, key) != value {
		t.Fatal("failed to update account state")
	}
}

// TestSetCode tests setting randomly generated code to an account in bulk load
func TestSetCode(t *testing.T) {
	fsDB, err := MakeFlatStateDB("", "go-memory", common.Hash{})
	if err != nil {
		t.Fatalf("failed to create flat state DB: %v", err)
	}

	fbl := fsDB.StartBulkLoad()

	addr := makeAccountAddress(t)

	fbl.CreateAccount(addr)

	// generate new randomized code
	code := makeAccountCode(t)

	fbl.SetCode(addr, code)

	if bytes.Compare(fsDB.GetCode(addr), code) != 0 {
		t.Fatal("failed to update account code")
	}
}

// TestAutomaticBlockEnd creates 100 randomized accounts and runs 1 000 000 randomized operations in random order
// to test automatic block ending
func TestAutomaticBlockEnd(t *testing.T) {
	fsDB, err := MakeFlatStateDB("", "go-memory", common.Hash{})
	if err != nil {
		t.Fatalf("failed to create flat state DB: %v", err)
	}

	fbl := fsDB.StartBulkLoad()

	// generate 100 randomized accounts
	accounts := [100]common.Address{}

	for i := 0; i < len(accounts); i++ {
		accounts[i] = makeAccountAddress(t)
	}

	for i := 0; i < (1000 * 1000); i++ {
		// seed the PRNG
		rand.Seed(time.Now().UnixNano())

		// get random account index
		rangeLower := 1
		rangeUpper := 99
		accIndex := rangeLower + rand.Intn(rangeUpper-rangeLower+1)
		account := accounts[accIndex]

		// randomized operation
		rangeLower = 1
		rangeUpper = 4
		operationType := rangeLower + rand.Intn(rangeUpper-rangeLower+1)

		switch {
		case operationType == 1:
			// set balance
			rangeLower = 1
			rangeUpper = 1000 * 5000
			randomInt := rangeLower + rand.Intn(rangeUpper-rangeLower+1)

			newBalance := big.NewInt(int64(randomInt))

			fbl.SetBalance(account, newBalance)
		case operationType == 2:
			// set code
			code := makeAccountCode(t)

			fbl.SetCode(account, code)
		case operationType == 3:
			// set state
			key := makeRandomHash(t)
			value := makeRandomHash(t)

			fbl.SetState(account, key, value)
		case operationType == 4:
			// set nonce
			rangeLower = 1
			rangeUpper = 1000 * 5000
			randomInt := rangeLower + rand.Intn(rangeUpper-rangeLower+1)

			newNonce := uint64(randomInt)

			fbl.SetNonce(account, newNonce)
		default:
			// set code by default
			code := makeAccountCode(t)

			fbl.SetCode(account, code)
		}
	}
}
