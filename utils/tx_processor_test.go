package utils

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/Fantom-foundation/Aida/state"
	substate "github.com/Fantom-foundation/Substate"
	"github.com/Fantom-foundation/go-opera/evmcore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

func newDummyResult(t *testing.T) *substate.SubstateResult {
	return &substate.SubstateResult{
		Logs:            []*types.Log{},
		ContractAddress: common.HexToAddress("0x0000000000085a12481aEdb59eb3200332aCA541"),
		GasUsed:         1000000,
		Status:          types.ReceiptStatusSuccessful,
	}
}

func newDummyAlloc(t *testing.T) substate.SubstateAlloc {
	dummyAddress1 := common.HexToAddress("0x0000000000085a12481aEdb59eb3200332aCA541")
	dummyAddress2 := common.HexToAddress("0x0000000000085a12481aEdb59eb3200332aCA542")
	dummyAddress3 := common.HexToAddress("0x0000000000085a12481aEdb59eb3200332aCA543")

	dummyKey1 := common.HexToHash("0x0fa0c3892eaaf05eeca5cf62d715e3a70780103ea10f080e42ebd1c7a2631e1b")
	dummyKey2 := common.HexToHash("0xea79a15cb6361d6a78eee4020c57bb2f58099dcb63a8fb5c2d15b82de2afc2b5")

	dummyValue1 := common.HexToHash("0x01")
	dummyValue2 := common.HexToHash("0x02")

	sa := make(substate.SubstateAlloc)
	// prime substate alloc
	sa[dummyAddress1] = substate.NewSubstateAccount(1, big.NewInt(1000000), []byte{})
	sa[dummyAddress1].Storage[dummyKey1] = dummyValue1
	sa[dummyAddress2] = substate.NewSubstateAccount(2, big.NewInt(2000000), []byte{})
	sa[dummyAddress2].Storage[dummyKey2] = dummyValue2
	sa[dummyAddress3] = substate.NewSubstateAccount(3, big.NewInt(3000000), []byte{})
	return sa
}

// TestPrepareBlockCtx tests a creation of block context from substate environment.
func TestPrepareBlockCtx(t *testing.T) {
	gaslimit := uint64(10000000)
	blocknum := uint64(4600000)
	basefee := big.NewInt(12345)
	env := &substate.SubstateEnv{
		Difficulty: big.NewInt(1),
		GasLimit:   gaslimit,
		Number:     blocknum,
		Timestamp:  1675961395,
		BaseFee:    basefee,
	}

	// BlockHashes are nil, expect an error
	blockCtx := prepareBlockCtx(env, nil)

	if blocknum != blockCtx.BlockNumber.Uint64() {
		t.Fatalf("Wrong block number")
	}
	if gaslimit != blockCtx.GasLimit {
		t.Fatalf("Wrong amount of gas limit")
	}
	if basefee.Cmp(blockCtx.BaseFee) != 0 {
		t.Fatalf("Wrong base fee")
	}
}

// TestCompileVMResult tests a construction of substate.Result from tx output
func TestCompileVMResult(t *testing.T) {
	var logs []*types.Log
	reciept_fail := &evmcore.ExecutionResult{UsedGas: 100, Err: fmt.Errorf("Test Error")}
	contract := common.HexToAddress("0x0000000000085a12481aEdb59eb3200332aCA541")

	sr := compileVMResult(logs, reciept_fail, contract)

	if sr.ContractAddress != contract {
		t.Fatalf("Wrong contract address")
	}
	if sr.GasUsed != reciept_fail.UsedGas {
		t.Fatalf("Wrong amount of gas used")
	}
	if sr.Status != types.ReceiptStatusFailed {
		t.Fatalf("Wrong transaction status")
	}

	reciept_success := &evmcore.ExecutionResult{UsedGas: 100, Err: nil}
	sr = compileVMResult(logs, reciept_success, contract)

	if sr.Status != types.ReceiptStatusSuccessful {
		t.Fatalf("Wrong transaction status")
	}
}

