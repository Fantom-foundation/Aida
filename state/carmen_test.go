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

func TestMakeCarmenStateDBMemory(t *testing.T) {
	_, err := MakeCarmenStateDB("", "", false)
	if err != nil {
		t.Fatalf("failed to create carmen state DB: %v", err)
	}
}

func TestMakeCarmenStateDBInvalid(t *testing.T) {
	_, err := MakeCarmenStateDB("", "invalid-variant", false)
	if err == nil {
		t.Fatalf("failed to throw error whale creating carmen state DB")
	}
}

func TestCloseCarmenDB(t *testing.T) {
	csDB, err := MakeCarmenStateDB("", "go-memory", false)
	if err != nil {
		t.Fatalf("failed to create carmen state DB: %v", err)
	}

	err = csDB.Close()
	if err != nil {
		t.Fatalf("failed to close carmen state DB: %v", err)
	}
}

func TestBeginBlockApply(t *testing.T) {
	csDB, err := MakeCarmenStateDB("", "go-memory", false)
	if err != nil {
		t.Fatalf("failed to create carmen state DB: %v", err)
	}

	err = csDB.BeginBlockApply()
	if err != nil {
		t.Fatalf("failed to begin block apply: %v", err)
	}
}

func TestAccountLifecycle(t *testing.T) {
	csDB, err := MakeCarmenStateDB("", "go-memory", false)
	if err != nil {
		t.Fatalf("failed to create carmen state DB: %v", err)
	}

	addr := makeAccountAddress(t)

	csDB.CreateAccount(addr)

	if !csDB.Exist(addr) {
		t.Fatal("failed to create carmen state DB account")
	}

	if !csDB.Empty(addr) {
		t.Fatal("failed to create carmen state DB account; should be empty")
	}

	if !csDB.Suicide(addr) {
		t.Fatal("failed to suicide carmen state DB account;")
	}

	if !csDB.HasSuicided(addr) {
		t.Fatal("failed to suicide carmen state DB account;")
	}
}

func TestAccountBalanceOperations(t *testing.T) {
	csDB, err := MakeCarmenStateDB("", "go-memory", false)
	if err != nil {
		t.Fatalf("failed to create carmen state DB: %v", err)
	}

	addr := makeAccountAddress(t)

	csDB.CreateAccount(addr)

	// seed the PRNG
	rand.Seed(time.Now().UnixNano())

	// get randomized balance
	rangeLower := 1
	rangeUpper := 1000 * 5000
	randInt := rangeUpper + rand.Intn(rangeUpper-rangeLower+1)
	addition := big.NewInt(int64(randInt))

	csDB.AddBalance(addr, addition)

	if csDB.GetBalance(addr).Cmp(addition) != 0 {
		t.Fatal("failed to add balance to carmen state DB account")
	}

	randInt = rand.Intn(rangeUpper - rangeLower + 1)
	subtraction := big.NewInt(int64(randInt))
	expectedResult := big.NewInt(0).Sub(addition, subtraction)

	csDB.SubBalance(addr, subtraction)

	if csDB.GetBalance(addr).Cmp(expectedResult) != 0 {
		t.Fatal("failed to subtract balance to carmen state DB account")
	}
}

func TestNonceOperations(t *testing.T) {
	csDB, err := MakeCarmenStateDB("", "go-memory", false)
	if err != nil {
		t.Fatalf("failed to create carmen state DB: %v", err)
	}

	addr := makeAccountAddress(t)

	csDB.CreateAccount(addr)

	// seed the PRNG
	rand.Seed(time.Now().UnixNano())

	// get randomized nonce
	rangeLower := 1
	rangeUpper := 1000 * 5000
	randomInt := rangeLower + rand.Intn(rangeUpper-rangeLower+1)

	newNonce := uint64(randomInt)

	csDB.SetNonce(addr, newNonce)

	if csDB.GetNonce(addr) != newNonce {
		t.Fatal("failed to update account nonce")
	}
}

func TestCodeOperations(t *testing.T) {
	csDB, err := MakeCarmenStateDB("", "go-memory", false)
	if err != nil {
		t.Fatalf("failed to create carmen state DB: %v", err)
	}

	addr := makeAccountAddress(t)

	csDB.CreateAccount(addr)

	// generate new randomized code
	code := makeAccountCode(t)

	if csDB.GetCodeSize(addr) != 0 {
		t.Fatal("failed to update account code; wrong initial size")
	}

	csDB.SetCode(addr, code)

	if bytes.Compare(csDB.GetCode(addr), code) != 0 {
		t.Fatal("failed to update account code; wrong value")
	}

	if csDB.GetCodeSize(addr) != len(code) {
		t.Fatal("failed to update account code; wrong size")
	}
}

