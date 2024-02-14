package proxy

import (
	"bytes"
	"errors"
	"math/big"
	"testing"

	"github.com/Fantom-foundation/Aida/state"
	carmen "github.com/Fantom-foundation/Carmen/go/state"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"go.uber.org/mock/gomock"
)

func makeTestShadowDB(t *testing.T, ctc state.CarmenStateTestCase) state.StateDB {
	csDB, err := state.MakeCarmenStateDB(t.TempDir(), ctc.Variant, ctc.Archive, ctc.Schema)
	if errors.Is(err, carmen.UnsupportedConfiguration) {
		t.Skip("unsupported configuration")
	}

	if err != nil {
		t.Fatalf("failed to create carmen state DB: %v", err)
	}

	gsDB, err := state.MakeGethStateDB(t.TempDir(), "", common.Hash{}, false, nil)

	if err != nil {
		t.Fatalf("failed to create geth state DB: %v", err)
	}

	shadowDB := NewShadowProxy(csDB, gsDB)

	return shadowDB
}

// TestShadowState_InitCloseShadowDB test closing db immediately after initialization
func TestShadowState_InitCloseShadowDB(t *testing.T) {
	for _, ctc := range state.GetCarmenStateTestCases() {
		t.Run(ctc.String(), func(t *testing.T) {
			shadowDB := makeTestShadowDB(t, ctc)

			err := shadowDB.Close()
			if err != nil {
				t.Fatalf("failed to close shadow state DB: %v", err)
			}
		})
	}
}

// TestShadowState_AccountLifecycle tests account operations - create, check if it exists, if it's empty, suicide and suicide confirmation
func TestShadowState_AccountLifecycle(t *testing.T) {
	for _, ctc := range state.GetCarmenStateTestCases() {
		t.Run(ctc.String(), func(t *testing.T) {
			shadowDB := makeTestShadowDB(t, ctc)

			// Close DB after test ends
			defer func(shadowDB state.StateDB) {
				err := shadowDB.Close()
				if err != nil {
					t.Fatalf("failed to close shadow state DB: %v", err)
				}
			}(shadowDB)

			addr := common.BytesToAddress(state.MakeRandomByteSlice(t, 40))

			shadowDB.CreateAccount(addr)

			if !shadowDB.Exist(addr) {
				t.Fatal("failed to create carmen state DB account")
			}

			if !shadowDB.Empty(addr) {
				t.Fatal("failed to create carmen state DB account; should be empty")
			}

			if !shadowDB.Suicide(addr) {
				t.Fatal("failed to suicide carmen state DB account;")
			}

			if !shadowDB.HasSuicided(addr) {
				t.Fatal("failed to suicide carmen state DB account;")
			}
		})
	}
}

// TestShadowState_AccountBalanceOperations tests balance operations - add, subtract and check if the value is correct
func TestShadowState_AccountBalanceOperations(t *testing.T) {
	for _, ctc := range state.GetCarmenStateTestCases() {
		t.Run(ctc.String(), func(t *testing.T) {
			shadowDB := makeTestShadowDB(t, ctc)

			// Close DB after test ends
			defer func(shadowDB state.StateDB) {
				err := shadowDB.Close()
				if err != nil {
					t.Fatalf("failed to close shadow state DB: %v", err)
				}
			}(shadowDB)

			addr := common.BytesToAddress(state.MakeRandomByteSlice(t, 40))

			shadowDB.CreateAccount(addr)

			// get randomized balance
			additionBase := state.GetRandom(1, 1000*5000)
			addition := big.NewInt(int64(additionBase))

			shadowDB.AddBalance(addr, addition)

			if shadowDB.GetBalance(addr).Cmp(addition) != 0 {
				t.Fatal("failed to add balance to carmen state DB account")
			}

			subtraction := big.NewInt(int64(state.GetRandom(1, additionBase)))
			expectedResult := big.NewInt(0).Sub(addition, subtraction)

			shadowDB.SubBalance(addr, subtraction)

			if shadowDB.GetBalance(addr).Cmp(expectedResult) != 0 {
				t.Fatal("failed to subtract balance to carmen state DB account")
			}
		})
	}
}

