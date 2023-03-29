package state

import (
	"bytes"
	"fmt"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

type erigonStateTestCase struct {
	variant string
}

func getErigonStateTestCases() []erigonStateTestCase {
	testCases := []erigonStateTestCase{
		{"go-memory"},
		{"go-ldb"},
	}

	return testCases
}

// TestErigonState_MakeErigonStateDBMemory tests creation of erigon state DB and closing it immediately
func TestErigonState_MakeErigonStateDBMemory(t *testing.T) {
	for _, tc := range getErigonStateTestCases() {
		t.Run(fmt.Sprintf("DB variant: %s", tc.variant), func(t *testing.T) {
			fsDB, err := MakeErigonStateDB(t.TempDir(), tc.variant, common.Hash{})
			if err != nil {
				t.Fatalf("failed to create erigon state DB: %v", err)
			}

			err = fsDB.Close()
			if err != nil {
				t.Fatalf("failed to close erigon state DB: %v", err)
			}
		})
	}
}

// TestErigonState_MakeErigonStateDBInvalid tests creation of erigon state DB without specifying a variant
func TestErigonState_MakeErigonStateDBInvalid(t *testing.T) {
	_, err := MakeErigonStateDB("", "", common.Hash{})
	if err == nil {
		t.Fatalf("failed to throw error while creating erigon DB")
	}
}

// TestErigonState_BeginBlockApply tests if starting block apply will run successfully
func TestErigonState_BeginBlockApply(t *testing.T) {
	for _, tc := range getErigonStateTestCases() {
		t.Run(fmt.Sprintf("DB variant: %s", tc.variant), func(t *testing.T) {
			fsDB, err := MakeErigonStateDB(t.TempDir(), tc.variant, common.Hash{})
			if err != nil {
				t.Fatalf("failed to create erigon state DB: %v", err)
			}

			err = fsDB.BeginBlockApply()
			if err != nil {
				t.Fatalf("failed to begin block apply: %v", err)
			}
		})
	}
}

// TestErigonState_StartBulkLoadAndClose tests starting and immediately closing bulk load without any operations
func TestErigonState_StartBulkLoadAndClose(t *testing.T) {
	for _, tc := range getErigonStateTestCases() {
		t.Run(fmt.Sprintf("DB variant: %s", tc.variant), func(t *testing.T) {
			fsDB, err := MakeErigonStateDB(t.TempDir(), tc.variant, common.Hash{})
			if err != nil {
				t.Fatalf("failed to create erigon state DB: %v", err)
			}

			fbl := fsDB.StartBulkLoad()

			err = fbl.Close()
			if err != nil {
				t.Fatalf("failed to close bulk load: %v", err)
			}
		})
	}
}

// TestErigonState_SetBalance tests setting random balance to an account in bulk load
func TestErigonState_SetBalance(t *testing.T) {
	for _, tc := range getErigonStateTestCases() {
		t.Run(fmt.Sprintf("DB variant: %s", tc.variant), func(t *testing.T) {
			fsDB, err := MakeErigonStateDB(t.TempDir(), tc.variant, common.Hash{})
			if err != nil {
				t.Fatalf("failed to create erigon state DB: %v", err)
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

// TestErigonState_SetNonce tests setting random nonce to an account in bulk load
func TestErigonState_SetNonce(t *testing.T) {
	for _, tc := range getErigonStateTestCases() {
		t.Run(fmt.Sprintf("DB variant: %s", tc.variant), func(t *testing.T) {
			fsDB, err := MakeErigonStateDB(t.TempDir(), tc.variant, common.Hash{})
			if err != nil {
				t.Fatalf("failed to create erigon state DB: %v", err)
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

// TestErigonState_SetState tests setting randomly generated state to an account in bulk load
func TestErigonState_SetState(t *testing.T) {
	for _, tc := range getErigonStateTestCases() {
		t.Run(fmt.Sprintf("DB variant: %s", tc.variant), func(t *testing.T) {
			fsDB, err := MakeErigonStateDB(t.TempDir(), tc.variant, common.Hash{})
			if err != nil {
				t.Fatalf("failed to create erigon state DB: %v", err)
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

// TestErigonState_SetCode tests setting randomly generated code to an account in bulk load
func TestErigonState_SetCode(t *testing.T) {
	for _, tc := range getErigonStateTestCases() {
		t.Run(fmt.Sprintf("DB variant: %s", tc.variant), func(t *testing.T) {
			fsDB, err := MakeErigonStateDB(t.TempDir(), tc.variant, common.Hash{})
			if err != nil {
				t.Fatalf("failed to create erigon state DB: %v", err)
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

// TestErigonState_AutomaticBlockEnd creates 100 randomized accounts and runs 1 000 000 randomized operations in random order
// to test automatic block ending
func TestErigonState_AutomaticBlockEnd(t *testing.T) {
	for _, tc := range getErigonStateTestCases() {
		t.Run(fmt.Sprintf("DB variant: %s", tc.variant), func(t *testing.T) {
			fsDB, err := MakeErigonStateDB(t.TempDir(), tc.variant, common.Hash{})
			if err != nil {
				t.Fatalf("failed to create erigon state DB: %v", err)
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