// TestValidateVMResult tests validatation of tx result.
func TestValidateVMResult(t *testing.T) {
	expectedResult := newDummyResult(t)
	vmResult := newDummyResult(t)

	// test positive
	err := validateVMResult(vmResult, expectedResult)
	if err != nil {
		t.Fatalf("Failed to validate VM output. %v", err)
	}

	// test negative
	// mismatched contract
	vmResult.ContractAddress = common.HexToAddress("0x0000000000085a12481aEdb59eb3200332aCA542")
	err = validateVMResult(vmResult, expectedResult)
	if err == nil {
		t.Fatalf("Failed to validate VM output. Expect contract address mismatch error.")
	}
	// mismatched gas used
	vmResult = newDummyResult(t)
	vmResult.GasUsed = 0
	err = validateVMResult(vmResult, expectedResult)
	if err == nil {
		t.Fatalf("Failed to validate VM output. Expect gas used mismatch error.")
	}

	// mismatched gas used
	vmResult = newDummyResult(t)
	vmResult.Status = types.ReceiptStatusFailed
	err = validateVMResult(vmResult, expectedResult)
	if err == nil {
		t.Fatalf("Failed to validate VM output. Expect staatus mismatch error.")
	}
}

// TestValidateVMResult tests validatation of statedb after tx processing.
func TestValidateVMAlloc_Positive(t *testing.T) {
	expectedResult := newDummyAlloc(t)
	vmResult := newDummyAlloc(t)
	db := state.MakeInMemoryStateDB(&vmResult, uint64(1234567))

	cfg := &Config{}
	cfg.StateValidationMode = SubsetCheck
	cfg.UpdateOnFailure = true

	// test positive
	if err := validateVMAlloc(db, expectedResult, cfg); err != nil {
		t.Fatalf("Failed to validate VM output. %v", err)
	}

	cfg.StateValidationMode = EqualityCheck
	if err := validateVMAlloc(db, expectedResult, cfg); err != nil {
		t.Fatalf("Failed to validate VM output. %v", err)
	}

	// DB has one more contract than expected result
	newAddress := common.HexToAddress("0x0000000000085a12481aEdb59eb3200332aCA000")
	vmResult[newAddress] = substate.NewSubstateAccount(1, big.NewInt(1000000), []byte{})
	db = state.MakeInMemoryStateDB(&vmResult, uint64(1234567))

	// check whether expectedResult is contained.
	cfg.StateValidationMode = SubsetCheck
	if err := validateVMAlloc(db, expectedResult, cfg); err != nil {
		t.Fatalf("Failed to validate VM output. %v", err)
	}
	// check for equality. Since db has an extra contract, an error is expected.
	cfg.StateValidationMode = EqualityCheck
	if err := validateVMAlloc(db, expectedResult, cfg); err == nil {
		t.Fatalf("Failed to detect an error.")
	}
}

func TestValidateVMAlloc_Negative(t *testing.T) {
	expectedResult := newDummyAlloc(t)
	vmResult := newDummyAlloc(t)
	db := state.MakeInMemoryStateDB(&vmResult, uint64(1234567))

	cfg := &Config{}
	cfg.UpdateOnFailure = true

	// DB has one more contract than expected result
	newAddress := common.HexToAddress("0x0000000000085a12481aEdb59eb3200332aCA000")
	vmResult[newAddress] = substate.NewSubstateAccount(1, big.NewInt(1000000), []byte{})
	db = state.MakeInMemoryStateDB(&vmResult, uint64(1234567))

	// test negative
	cfg.StateValidationMode = SubsetCheck
	vmResult = make(substate.SubstateAlloc)
	if err := validateVMAlloc(db, expectedResult, cfg); err == nil {
		t.Fatalf("Failed to detect an error.")
	}

	cfg.StateValidationMode = EqualityCheck
	if err := validateVMAlloc(db, expectedResult, cfg); err == nil {
		t.Fatalf("Failed to detect an error.")
	}
}

func TestValidateVMAlloc_DbHasOneMoreContractThanExpected(t *testing.T) {
	expectedResult := newDummyAlloc(t)
	vmResult := newDummyAlloc(t)
	db := state.MakeInMemoryStateDB(&vmResult, uint64(1234567))

	cfg := &Config{}
	cfg.UpdateOnFailure = true

	newAddress := common.HexToAddress("0x0000000000085a12481aEdb59eb3200332aCA000")
	vmResult[newAddress] = substate.NewSubstateAccount(1, big.NewInt(1000000), []byte{})
	db = state.MakeInMemoryStateDB(&vmResult, uint64(1234567))

	// check whether expectedResult is contained.
	cfg.StateValidationMode = SubsetCheck
	if err := validateVMAlloc(db, expectedResult, cfg); err != nil {
		t.Fatalf("Failed to validate VM output. %v", err)
	}
	// check for equality. Since db has an extra contract, an error is expected.
	cfg.StateValidationMode = EqualityCheck
	if err := validateVMAlloc(db, expectedResult, cfg); err == nil {
		t.Fatalf("Failed to detect an error.")
	}
}
