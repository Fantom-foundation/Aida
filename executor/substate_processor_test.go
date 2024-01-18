package executor

import (
	"fmt"
	"math/big"
	"strings"
	"testing"

	substatecontext "github.com/Fantom-foundation/Aida/txcontext/substate"
	substate "github.com/Fantom-foundation/Substate"
	"github.com/Fantom-foundation/go-opera/evmcore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

func newDummyResult(t *testing.T) *substate.SubstateResult {
	r := &substate.SubstateResult{
		Logs:            []*types.Log{},
		ContractAddress: common.HexToAddress("0x0000000000085a12481aEdb59eb3200332aCA541"),
		GasUsed:         1000000,
		Status:          types.ReceiptStatusSuccessful,
	}
	return r
}

// TestPrepareBlockCtx tests a creation of block context from substate environment.
func TestPrepareBlockCtx(t *testing.T) {
	gaslimit := uint64(10000000)
	blocknum := uint64(4600000)
	basefee := big.NewInt(12345)
	env := substatecontext.NewBlockEnvironment(&substate.SubstateEnv{Difficulty: big.NewInt(1), GasLimit: gaslimit, Number: blocknum, Timestamp: 1675961395, BaseFee: basefee})

	// BlockHashes are nil, expect an error
	blockCtx := prepareBlockCtx(env)

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

// TestCompileVMResult tests a construction of substate.Result from data output
func TestCompileVMResult(t *testing.T) {
	var logs []*types.Log
	reciept_fail := &evmcore.ExecutionResult{UsedGas: 100, Err: fmt.Errorf("Test Error")}
	contract := common.HexToAddress("0x0000000000085a12481aEdb59eb3200332aCA541")

	sr := compileVMResult(logs, reciept_fail.UsedGas, reciept_fail.Failed(), contract)

	if sr.GetContractAddress() != contract {
		t.Fatalf("Wrong contract address")
	}
	if sr.GetGasUsed() != reciept_fail.UsedGas {
		t.Fatalf("Wrong amount of gas used")
	}
	if sr.GetStatus() != types.ReceiptStatusFailed {
		t.Fatalf("Wrong txcontext status")
	}

	reciept_success := &evmcore.ExecutionResult{UsedGas: 100, Err: nil}
	sr = compileVMResult(logs, reciept_success.UsedGas, reciept_success.Failed(), contract)

	if sr.GetStatus() != types.ReceiptStatusSuccessful {
		t.Fatalf("Wrong txcontext status")
	}
}

// TestValidateVMResult tests validatation of data result.
func TestValidateVMResult(t *testing.T) {
	expectedResult := newDummyResult(t)
	vmResult := newDummyResult(t)

	// test positive
	err := validateVMResult(substatecontext.NewReceipt(vmResult), substatecontext.NewReceipt(expectedResult))
	if err != nil {
		t.Fatalf("Failed to validate VM output. %v", err)
	}

	// test negative
	// mismatched contract
	vmResult.ContractAddress = common.HexToAddress("0x0000000000085a12481aEdb59eb3200332aCA542")
	err = validateVMResult(substatecontext.NewReceipt(vmResult), substatecontext.NewReceipt(expectedResult))
	if err == nil {
		t.Fatalf("Failed to validate VM output. Expect contract address mismatch error.")
	}
	// mismatched gas used
	vmResult = newDummyResult(t)
	vmResult.GasUsed = 0
	err = validateVMResult(substatecontext.NewReceipt(vmResult), substatecontext.NewReceipt(expectedResult))
	if err == nil {
		t.Fatalf("Failed to validate VM output. Expect gas used mismatch error.")
	}

	// mismatched gas used
	vmResult = newDummyResult(t)
	vmResult.Status = types.ReceiptStatusFailed
	err = validateVMResult(substatecontext.NewReceipt(vmResult), substatecontext.NewReceipt(expectedResult))
	if err == nil {
		t.Fatalf("Failed to validate VM output. Expect staatus mismatch error.")
	}
}

func TestValidateVMResult_ErrorIsInCorrectFormat(t *testing.T) {
	expectedResult := newDummyResult(t)
	vmResult := newDummyResult(t)

	// change result so validation fails
	expectedResult.GasUsed = 15000

	vmRes := substatecontext.NewReceipt(vmResult)
	expRes := substatecontext.NewReceipt(expectedResult)

	err := validateVMResult(vmRes, expRes)
	if err == nil {
		t.Fatal("validation must fail")
	}

	want := fmt.Sprintf("inconsistent output\n"+
		"\ngot:\n"+
		"\tstatus: %v\n"+
		"\tbloom: %v\n"+
		"\tlogs: %v\n"+
		"\tcontract address: %v\n"+
		"\tgas used: %v\n"+
		"\nwant:\n"+
		"\tstatus: %v\n"+
		"\tbloom: %v\n"+
		"\tlogs: %v\n"+
		"\tcontract address: %v\n"+
		"\tgas used: %v\n",
		vmRes.GetStatus(),
		vmRes.GetBloom().Big().Uint64(),
		vmRes.GetLogs(),
		vmRes.GetContractAddress(),
		vmRes.GetGasUsed(),
		expRes.GetStatus(),
		expRes.GetBloom().Big().Uint64(),
		expRes.GetLogs(),
		expRes.GetContractAddress(),
		expRes.GetGasUsed(),
	)
	got := err.Error()

	if strings.Compare(got, want) != 0 {
		t.Fatalf("unexpected err\ngot: %v\n want: %v\n", got, want)
	}
}
