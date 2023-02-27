package state

import (
	"bytes"
	"fmt"
	"math/big"
	"math/rand"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
)

type testCase struct {
	directory string
	variant   string
	rootHash  common.Hash
}

func getTestCases(t *testing.T) []testCase {
	testCases := []testCase{
		{t.TempDir(), "go-memory", common.Hash{}},
		{t.TempDir(), "go-ldb", common.Hash{}},
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

// TestFlatState_MakeFlatStateDBMemory tests creation of flat state DB and closing it immediately
func TestFlatState_MakeFlatStateDBMemory(t *testing.T) {
	for _, tc := range getTestCases(t) {
		t.Run(fmt.Sprintf("DB variant: %s, archive enabled: %v", tc.variant, tc.rootHash), func(t *testing.T) {
			fsDB, err := MakeFlatStateDB(tc.directory, tc.variant, tc.rootHash)
			if err != nil {
				t.Fatalf("failed to create flat state DB: %v", err)
			}

			err = fsDB.Close()
			if err != nil {
				t.Fatalf("failed to close flat state DB: %v", err)
			}
		})
	}
}

// TestFlatState_MakeFlatStateDBInvalid tests creation of flat state DB without specifying a variant
func TestFlatState_MakeFlatStateDBInvalid(t *testing.T) {
	_, err := MakeFlatStateDB("", "", common.Hash{})
	if err == nil {
		t.Fatalf("failed to throw error while creating flat DB")
	}
}

// TestFlatState_BeginBlockApply tests if starting block apply will run successfully
func TestFlatState_BeginBlockApply(t *testing.T) {
	for _, tc := range getTestCases(t) {
		t.Run(fmt.Sprintf("DB variant: %s, archive enabled: %v", tc.variant, tc.rootHash), func(t *testing.T) {
			fsDB, err := MakeFlatStateDB(tc.directory, tc.variant, tc.rootHash)
			if err != nil {
				t.Fatalf("failed to create flat state DB: %v", err)
			}

			err = fsDB.BeginBlockApply()
			if err != nil {
				t.Fatalf("failed to begin block apply: %v", err)
			}
		})
	}
}

// TestFlatState_StartBulkLoadAndClose tests starting and immediately closing bulk load without any operations
func TestFlatState_StartBulkLoadAndClose(t *testing.T) {
	for _, tc := range getTestCases(t) {
		t.Run(fmt.Sprintf("DB variant: %s, archive enabled: %v", tc.variant, tc.rootHash), func(t *testing.T) {
			fsDB, err := MakeFlatStateDB(tc.directory, tc.variant, tc.rootHash)
			if err != nil {
				t.Fatalf("failed to create flat state DB: %v", err)
			}

			fbl := fsDB.StartBulkLoad()

			err = fbl.Close()
			if err != nil {
				t.Fatalf("failed to close bulk load: %v", err)
			}
		})
	}
}

// TestFlatState_SetBalance tests setting random balance to an account in bulk load
func TestFlatState_SetBalance(t *testing.T) {
	for _, tc := range getTestCases(t) {
		t.Run(fmt.Sprintf("DB variant: %s, archive enabled: %v", tc.variant, tc.rootHash), func(t *testing.T) {
			fsDB, err := MakeFlatStateDB(tc.directory, tc.variant, tc.rootHash)
			if err != nil {
				t.Fatalf("failed to create flat state DB: %v", err)
			}

			fbl := fsDB.StartBulkLoad()

			addr := common.BytesToAddress(makeRandomByteSlice(t, 40))

			fbl.CreateAccount(addr)

			randomInt := getRandom(1, 1000*5000)
			newBalance := big.NewInt(int64(randomInt))
			fbl.SetBalance(addr, newBalance)

			if fsDB.GetBalance(addr).Cmp(newBalance) != 0 {
				t.Fatal("failed to update account balance")
			}
		})
	}
}

// TestFlatState_SetNonce tests setting random nonce to an account in bulk load
func TestFlatState_SetNonce(t *testing.T) {
	for _, tc := range getTestCases(t) {
		t.Run(fmt.Sprintf("DB variant: %s, archive enabled: %v", tc.variant, tc.rootHash), func(t *testing.T) {
			fsDB, err := MakeFlatStateDB(tc.directory, tc.variant, tc.rootHash)
			if err != nil {
				t.Fatalf("failed to create flat state DB: %v", err)
			}

			fbl := fsDB.StartBulkLoad()

			addr := common.BytesToAddress(makeRandomByteSlice(t, 40))

			fbl.CreateAccount(addr)

			randomInt := getRandom(1, 1000*5000)
			newNonce := uint64(randomInt)
			fbl.SetNonce(addr, newNonce)

			if fsDB.GetNonce(addr) != newNonce {
				t.Fatal("failed to update account nonce")
			}
		})
	}
}

// TestFlatState_SetState tests setting randomly generated state to an account in bulk load
func TestFlatState_SetState(t *testing.T) {
	for _, tc := range getTestCases(t) {
		t.Run(fmt.Sprintf("DB variant: %s, archive enabled: %v", tc.variant, tc.rootHash), func(t *testing.T) {
			fsDB, err := MakeFlatStateDB(tc.directory, tc.variant, tc.rootHash)
			if err != nil {
				t.Fatalf("failed to create flat state DB: %v", err)
			}

			fbl := fsDB.StartBulkLoad()

			addr := common.BytesToAddress(makeRandomByteSlice(t, 40))

			fbl.CreateAccount(addr)

			// generate state key and value
			key := common.BytesToHash(makeRandomByteSlice(t, 32))
			value := common.BytesToHash(makeRandomByteSlice(t, 32))

			fbl.SetState(addr, key, value)

			if fsDB.GetState(addr, key) != value {
				t.Fatal("failed to update account state")
			}
		})
	}
}

// TestFlatState_SetCode tests setting randomly generated code to an account in bulk load
func TestFlatState_SetCode(t *testing.T) {
	for _, tc := range getTestCases(t) {
		t.Run(fmt.Sprintf("DB variant: %s, archive enabled: %v", tc.variant, tc.rootHash), func(t *testing.T) {
			fsDB, err := MakeFlatStateDB(tc.directory, tc.variant, tc.rootHash)
			if err != nil {
				t.Fatalf("failed to create flat state DB: %v", err)
			}

			fbl := fsDB.StartBulkLoad()

			addr := common.BytesToAddress(makeRandomByteSlice(t, 40))

			fbl.CreateAccount(addr)

			// generate new randomized code
			code := makeRandomByteSlice(t, 2048)

			fbl.SetCode(addr, code)

			if bytes.Compare(fsDB.GetCode(addr), code) != 0 {
				t.Fatal("failed to update account code")
			}
		})
	}
}

// TestFlatState_AutomaticBlockEnd creates 100 randomized accounts and runs 1 000 000 randomized operations in random order
// to test automatic block ending
func TestFlatState_AutomaticBlockEnd(t *testing.T) {
	for _, tc := range getTestCases(t) {
		t.Run(fmt.Sprintf("DB variant: %s, archive enabled: %v", tc.variant, tc.rootHash), func(t *testing.T) {
			fsDB, err := MakeFlatStateDB(tc.directory, tc.variant, tc.rootHash)
			if err != nil {
				t.Fatalf("failed to create flat state DB: %v", err)
			}

			fbl := fsDB.StartBulkLoad()

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
					fbl.SetBalance(account, newBalance)
				case operationType == 2:
					// set code
					code := makeRandomByteSlice(t, 2048)
					fbl.SetCode(account, code)
				case operationType == 3:
					// set state
					key := common.BytesToHash(makeRandomByteSlice(t, 32))
					value := common.BytesToHash(makeRandomByteSlice(t, 32))
					fbl.SetState(account, key, value)
				case operationType == 4:
					// set nonce
					newNonce := uint64(getRandom(0, 1000*5000))
					fbl.SetNonce(account, newNonce)
				default:
					// set code by default
					code := makeRandomByteSlice(t, 2048)
					fbl.SetCode(account, code)
				}
			}
		})
	}
}
