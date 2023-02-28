package state

import (
	"bytes"
	"fmt"
	"math/big"
	"math/rand"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

type testCase struct {
	variant string
	archive string
}

func getTestCases() []testCase {
	variants := []string{
		"",
		"go-memory",
		"go-file-nocache",
		"go-file",
		"go-ldb-nocache",
		"go-ldb",
		"cpp-memory",
		"cpp-file",
		"cpp-ldb",
	}

	archives := []string{
		"none",
		"leveldb",
		"sqlite",
	}

	var testCases []testCase

	for _, variant := range variants {
		for _, archive := range archives {
			testCases = append(testCases, testCase{variant: variant, archive: archive})
		}
	}

	return testCases
}

// makeRandomByteSlice creates byte slice of given length with randomized values
func makeRandomByteSlice(t *testing.T, bufferLength int) []byte {
	// make byte slice
	buffer := make([]byte, bufferLength)

	// fill the slice with random data
	_, err := rand.Read(buffer)
	if err != nil {
		t.Fatalf("failed test data; can not generate random byte slice; %s", err.Error())
	}

	return buffer
}

func getRandom(rangeLower int, rangeUpper int) int {
	// seed the PRNG
	rand.Seed(time.Now().UnixNano())

	// get randomized balance
	randInt := rangeLower + rand.Intn(rangeUpper-rangeLower+1)
	return randInt
}

// TestCarmenState_MakeCarmenStateDBInvalid tests db initialization with invalid variant
func TestCarmenState_MakeCarmenStateDBInvalid(t *testing.T) {
	_, err := MakeCarmenStateDB("", "invalid-variant", "")
	if err == nil {
		t.Fatalf("failed to throw error while creating carmen state DB")
	}
}

// TestCarmenState_CloseCarmenDB test closing db immediately after initialization
func TestCarmenState_InitCloseCarmenDB(t *testing.T) {
	for _, tc := range getTestCases() {
		t.Run(fmt.Sprintf("DB variant: %s, archive enabled: %v", tc.variant, tc.archive), func(t *testing.T) {
			csDB, err := MakeCarmenStateDB(t.TempDir(), tc.variant, tc.archive)

			if err != nil {
				t.Fatalf("failed to create carmen state DB: %v", err)
			}

			err = csDB.Close()
			if err != nil {
				t.Fatalf("failed to close carmen state DB: %v", err)
			}
		})
	}
}

// TestCarmenState_BeginBlockApply tests block apply start
func TestCarmenState_BeginBlockApply(t *testing.T) {
	for _, tc := range getTestCases() {
		t.Run(fmt.Sprintf("DB variant: %s, archive enabled: %v", tc.variant, tc.archive), func(t *testing.T) {
			csDB, err := MakeCarmenStateDB(t.TempDir(), tc.variant, tc.archive)

			if err != nil {
				t.Fatalf("failed to create carmen state DB: %v", err)
			}

			err = csDB.BeginBlockApply()
			if err != nil {
				t.Fatalf("failed to begin block apply: %v", err)
			}
		})
	}
}

// TestCarmenState_AccountLifecycle tests account operations - create, check if it exists, if it's empty, suicide and suicide confirmation
func TestCarmenState_AccountLifecycle(t *testing.T) {
	for _, tc := range getTestCases() {
		t.Run(fmt.Sprintf("DB variant: %s, archive enabled: %v", tc.variant, tc.archive), func(t *testing.T) {
			csDB, err := MakeCarmenStateDB(t.TempDir(), tc.variant, tc.archive)

			if err != nil {
				t.Fatalf("failed to create carmen state DB: %v", err)
			}

			addr := common.BytesToAddress(makeRandomByteSlice(t, 40))

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
		})
	}
}

// TestCarmenState_AccountBalanceOperations tests balance operations - add, subtract and check if the value is correct
func TestCarmenState_AccountBalanceOperations(t *testing.T) {
	for _, tc := range getTestCases() {
		t.Run(fmt.Sprintf("DB variant: %s, archive enabled: %v", tc.variant, tc.archive), func(t *testing.T) {
			csDB, err := MakeCarmenStateDB(t.TempDir(), tc.variant, tc.archive)

			if err != nil {
				t.Fatalf("failed to create carmen state DB: %v", err)
			}

			addr := common.BytesToAddress(makeRandomByteSlice(t, 40))

			csDB.CreateAccount(addr)

			// get randomized balance
			addition := big.NewInt(int64(getRandom(1, 1000*5000)))

			csDB.AddBalance(addr, addition)

			if csDB.GetBalance(addr).Cmp(addition) != 0 {
				t.Fatal("failed to add balance to carmen state DB account")
			}

			subtraction := big.NewInt(int64(getRandom(1, 1000*5000)))
			expectedResult := big.NewInt(0).Sub(addition, subtraction)

			csDB.SubBalance(addr, subtraction)

			if csDB.GetBalance(addr).Cmp(expectedResult) != 0 {
				t.Fatal("failed to subtract balance to carmen state DB account")
			}
		})
	}
}

// TestCarmenState_NonceOperations tests account nonce updating
func TestCarmenState_NonceOperations(t *testing.T) {
	for _, tc := range getTestCases() {
		t.Run(fmt.Sprintf("DB variant: %s, archive enabled: %v", tc.variant, tc.archive), func(t *testing.T) {
			csDB, err := MakeCarmenStateDB(t.TempDir(), tc.variant, tc.archive)

			if err != nil {
				t.Fatalf("failed to create carmen state DB: %v", err)
			}

			addr := common.BytesToAddress(makeRandomByteSlice(t, 40))

			csDB.CreateAccount(addr)

			// get randomized nonce
			newNonce := uint64(getRandom(1, 1000*5000))

			csDB.SetNonce(addr, newNonce)

			if csDB.GetNonce(addr) != newNonce {
				t.Fatal("failed to update account nonce")
			}
		})
	}
}

// TestCarmenState_CodeOperations tests account code updating
func TestCarmenState_CodeOperations(t *testing.T) {
	for _, tc := range getTestCases() {
		t.Run(fmt.Sprintf("DB variant: %s, archive enabled: %v", tc.variant, tc.archive), func(t *testing.T) {
			csDB, err := MakeCarmenStateDB(t.TempDir(), tc.variant, tc.archive)

			if err != nil {
				t.Fatalf("failed to create carmen state DB: %v", err)
			}

			addr := common.BytesToAddress(makeRandomByteSlice(t, 40))

			csDB.CreateAccount(addr)

			// generate new randomized code
			code := makeRandomByteSlice(t, 2048)

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
		})
	}
}

// TestCarmenState_StateOperations tests account state update
func TestCarmenState_StateOperations(t *testing.T) {
	for _, tc := range getTestCases() {
		t.Run(fmt.Sprintf("DB variant: %s, archive enabled: %v", tc.variant, tc.archive), func(t *testing.T) {
			csDB, err := MakeCarmenStateDB(t.TempDir(), tc.variant, tc.archive)

			if err != nil {
				t.Fatalf("failed to create carmen state DB: %v", err)
			}

			addr := common.BytesToAddress(makeRandomByteSlice(t, 40))

			csDB.CreateAccount(addr)

			// generate state key and value
			key := common.BytesToHash(makeRandomByteSlice(t, 32))
			value := common.BytesToHash(makeRandomByteSlice(t, 32))

			csDB.SetState(addr, key, value)

			if csDB.GetState(addr, key) != value {
				t.Fatal("failed to update account state")
			}
		})
	}
}

// TestCarmenState_TrxBlockEpochOperations tests creation of randomized epochs with blocks and transactions
func TestCarmenState_TrxBlockEpochOperations(t *testing.T) {
	for _, tc := range getTestCases() {
		t.Run(fmt.Sprintf("DB variant: %s, archive enabled: %v", tc.variant, tc.archive), func(t *testing.T) {
			csDB, err := MakeCarmenStateDB(t.TempDir(), tc.variant, tc.archive)

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
		})
	}
}

// TestCarmenState_RefundOperations tests adding and subtracting refund value
func TestCarmenState_RefundOperations(t *testing.T) {
	for _, tc := range getTestCases() {
		t.Run(fmt.Sprintf("DB variant: %s, archive enabled: %v", tc.variant, tc.archive), func(t *testing.T) {
			csDB, err := MakeCarmenStateDB(t.TempDir(), tc.variant, tc.archive)

			if err != nil {
				t.Fatalf("failed to create carmen state DB: %v", err)
			}

			refundValue := uint64(getRandom(10000*4000, 10000*5000))
			csDB.AddRefund(refundValue)

			if csDB.GetRefund() != refundValue {
				t.Fatal("failed to add refund")
			}

			reducedRefund := refundValue - uint64(30000000)

			csDB.SubRefund(uint64(30000000))

			if csDB.GetRefund() != reducedRefund {
				t.Fatal("failed to subtract refund")
			}
		})
	}
}

// TestCarmenState_AccessListOperations tests operations with creating, updating a checking AccessList
func TestCarmenState_AccessListOperations(t *testing.T) {
	for _, tc := range getTestCases() {
		t.Run(fmt.Sprintf("DB variant: %s, archive enabled: %v", tc.variant, tc.archive), func(t *testing.T) {
			csDB, err := MakeCarmenStateDB(t.TempDir(), tc.variant, tc.archive)

			if err != nil {
				t.Fatalf("failed to create carmen state DB: %v", err)
			}

			// prepare content of access list
			sender := common.BytesToAddress(makeRandomByteSlice(t, 40))
			dest := common.BytesToAddress(makeRandomByteSlice(t, 40))
			precompiles := []common.Address{
				common.BytesToAddress(makeRandomByteSlice(t, 40)),
				common.BytesToAddress(makeRandomByteSlice(t, 40)),
				common.BytesToAddress(makeRandomByteSlice(t, 40)),
			}
			txAccesses := types.AccessList{
				types.AccessTuple{
					Address: common.BytesToAddress(makeRandomByteSlice(t, 40)),
					StorageKeys: []common.Hash{
						common.BytesToHash(makeRandomByteSlice(t, 32)),
						common.BytesToHash(makeRandomByteSlice(t, 32)),
					},
				},
				types.AccessTuple{
					Address: common.BytesToAddress(makeRandomByteSlice(t, 40)),
					StorageKeys: []common.Hash{
						common.BytesToHash(makeRandomByteSlice(t, 32)),
						common.BytesToHash(makeRandomByteSlice(t, 32)),
						common.BytesToHash(makeRandomByteSlice(t, 32)),
						common.BytesToHash(makeRandomByteSlice(t, 32)),
					},
				},
			}

			// create access list
			csDB.PrepareAccessList(sender, &dest, precompiles, txAccesses)

			// add some more data after the creation for good measure
			newAddr := common.BytesToAddress(makeRandomByteSlice(t, 40))
			newSlot := common.BytesToHash(makeRandomByteSlice(t, 32))
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
		})
	}
}

// TestCarmenState_GetArchiveState tests retrieving an archive state
func TestCarmenState_GetArchiveState(t *testing.T) {
	for _, tc := range getTestCases() {
		t.Run(fmt.Sprintf("DB variant: %s, archive enabled: %v", tc.variant, tc.archive), func(t *testing.T) {
			csDB, err := MakeCarmenStateDB(t.TempDir(), tc.variant, tc.archive)

			if err != nil {
				t.Fatalf("failed to create carmen state DB: %v", err)
			}

			_, err = csDB.GetArchiveState(1)

			if (tc.archive == "sqlite" || tc.archive == "leveldb") && err != nil {
				t.Fatalf("failed to retrieve archive state of carmen state DB: %v", err)
			}
		})
	}
}

// TestCarmenState_SetBalanceUsingBulkInsertion tests setting an accounts balance
func TestCarmenState_SetBalanceUsingBulkInsertion(t *testing.T) {
	for _, tc := range getTestCases() {
		t.Run(fmt.Sprintf("DB variant: %s, archive enabled: %v", tc.variant, tc.archive), func(t *testing.T) {
			csDB, err := MakeCarmenStateDB(t.TempDir(), tc.variant, tc.archive)

			if err != nil {
				t.Fatalf("failed to create carmen state DB: %v", err)
			}

			cbl := csDB.StartBulkLoad()

			addr := common.BytesToAddress(makeRandomByteSlice(t, 40))

			cbl.CreateAccount(addr)

			newBalance := big.NewInt(int64(getRandom(1, 1000*5000)))
			cbl.SetBalance(addr, newBalance)

			err = cbl.Close()
			if err != nil {
				t.Fatal("failed to close bulk load")
			}

			if csDB.GetBalance(addr).Cmp(newBalance) != 0 {
				t.Fatal("failed to update account balance")
			}
		})
	}
}

// TestCarmenState_SetNonceUsingBulkInsertion tests setting an accounts nonce
func TestCarmenState_SetNonceUsingBulkInsertion(t *testing.T) {
	for _, tc := range getTestCases() {
		t.Run(fmt.Sprintf("DB variant: %s, archive enabled: %v", tc.variant, tc.archive), func(t *testing.T) {
			csDB, err := MakeCarmenStateDB(t.TempDir(), tc.variant, tc.archive)

			if err != nil {
				t.Fatalf("failed to create carmen state DB: %v", err)
			}

			cbl := csDB.StartBulkLoad()

			addr := common.BytesToAddress(makeRandomByteSlice(t, 40))

			cbl.CreateAccount(addr)

			newNonce := uint64(getRandom(1, 1000*5000))

			cbl.SetNonce(addr, newNonce)

			err = cbl.Close()
			if err != nil {
				t.Fatal("failed to close bulk load")
			}

			if csDB.GetNonce(addr) != newNonce {
				t.Fatal("failed to update account nonce")
			}
		})
	}
}

// TestCarmenState_SetStateUsingBulkInsertion tests setting an accounts state
func TestCarmenState_SetStateUsingBulkInsertion(t *testing.T) {
	for _, tc := range getTestCases() {
		t.Run(fmt.Sprintf("DB variant: %s, archive enabled: %v", tc.variant, tc.archive), func(t *testing.T) {
			csDB, err := MakeCarmenStateDB(t.TempDir(), tc.variant, tc.archive)

			if err != nil {
				t.Fatalf("failed to create carmen state DB: %v", err)
			}

			cbl := csDB.StartBulkLoad()

			addr := common.BytesToAddress(makeRandomByteSlice(t, 40))

			cbl.CreateAccount(addr)

			// generate state key and value
			key := common.BytesToHash(makeRandomByteSlice(t, 32))
			value := common.BytesToHash(makeRandomByteSlice(t, 32))

			cbl.SetState(addr, key, value)

			err = cbl.Close()
			if err != nil {
				t.Fatal("failed to close bulk load")
			}

			if csDB.GetState(addr, key) != value {
				t.Fatal("failed to update account state")
			}
		})
	}
}

// TestCarmenState_SetCodeUsingBulkInsertion tests setting an accounts code
func TestCarmenState_SetCodeUsingBulkInsertion(t *testing.T) {
	for _, tc := range getTestCases() {
		t.Run(fmt.Sprintf("DB variant: %s, archive enabled: %v", tc.variant, tc.archive), func(t *testing.T) {
			csDB, err := MakeCarmenStateDB(t.TempDir(), tc.variant, tc.archive)

			if err != nil {
				t.Fatalf("failed to create carmen state DB: %v", err)
			}

			cbl := csDB.StartBulkLoad()

			addr := common.BytesToAddress(makeRandomByteSlice(t, 40))

			cbl.CreateAccount(addr)

			// generate new randomized code
			code := makeRandomByteSlice(t, 2048)

			cbl.SetCode(addr, code)

			err = cbl.Close()
			if err != nil {
				t.Fatal("failed to close bulk load")
			}

			if bytes.Compare(csDB.GetCode(addr), code) != 0 {
				t.Fatal("failed to update account code")
			}
		})
	}
}

// TestCarmenState_BulkloadOperations tests multiple operation in one bulkload
func TestCarmenState_BulkloadOperations(t *testing.T) {
	for _, tc := range getTestCases() {
		t.Run(fmt.Sprintf("DB variant: %s, archive enabled: %v", tc.variant, tc.archive), func(t *testing.T) {
			csDB, err := MakeCarmenStateDB(t.TempDir(), tc.variant, tc.archive)
			if err != nil {
				t.Fatalf("failed to create carmen state DB: %v", err)
			}

			cbl := csDB.StartBulkLoad()

			// generate 100 randomized accounts
			accounts := [100]common.Address{}

			for i := 0; i < len(accounts); i++ {
				accounts[i] = common.BytesToAddress(makeRandomByteSlice(t, 40))
			}

			for i := 0; i < (1000 * 1000); i++ {
				// get random account index
				accIndex := getRandom(0, 99)
				account := accounts[accIndex]

				// randomized operation
				operationType := getRandom(0, 4)

				switch {
				case operationType == 1:
					// set balance
					newBalance := big.NewInt(int64(getRandom(0, 1000*5000)))

					cbl.SetBalance(account, newBalance)
				case operationType == 2:
					// set code
					code := makeRandomByteSlice(t, 2048)

					cbl.SetCode(account, code)
				case operationType == 3:
					// set state
					key := common.BytesToHash(makeRandomByteSlice(t, 32))
					value := common.BytesToHash(makeRandomByteSlice(t, 32))

					cbl.SetState(account, key, value)
				case operationType == 4:
					// set nonce
					newNonce := uint64(getRandom(0, 1000*5000))

					cbl.SetNonce(account, newNonce)
				default:
					// set code by default
					code := makeRandomByteSlice(t, 2048)

					cbl.SetCode(account, code)
				}
			}
		})
	}
}