// TestShadowState_NonceOperations tests account nonce updating
func TestShadowState_NonceOperations(t *testing.T) {
	for _, ctc := range state.GetCarmenStateTestCases() {
		t.Run(ctc.String(), func(t *testing.T) {
			shadowDB := makeTestShadowDB(t, ctc)

			// Close DB after test ends
			defer func(shadowDB state.StateDB) {
				err := shadowDB.Close()
				if err != nil {
					t.Fatalf("failed to close shadow state DB: %v", err)
				}
			}(shadowDB)

			addr := common.BytesToAddress(state.MakeRandomByteSlice(t, 40))

			shadowDB.CreateAccount(addr)

			// get randomized nonce
			newNonce := uint64(state.GetRandom(1, 1000*5000))

			shadowDB.SetNonce(addr, newNonce)

			if shadowDB.GetNonce(addr) != newNonce {
				t.Fatal("failed to update account nonce")
			}
		})
	}
}

// TestShadowState_CodeOperations tests account code updating
func TestShadowState_CodeOperations(t *testing.T) {
	for _, ctc := range state.GetCarmenStateTestCases() {
		t.Run(ctc.String(), func(t *testing.T) {
			shadowDB := makeTestShadowDB(t, ctc)

			// Close DB after test ends
			defer func(shadowDB state.StateDB) {
				err := shadowDB.Close()
				if err != nil {
					t.Fatalf("failed to close shadow state DB: %v", err)
				}
			}(shadowDB)

			addr := common.BytesToAddress(state.MakeRandomByteSlice(t, 40))

			shadowDB.CreateAccount(addr)

			// generate new randomized code
			code := state.MakeRandomByteSlice(t, 2048)

			if shadowDB.GetCodeSize(addr) != 0 {
				t.Fatal("failed to update account code; wrong initial size")
			}

			shadowDB.SetCode(addr, code)

			if bytes.Compare(shadowDB.GetCode(addr), code) != 0 {
				t.Fatal("failed to update account code; wrong value")
			}

			if shadowDB.GetCodeSize(addr) != len(code) {
				t.Fatal("failed to update account code; wrong size")
			}
		})
	}
}

// TestShadowState_StateOperations tests account state update
func TestShadowState_StateOperations(t *testing.T) {
	for _, ctc := range state.GetCarmenStateTestCases() {
		t.Run(ctc.String(), func(t *testing.T) {
			shadowDB := makeTestShadowDB(t, ctc)

			// Close DB after test ends
			defer func(shadowDB state.StateDB) {
				err := shadowDB.Close()
				if err != nil {
					t.Fatalf("failed to close shadow state DB: %v", err)
				}
			}(shadowDB)

			addr := common.BytesToAddress(state.MakeRandomByteSlice(t, 40))

			shadowDB.CreateAccount(addr)

			// generate state key and value
			key := common.BytesToHash(state.MakeRandomByteSlice(t, 32))
			value := common.BytesToHash(state.MakeRandomByteSlice(t, 32))

			shadowDB.SetState(addr, key, value)

			if shadowDB.GetState(addr, key) != value {
				t.Fatal("failed to update account state")
			}
		})
	}
}