func TestStartBulkLoadAndClose(t *testing.T) {
	csDB, err := MakeCarmenStateDB("", "go-memory", false)
	if err != nil {
		t.Fatalf("failed to create carmen state DB: %v", err)
	}

	cbl := csDB.StartBulkLoad()

	err = cbl.Close()
	if err != nil {
		t.Fatalf("failed to close bulk load: %v", err)
	}
}

func TestSetBalance(t *testing.T) {
	csDB, err := MakeCarmenStateDB("", "go-memory", false)
	if err != nil {
		t.Fatalf("failed to create carmen state DB: %v", err)
	}

	cbl := csDB.StartBulkLoad()

	addr := makeAccountAddress(t)

	cbl.CreateAccount(addr)

	// seed the PRNG
	rand.Seed(time.Now().UnixNano())

	// get randomized balance
	rangeLower := 1
	rangeUpper := 1000 * 5000
	randomInt := rangeLower + rand.Intn(rangeUpper-rangeLower+1)

	newBalance := big.NewInt(int64(randomInt))
	cbl.SetBalance(addr, newBalance)

	if csDB.GetBalance(addr).Cmp(newBalance) != 0 {
		t.Fatal("failed to update account balance")
	}
}

func TestSetNonce(t *testing.T) {
	csDB, err := MakeCarmenStateDB("", "go-memory", false)
	if err != nil {
		t.Fatalf("failed to create carmen state DB: %v", err)
	}

	cbl := csDB.StartBulkLoad()

	addr := makeAccountAddress(t)

	cbl.CreateAccount(addr)

	// seed the PRNG
	rand.Seed(time.Now().UnixNano())

	// get randomized nonce
	rangeLower := 1
	rangeUpper := 1000 * 5000
	randomInt := rangeLower + rand.Intn(rangeUpper-rangeLower+1)

	newNonce := uint64(randomInt)

	cbl.SetNonce(addr, newNonce)

	if csDB.GetNonce(addr) != newNonce {
		t.Fatal("failed to update account nonce")
	}
}

func TestSetState(t *testing.T) {
	csDB, err := MakeCarmenStateDB("", "go-memory", false)
	if err != nil {
		t.Fatalf("failed to create carmen state DB: %v", err)
	}

	cbl := csDB.StartBulkLoad()

	addr := makeAccountAddress(t)

	cbl.CreateAccount(addr)

	// generate state key and value
	key := makeRandomHash(t)
	value := makeRandomHash(t)

	cbl.SetState(addr, key, value)

	if csDB.GetState(addr, key) != value {
		t.Fatal("failed to update account state")
	}
}

func TestSetCode(t *testing.T) {
	csDB, err := MakeCarmenStateDB("", "go-memory", false)
	if err != nil {
		t.Fatalf("failed to create carmen state DB: %v", err)
	}

	cbl := csDB.StartBulkLoad()

	addr := makeAccountAddress(t)

	cbl.CreateAccount(addr)

	// generate new randomized code
	code := makeAccountCode(t)

	cbl.SetCode(addr, code)

	if bytes.Compare(csDB.GetCode(addr), code) != 0 {
		t.Fatal("failed to update account code")
	}
}

func TestAutomaticBlockEnd(t *testing.T) {
	csDB, err := MakeCarmenStateDB("", "go-memory", false)
	if err != nil {
		t.Fatalf("failed to create carmen state DB: %v", err)
	}

	cbl := csDB.StartBulkLoad()

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

			cbl.SetBalance(account, newBalance)
		case operationType == 2:
			// set code
			code := makeAccountCode(t)

			cbl.SetCode(account, code)
		case operationType == 3:
			// set state
			key := makeRandomHash(t)
			value := makeRandomHash(t)

			cbl.SetState(account, key, value)
		case operationType == 4:
			// set nonce
			rangeLower = 1
			rangeUpper = 1000 * 5000
			randomInt := rangeLower + rand.Intn(rangeUpper-rangeLower+1)

			newNonce := uint64(randomInt)

			cbl.SetNonce(account, newNonce)
		default:
			// set code by default
			code := makeAccountCode(t)

			cbl.SetCode(account, code)
		}
	}
}
