package state

import (
	"bytes"
	"github.com/ethereum/go-ethereum/core/types"
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

// TestMakeCarmenStateDBMemory tests db initialization with no variant specified
func TestMakeCarmenStateDBMemory(t *testing.T) {
	_, err := MakeCarmenStateDB("", "", false)
	if err != nil {
		t.Fatalf("failed to create carmen state DB: %v", err)
	}
}

// TestMakeCarmenStateDBInvalid tests db initialization with invalid variant
func TestMakeCarmenStateDBInvalid(t *testing.T) {
	_, err := MakeCarmenStateDB("", "invalid-variant", false)
	if err == nil {
		t.Fatalf("failed to throw error whale creating carmen state DB")
	}
}

// TestCloseCarmenDB test closing db immediately after initialization
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

// TestBeginBlockApply tests block apply start
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

// TestAccountLifecycle tests account operations - create, check if it exists, if it's empty, suicide and suicide confirmation
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

// TestAccountBalanceOperations tests balance operations - add, subtract and check if the value is correct
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

// TestNonceOperations tests account nonce updating
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

// TestCodeOperations tests account code updating
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

// TestStateOperations tests account state update
func TestStateOperations(t *testing.T) {
	csDB, err := MakeCarmenStateDB("", "go-memory", false)
	if err != nil {
		t.Fatalf("failed to create carmen state DB: %v", err)
	}

	addr := makeAccountAddress(t)

	csDB.CreateAccount(addr)

	// generate state key and value
	key := makeRandomHash(t)
	value := makeRandomHash(t)

	csDB.SetState(addr, key, value)

	if csDB.GetState(addr, key) != value {
		t.Fatal("failed to update account state")
	}
}

// TestTrxBlockEpochOperations tests creation of randomized epochs with blocks and transactions
func TestTrxBlockEpochOperations(t *testing.T) {
	csDB, err := MakeCarmenStateDB("", "go-memory", false)
	if err != nil {
		t.Fatalf("failed to create carmen state DB: %v", err)
	}

	blockNumber := 0
	trxNumber := 0
	for i := 0; i < 10; i++ {
		csDB.BeginEpoch(uint64(i))

		for j := 0; j < 100; j++ {
			csDB.BeginBlock(uint64(blockNumber))
			blockNumber++

			for k := 0; k < 100; k++ {
				csDB.BeginTransaction(uint32(trxNumber))
				trxNumber++
				csDB.EndTransaction()
			}

			csDB.EndBlock()
		}

		csDB.EndEpoch()
	}
}

// TestRefundOperations tests adding and subtracting refund value
func TestRefundOperations(t *testing.T) {
	csDB, err := MakeCarmenStateDB("", "go-memory", false)
	if err != nil {
		t.Fatalf("failed to create carmen state DB: %v", err)
	}

	refundValue := uint64(56166560)
	csDB.AddRefund(refundValue)

	if csDB.GetRefund() != refundValue {
		t.Fatal("failed to add refund")
	}

	reducedRefund := refundValue - uint64(30000000)

	csDB.SubRefund(uint64(30000000))

	if csDB.GetRefund() != reducedRefund {
		t.Fatal("failed to subtract refund")
	}
}

// TestAccessListOperations tests operations with creating, updating a checking AccessList
func TestAccessListOperations(t *testing.T) {
	csDB, err := MakeCarmenStateDB("", "go-memory", false)
	if err != nil {
		t.Fatalf("failed to create carmen state DB: %v", err)
	}

	// prepare content of access list
	sender := makeAccountAddress(t)
	dest := makeAccountAddress(t)
	precompiles := []common.Address{
		makeAccountAddress(t),
		makeAccountAddress(t),
		makeAccountAddress(t),
	}
	txAccesses := types.AccessList{
		types.AccessTuple{
			Address: makeAccountAddress(t),
			StorageKeys: []common.Hash{
				makeRandomHash(t),
				makeRandomHash(t),
			},
		},
		types.AccessTuple{
			Address: makeAccountAddress(t),
			StorageKeys: []common.Hash{
				makeRandomHash(t),
				makeRandomHash(t),
				makeRandomHash(t),
				makeRandomHash(t),
			},
		},
	}

	// create access list
	csDB.PrepareAccessList(sender, &dest, precompiles, txAccesses)

	// add some more data after the creation for good measure
	newAddr := makeAccountAddress(t)
	newSlot := makeRandomHash(t)
	csDB.AddAddressToAccessList(newAddr)
	csDB.AddSlotToAccessList(newAddr, newSlot)

	// check content of access list
	if !csDB.AddressInAccessList(sender) {
		t.Fatal("failed to add sender address to access list")
	}

	if !csDB.AddressInAccessList(dest) {
		t.Fatal("failed to add destination address to access list")
	}

	if !csDB.AddressInAccessList(newAddr) {
		t.Fatal("failed to add new address to access list after it was already created")
	}

	for _, addr := range precompiles {
		if !csDB.AddressInAccessList(addr) {
			t.Fatal("failed to add precompile address to access list")
		}
	}

	for _, txAccess := range txAccesses {
		if !csDB.AddressInAccessList(txAccess.Address) {
			t.Fatal("failed to add transaction access address to access list")
		}

		for _, storageKey := range txAccess.StorageKeys {
			addrOK, slotOK := csDB.SlotInAccessList(txAccess.Address, storageKey)
			if !addrOK || !slotOK {
				t.Fatal("failed to add transaction access address to access list")
			}
		}
	}

	addrOK, slotOK := csDB.SlotInAccessList(newAddr, newSlot)
	if !addrOK || !slotOK {
		t.Fatal("failed to add new slot to access list after it was already created")
	}
}

// TestGetArchiveState tests retrieving an archive state
func TestGetArchiveState(t *testing.T) {
	tmpDir := t.TempDir()
	csDB, err := MakeCarmenStateDB(tmpDir, "go-ldb", true)
	if err != nil {
		t.Fatalf("failed to create carmen state DB: %v", err)
	}

	_, err = csDB.GetArchiveState(1)

	if err != nil {
		t.Fatalf("failed to retrieve archive state of carmen state DB: %v", err)
	}
}

// TestSetBalance tests setting an accounts balance
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

	err = cbl.Close()
	if err != nil {
		t.Fatal("failed to close bulk load")
	}

	if csDB.GetBalance(addr).Cmp(newBalance) != 0 {
		t.Fatal("failed to update account balance")
	}
}

// TestSetBalance tests setting an accounts nonce
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

	err = cbl.Close()
	if err != nil {
		t.Fatal("failed to close bulk load")
	}

	if csDB.GetNonce(addr) != newNonce {
		t.Fatal("failed to update account nonce")
	}
}

// TestSetBalance tests setting an accounts state
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

	err = cbl.Close()
	if err != nil {
		t.Fatal("failed to close bulk load")
	}

	if csDB.GetState(addr, key) != value {
		t.Fatal("failed to update account state")
	}
}

// TestSetBalance tests setting an accounts code
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

	err = cbl.Close()
	if err != nil {
		t.Fatal("failed to close bulk load")
	}

	if bytes.Compare(csDB.GetCode(addr), code) != 0 {
		t.Fatal("failed to update account code")
	}
}

// TestBulkloadOperations tests multiple operation in one bulkload
func TestBulkloadOperations(t *testing.T) {
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