// TestShadowState_TrxBlockSyncPeriodOperations tests creation of randomized sync-periods with blocks and transactions
func TestShadowState_TrxBlockSyncPeriodOperations(t *testing.T) {
	for _, ctc := range state.GetCarmenStateTestCases() {
		t.Run(ctc.String(), func(t *testing.T) {
			shadowDB := makeTestShadowDB(t, ctc)

			// Close DB after test ends
			defer func(shadowDB state.StateDB) {
				err := shadowDB.Close()
				if err != nil {
					t.Fatalf("failed to close shadow state DB: %v", err)
				}
			}(shadowDB)

			blockNumber := 1
			trxNumber := 1
			for i := 0; i < 10; i++ {
				shadowDB.BeginSyncPeriod(uint64(i))

				for j := 0; j < 100; j++ {
					shadowDB.BeginBlock(uint64(blockNumber))
					blockNumber++

					for k := 0; k < 100; k++ {
						shadowDB.BeginTransaction(uint32(trxNumber))
						trxNumber++
						shadowDB.EndTransaction()
					}

					shadowDB.EndBlock()
				}

				shadowDB.EndSyncPeriod()
			}
		})
	}
}

// TestShadowState_RefundOperations tests adding and subtracting refund value
func TestShadowState_RefundOperations(t *testing.T) {
	for _, ctc := range state.GetCarmenStateTestCases() {
		t.Run(ctc.String(), func(t *testing.T) {
			shadowDB := makeTestShadowDB(t, ctc)

			// Close DB after test ends
			defer func(shadowDB state.StateDB) {
				err := shadowDB.Close()
				if err != nil {
					t.Fatalf("failed to close shadow state DB: %v", err)
				}
			}(shadowDB)

			refundValue := uint64(state.GetRandom(10000*4000, 10000*5000))
			shadowDB.AddRefund(refundValue)

			if shadowDB.GetRefund() != refundValue {
				t.Fatal("failed to add refund")
			}

			reducedRefund := refundValue - uint64(30000000)

			shadowDB.SubRefund(uint64(30000000))

			if shadowDB.GetRefund() != reducedRefund {
				t.Fatal("failed to subtract refund")
			}
		})
	}
}

// TestShadowState_AccessListOperations tests operations with creating, updating a checking AccessList
func TestShadowState_AccessListOperations(t *testing.T) {
	for _, ctc := range state.GetCarmenStateTestCases() {
		t.Run(ctc.String(), func(t *testing.T) {
			shadowDB := makeTestShadowDB(t, ctc)

			// Close DB after test ends
			defer func(shadowDB state.StateDB) {
				err := shadowDB.Close()
				if err != nil {
					t.Fatalf("failed to close shadow state DB: %v", err)
				}
			}(shadowDB)

			// prepare content of access list
			sender := common.BytesToAddress(state.MakeRandomByteSlice(t, 40))
			dest := common.BytesToAddress(state.MakeRandomByteSlice(t, 40))
			precompiles := []common.Address{
				common.BytesToAddress(state.MakeRandomByteSlice(t, 40)),
				common.BytesToAddress(state.MakeRandomByteSlice(t, 40)),
				common.BytesToAddress(state.MakeRandomByteSlice(t, 40)),
			}
			txAccesses := types.AccessList{
				types.AccessTuple{
					Address: common.BytesToAddress(state.MakeRandomByteSlice(t, 40)),
					StorageKeys: []common.Hash{
						common.BytesToHash(state.MakeRandomByteSlice(t, 32)),
						common.BytesToHash(state.MakeRandomByteSlice(t, 32)),
					},
				},
				types.AccessTuple{
					Address: common.BytesToAddress(state.MakeRandomByteSlice(t, 40)),
					StorageKeys: []common.Hash{
						common.BytesToHash(state.MakeRandomByteSlice(t, 32)),
						common.BytesToHash(state.MakeRandomByteSlice(t, 32)),
						common.BytesToHash(state.MakeRandomByteSlice(t, 32)),
						common.BytesToHash(state.MakeRandomByteSlice(t, 32)),
					},
				},
			}

			// create access list
			shadowDB.PrepareAccessList(sender, &dest, precompiles, txAccesses)

			// add some more data after the creation for good measure
			newAddr := common.BytesToAddress(state.MakeRandomByteSlice(t, 40))
			newSlot := common.BytesToHash(state.MakeRandomByteSlice(t, 32))
			shadowDB.AddAddressToAccessList(newAddr)
			shadowDB.AddSlotToAccessList(newAddr, newSlot)

			// check content of access list
			if !shadowDB.AddressInAccessList(sender) {
				t.Fatal("failed to add sender address to access list")
			}

			if !shadowDB.AddressInAccessList(dest) {
				t.Fatal("failed to add destination address to access list")
			}

			if !shadowDB.AddressInAccessList(newAddr) {
				t.Fatal("failed to add new address to access list after it was already created")
			}

			for _, addr := range precompiles {
				if !shadowDB.AddressInAccessList(addr) {
					t.Fatal("failed to add precompile address to access list")
				}
			}

			for _, txAccess := range txAccesses {
				if !shadowDB.AddressInAccessList(txAccess.Address) {
					t.Fatal("failed to add transaction access address to access list")
				}

				for _, storageKey := range txAccess.StorageKeys {
					addrOK, slotOK := shadowDB.SlotInAccessList(txAccess.Address, storageKey)
					if !addrOK || !slotOK {
						t.Fatal("failed to add transaction access address to access list")
					}
				}
			}

			addrOK, slotOK := shadowDB.SlotInAccessList(newAddr, newSlot)
			if !addrOK || !slotOK {
				t.Fatal("failed to add new slot to access list after it was already created")
			}
		})
	}
}

// TestShadowState_SetBalanceUsingBulkInsertion tests setting an accounts balance
func TestShadowState_SetBalanceUsingBulkInsertion(t *testing.T) {
	for _, ctc := range state.GetCarmenStateTestCases() {
		t.Run(ctc.String(), func(t *testing.T) {
			shadowDB := makeTestShadowDB(t, ctc)

			// Close DB after test ends
			defer func(shadowDB state.StateDB) {
				err := shadowDB.Close()
				if err != nil {
					t.Fatalf("failed to close shadow state DB: %v", err)
				}
			}(shadowDB)

			cbl := shadowDB.StartBulkLoad(0)

			addr := common.BytesToAddress(state.MakeRandomByteSlice(t, 40))

			cbl.CreateAccount(addr)

			newBalance := big.NewInt(int64(state.GetRandom(1, 1000*5000)))
			cbl.SetBalance(addr, newBalance)

			err := cbl.Close()
			if err != nil {
				t.Fatalf("failed to close bulk load: %v", err)
			}

			if shadowDB.GetBalance(addr).Cmp(newBalance) != 0 {
				t.Fatal("failed to update account balance")
			}
		})
	}
}

// TestShadowState_SetNonceUsingBulkInsertion tests setting an accounts nonce
func TestShadowState_SetNonceUsingBulkInsertion(t *testing.T) {
	for _, ctc := range state.GetCarmenStateTestCases() {
		t.Run(ctc.String(), func(t *testing.T) {
			shadowDB := makeTestShadowDB(t, ctc)

			// Close DB after test ends
			defer func(shadowDB state.StateDB) {
				err := shadowDB.Close()
				if err != nil {
					t.Fatalf("failed to close shadow state DB: %v", err)
				}
			}(shadowDB)

			cbl := shadowDB.StartBulkLoad(0)

			addr := common.BytesToAddress(state.MakeRandomByteSlice(t, 40))

			cbl.CreateAccount(addr)

			newNonce := uint64(state.GetRandom(1, 1000*5000))

			cbl.SetNonce(addr, newNonce)

			err := cbl.Close()
			if err != nil {
				t.Fatalf("failed to close bulk load: %v", err)
			}

			if shadowDB.GetNonce(addr) != newNonce {
				t.Fatal("failed to update account nonce")
			}
		})
	}
}

// TestShadowState_SetStateUsingBulkInsertion tests setting an accounts state
func TestShadowState_SetStateUsingBulkInsertion(t *testing.T) {
	for _, ctc := range state.GetCarmenStateTestCases() {
		t.Run(ctc.String(), func(t *testing.T) {
			shadowDB := makeTestShadowDB(t, ctc)

			// Close DB after test ends
			defer func(shadowDB state.StateDB) {
				err := shadowDB.Close()
				if err != nil {
					t.Fatalf("failed to close shadow state DB: %v", err)
				}
			}(shadowDB)

			cbl := shadowDB.StartBulkLoad(0)

			addr := common.BytesToAddress(state.MakeRandomByteSlice(t, 40))

			cbl.CreateAccount(addr)

			// generate state key and value
			key := common.BytesToHash(state.MakeRandomByteSlice(t, 32))
			value := common.BytesToHash(state.MakeRandomByteSlice(t, 32))

			cbl.SetState(addr, key, value)

			err := cbl.Close()
			if err != nil {
				t.Fatalf("failed to close bulk load: %v", err)
			}

			if shadowDB.GetState(addr, key) != value {
				t.Fatal("failed to update account state")
			}
		})
	}
}

// TestShadowState_SetCodeUsingBulkInsertion tests setting an accounts code
func TestShadowState_SetCodeUsingBulkInsertion(t *testing.T) {
	for _, ctc := range state.GetCarmenStateTestCases() {
		t.Run(ctc.String(), func(t *testing.T) {
			shadowDB := makeTestShadowDB(t, ctc)

			// Close DB after test ends
			defer func(shadowDB state.StateDB) {
				err := shadowDB.Close()
				if err != nil {
					t.Fatalf("failed to close shadow state DB: %v", err)
				}
			}(shadowDB)

			cbl := shadowDB.StartBulkLoad(0)

			addr := common.BytesToAddress(state.MakeRandomByteSlice(t, 40))

			cbl.CreateAccount(addr)

			// generate new randomized code
			code := state.MakeRandomByteSlice(t, 2048)

			cbl.SetCode(addr, code)

			err := cbl.Close()
			if err != nil {
				t.Fatalf("failed to close bulk load: %v", err)
			}

			if bytes.Compare(shadowDB.GetCode(addr), code) != 0 {
				t.Fatal("failed to update account code")
			}
		})
	}
}

// TestShadowState_BulkloadOperations tests multiple operation in one bulkload
func TestShadowState_BulkloadOperations(t *testing.T) {
	for _, ctc := range state.GetCarmenStateTestCases() {
		t.Run(ctc.String(), func(t *testing.T) {
			shadowDB := makeTestShadowDB(t, ctc)

			// generate 100 randomized accounts
			accounts := [100]common.Address{}

			for i := 0; i < len(accounts); i++ {
				accounts[i] = common.BytesToAddress(state.MakeRandomByteSlice(t, 40))
				shadowDB.CreateAccount(accounts[i])
			}

			cbl := shadowDB.StartBulkLoad(0)

			for _, account := range accounts {
				// randomized operation
				operationType := state.GetRandom(0, 4)

				switch {
				case operationType == 1:
					// set balance
					newBalance := big.NewInt(int64(state.GetRandom(0, 1000*5000)))

					cbl.SetBalance(account, newBalance)
				case operationType == 2:
					// set code
					code := state.MakeRandomByteSlice(t, 2048)

					cbl.SetCode(account, code)
				case operationType == 3:
					// set state
					key := common.BytesToHash(state.MakeRandomByteSlice(t, 32))
					value := common.BytesToHash(state.MakeRandomByteSlice(t, 32))

					cbl.SetState(account, key, value)
				case operationType == 4:
					// set nonce
					newNonce := uint64(state.GetRandom(0, 1000*5000))

					cbl.SetNonce(account, newNonce)
				default:
					// set code by default
					code := state.MakeRandomByteSlice(t, 2048)

					cbl.SetCode(account, code)
				}
			}

			err := cbl.Close()
			if err != nil {
				t.Fatalf("failed to close bulk load: %v", err)
			}

			// This is placed at the end instead of in a defer clause to
			// avoid being called in case of a panic occurring during the
			// test. This would make error diagnostic very difficult.
			if err := shadowDB.Close(); err != nil {
				t.Fatalf("failed to close shadow state DB: %v", err)
			}
		})
	}
}

func TestShadowState_GetShadowDB(t *testing.T) {
	for _, ctc := range state.GetCarmenStateTestCases() {
		t.Run(ctc.String(), func(t *testing.T) {
			csDB, err := state.MakeCarmenStateDB(t.TempDir(), ctc.Variant, ctc.Archive, 1)
			if errors.Is(err, carmen.UnsupportedConfiguration) {
				t.Skip("unsupported configuration")
			}

			if err != nil {
				t.Fatalf("failed to create carmen state DB: %v", err)
			}

			gsDB, err := state.MakeGethStateDB(t.TempDir(), "", common.Hash{}, false, nil)

			if err != nil {
				t.Fatalf("failed to create geth state DB: %v", err)
			}

			shadowDB := NewShadowProxy(csDB, gsDB)

			// Close DB after test ends
			defer func(shadowDB state.StateDB) {
				err := shadowDB.Close()
				if err != nil {
					t.Fatalf("failed to close shadow state DB: %v", err)
				}
			}(shadowDB)

			if shadowDB.GetShadowDB() != gsDB {
				t.Fatal("Wrong return value of GetShadowDB")
			}

		})
	}
}

func TestShadowState_GetLogsSuccessful(t *testing.T) {
	ctrl := gomock.NewController(t)
	pdb := state.NewMockStateDB(ctrl)
	sdb := state.NewMockStateDB(ctrl)
	db := NewShadowProxy(pdb, sdb)
	txHash := common.HexToHash("0x1")
	blockHash := common.HexToHash("0x2")
	log1 := &types.Log{}

	pdb.EXPECT().GetLogs(txHash, blockHash).Return([]*types.Log{log1})
	sdb.EXPECT().GetLogs(txHash, blockHash).Return([]*types.Log{log1})

	db.GetLogs(txHash, blockHash)
	if err := db.Error(); err != nil {
		t.Fatalf("Failed to compare logs; %v", err)
	}
}

func TestShadowState_GetLogsExpectError_LengthDifferent(t *testing.T) {
	ctrl := gomock.NewController(t)
	pdb := state.NewMockStateDB(ctrl)
	sdb := state.NewMockStateDB(ctrl)
	db := NewShadowProxy(pdb, sdb)
	txHash := common.HexToHash("0x1")
	blockHash := common.HexToHash("0x2")
	log1 := &types.Log{}

	pdb.EXPECT().GetLogs(txHash, blockHash).Return(nil)
	sdb.EXPECT().GetLogs(txHash, blockHash).Return([]*types.Log{log1})

	db.GetLogs(txHash, blockHash)
	if err := db.Error(); err == nil {
		t.Fatal("Expect error in GetLogs lengths")
	}
}

func TestShadowState_GetLogsExpectError_BloomDifferent(t *testing.T) {
	ctrl := gomock.NewController(t)
	pdb := state.NewMockStateDB(ctrl)
	sdb := state.NewMockStateDB(ctrl)
	db := NewShadowProxy(pdb, sdb)
	txHash := common.HexToHash("0x1")
	blockHash := common.HexToHash("0x2")
	log1 := &types.Log{}
	log2 := &types.Log{Address: common.HexToAddress("0x3")}

	pdb.EXPECT().GetLogs(txHash, blockHash).Return([]*types.Log{log1})
	sdb.EXPECT().GetLogs(txHash, blockHash).Return([]*types.Log{log2})

	db.GetLogs(txHash, blockHash)
	if err := db.Error(); err == nil {
		t.Fatal("Expect error in GetLogs lengths")
	}
}
