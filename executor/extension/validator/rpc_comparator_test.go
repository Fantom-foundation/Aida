// Copyright 2024 Fantom Foundation
// This file is part of Aida Testing Infrastructure for Sonic
//
// Aida is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Aida is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with Aida. If not, see <http://www.gnu.org/licenses/>.

package validator

import (
	"encoding/json"
	"errors"
	"math/big"
	"strings"
	"testing"

	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/executor/extension"
	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/rpc"
	"github.com/Fantom-foundation/Aida/utils"
	"github.com/Fantom-foundation/lachesis-base/common/littleendian"
	"github.com/status-im/keycard-go/hexutils"
)

const (
	hexOne  = "0x1"
	hexZero = "0x0"

	// 32 bytes returned by EVM as result one
	longHexOne = "0x0000000000000000000000000000000000000000000000000000000000000001"

	// 32 bytes returned by EVM as a zero result
	longHexZero = "0x0000000000000000000000000000000000000000000000000000000000000000"
)

func TestRPCComparator_RPCComparatorIsNotCreatedIfNotEnabled(t *testing.T) {
	cfg := &utils.Config{}
	cfg.Validate = false

	c := MakeRpcComparator(cfg)
	if _, ok := c.(extension.NilExtension[*rpc.RequestAndResults]); !ok {
		t.Error("extension must be nil")
	}
}

func TestRPCComparator_PostTransactionDoesNotFailIfContinueOnFailureIsTrue(t *testing.T) {
	cfg := &utils.Config{}
	cfg.Validate = true
	cfg.ContinueOnFailure = true

	bigRes, _ := new(big.Int).SetString("1", 16)
	rec, _ := json.Marshal(hexZero)

	data := &rpc.RequestAndResults{
		Query: &rpc.Body{
			MethodBase: "getBalance",
			Method:     "eth_getBalance",
		},
		Response: &rpc.Response{
			Result: rec,
		},
	}

	s := executor.State[*rpc.RequestAndResults]{
		Data: data,
	}

	c := makeRPCComparator(cfg, logger.NewLogger("critical", "rpc-test"))

	ctx := new(executor.Context)
	ctx.ErrorInput = make(chan error, 10)
	ctx.ExecutionResult = rpc.NewResult(bigRes.Bytes(), nil, 10)
	err := c.PostTransaction(s, ctx)
	if err != nil {
		t.Errorf("unexpected error in post transaction; %v", err)
	}

}

func TestRPCComparator_PostTransactionFailsWhenContinueOnFailureIsNotEnabled(t *testing.T) {
	cfg := &utils.Config{}
	cfg.Validate = true
	cfg.ContinueOnFailure = false

	bigRes, _ := new(big.Int).SetString("1", 16)
	rec, _ := json.Marshal(hexZero)

	data := &rpc.RequestAndResults{
		Query: &rpc.Body{
			MethodBase: "getBalance",
		},
		Response: &rpc.Response{
			Result: rec,
		},
		IsRecovered: true,
	}

	s := executor.State[*rpc.RequestAndResults]{
		Data: data,
	}

	res := rpc.NewResult(bigRes.Bytes(), nil, 10)
	c := makeRPCComparator(cfg, logger.NewLogger("critical", "rpc-test"))
	err := c.PostTransaction(s, &executor.Context{ExecutionResult: res})
	if err == nil {
		t.Errorf("post transaction must return error; %v", err)
	}

}

// Test_compareBalanceOK tests compare func for getBalance method
// It expects no error since results are same
func Test_compareBalanceOK(t *testing.T) {
	bigRes, _ := new(big.Int).SetString("1", 16)
	rec, _ := json.Marshal(hexOne)

	data := &rpc.RequestAndResults{
		Query: &rpc.Body{
			Method: "ftm_getBalance",
		},
		Response: &rpc.Response{
			Result: rec,
		},
	}

	res := rpc.NewResult(bigRes.Bytes(), nil, 10)
	err := compareBalance(res, data, 0)
	if err != nil {
		t.Errorf("error must be nil; err: %v", err)
	}

}

// Test_compareBalanceErrorNoMatchingResult tests compare func for getBalance method
// It expects an error of no matching results since results are different
func Test_compareBalanceErrorNoMatchingResult(t *testing.T) {
	bigRes, _ := new(big.Int).SetString("1", 16)
	rec, _ := json.Marshal(hexZero)

	data := &rpc.RequestAndResults{
		Query: &rpc.Body{
			Method: "ftm_getBalance",
		},
		Response: &rpc.Response{
			Result: rec,
		},
	}
	res := rpc.NewResult(bigRes.Bytes(), nil, 10)
	err := compareBalance(res, data, 0)
	if err == nil {
		t.Errorf("error must not be nil; err: %v", err)
		return
	}

	if err.typ != noMatchingResult {
		t.Errorf("error must be type 'noMatchingResult'; err: %v", err)
	}

}

// Test_compareTransactionCountOK tests compare func for getTransactionCount method
// It expects no error since results are same
func Test_compareTransactionCountOK(t *testing.T) {
	rec, _ := json.Marshal(hexOne)

	data := &rpc.RequestAndResults{
		Query: &rpc.Body{
			Method: "ftm_getTransactionCount",
		},
		Response: &rpc.Response{
			Result: rec,
		},
	}

	res := rpc.NewResult(littleendian.Uint64ToBytes(1), nil, 10)
	err := compareTransactionCount(res, data, 0)

	if err != nil {
		t.Errorf("error must be nil; err: %v", err)
	}

}

// Test_compareTransactionCountErrorNoMatchingResult tests compare func for getTransactionCount method
// It expects an error of no matching results since results are different
func Test_compareTransactionCountErrorNoMatchingResult(t *testing.T) {
	rec, _ := json.Marshal(hexZero)

	data := &rpc.RequestAndResults{
		Query: &rpc.Body{
			Method: "ftm_getTransactionCount",
		},
		Response: &rpc.Response{
			Result: rec,
		},
	}

	res := rpc.NewResult(littleendian.Uint64ToBytes(1), nil, 10)
	err := compareTransactionCount(res, data, 0)
	if err == nil {
		t.Errorf("error must not be nil; err: %v", err)
		return
	}

	if err.typ != noMatchingResult {
		t.Errorf("error must be type 'noMatchingResult'; err: %v", err)
	}

}

// Test_compareCallOK tests compare func for call method
// It expects no error since results are same
func Test_compareCallOK(t *testing.T) {
	rec, _ := json.Marshal(longHexOne)

	data := &rpc.RequestAndResults{
		Query: &rpc.Body{
			Method: "ftm_call",
		},
		Response: &rpc.Response{
			Result: rec,
		},
	}

	res := rpc.NewResult(hexutils.HexToBytes(strings.TrimPrefix(longHexOne, "0x")), nil, 10)
	err := compareCall(res, data, 0)
	if err != nil {
		t.Errorf("error must be nil; err: %v", err)
	}
}

// Test_compareCallErrorNoMatchingResult tests compare func for call method
// It expects an error of no matching results since results are different
func Test_compareCallErrorNoMatchingResult(t *testing.T) {
	rec, _ := json.Marshal(longHexOne)

	data := &rpc.RequestAndResults{
		Query: &rpc.Body{
			Method: "ftm_call",
		},
		Response: &rpc.Response{
			Result: rec,
		},
	}

	res := rpc.NewResult(hexutils.HexToBytes(strings.TrimPrefix(longHexZero, "0x")), nil, 10)
	err := compareCall(res, data, 0)
	if err == nil {
		t.Errorf("error must not be nil; err: %v", err)
		return
	}

	if err.typ != noMatchingResult {
		t.Errorf("error must be type 'noMatchingResult'; err: %v", err)
	}

}

// Test_compareCallErrorExpectedResultGotErr tests compare func for call method
// It expects an error of "expected valid result, got error" since recorded data is a valid result but EVM returns error
func Test_compareCallErrorExpectedResultGotErr(t *testing.T) {
	data := &rpc.RequestAndResults{
		Query: &rpc.Body{
			Method: "eth_call",
		},
		Response: &rpc.Response{
			Result: []byte(hexOne),
		},
	}

	res := rpc.NewResult(nil, errors.New("err"), 10)
	err := compareCall(res, data, 0)
	if err == nil {
		t.Errorf("error must not be nil; err: %v", err)
		return
	}

	if err.typ != expectedResultGotError {
		t.Errorf("error must be type 'expectedResultGotError'; err: %v", err)
	}

}

// Test_compareCallErrorExpectedErrGotResult tests compare func for call method
// It expects an error of "expected error, got valid result" since recorded data is an error but EVM returns valid result
func Test_compareCallErrorExpectedErrGotResult(t *testing.T) {
	data := &rpc.RequestAndResults{
		Query: &rpc.Body{
			Method: "eth_call",
		},
		Error: &rpc.ErrorResponse{
			Error: rpc.ErrorMessage{
				Code:    -32000,
				Message: "error",
			},
		},
	}

	res := rpc.NewResult(hexutils.HexToBytes(strings.TrimPrefix(longHexZero, "0x")), nil, 10)
	err := compareCall(res, data, 0)
	if err == nil {
		t.Errorf("error must not be null")
		return
	}

	if err.typ != expectedErrorGotResult {
		t.Errorf("error must be type 'expectedErrorGotResult'; err: %v", err)
	}

}

// Test_compareEstimateGasOK tests compare func for estimateGas method
// It expects no error since results are same
func Test_compareEstimateGasOK(t *testing.T) {
	rec, _ := json.Marshal(hexOne)

	data := &rpc.RequestAndResults{
		Query: &rpc.Body{
			Method: "eth_estimateGas",
		},
		Response: &rpc.Response{
			Result: rec,
		},
	}

	res := rpc.NewResult(littleendian.Uint64ToBytes(uint64(1)), nil, 10)
	err := compareEstimateGas(res, data, 0)
	if err != nil {
		t.Errorf("error must be nil; err: %v", err)
	}
}

// Test_compareEstimateGasErrorNoMatchingResult tests compare func for estimateGas method
// It expects an error of no matching results since results are different
func Test_compareEstimateGasErrorNoMatchingResult(t *testing.T) {
	rec, _ := json.Marshal(hexOne)

	data := &rpc.RequestAndResults{
		Query: &rpc.Body{
			Method: "eth_estimateGas",
		},
		Response: &rpc.Response{
			Result: rec,
		},
	}

	res := rpc.NewResult(littleendian.Uint64ToBytes(uint64(0)), nil, 10)
	err := compareEstimateGas(res, data, 0)
	if err == nil {
		t.Errorf("error must not be null")
		return
	}

	if err.typ != noMatchingResult {
		t.Errorf("error must be type 'noMatchingResult'; err: %v", err)
	}

}

// Test_compareEstimateGasErrorExpectedResultGotErr tests compare func for estimateGas method
// It expects an error of "expected valid result, got error" since recorded data is a valid result but EVM returns error
func Test_compareEstimateGasErrorExpectedResultGotErr(t *testing.T) {
	data := &rpc.RequestAndResults{
		Query: &rpc.Body{
			Method: "eth_estimateGas",
		},
		Response: &rpc.Response{
			Result: []byte(hexOne),
		},
	}

	res := rpc.NewResult(nil, errors.New("error"), 10)
	err := compareEstimateGas(res, data, 0)
	if err == nil {
		t.Errorf("error must be nil; err: %v", err)
		return
	}

	if err.typ != expectedResultGotError {
		t.Errorf("error must be type 'expectedResultGotError'; err: %v", err)
	}
}

// Test_compareEstimateGasErrorExpectedErrGotResult tests compare func for estimateGas method
// It expects an error of "expected error, got valid result" since recorded data is an error but EVM returns valid result
func Test_compareEstimateGasErrorExpectedErrGotResult(t *testing.T) {
	data := &rpc.RequestAndResults{
		Query: &rpc.Body{
			Method: "eth_estimateGas",
		},
		Error: &rpc.ErrorResponse{
			Error: rpc.ErrorMessage{
				Code:    1000,
				Message: "error",
			},
		},
	}
	res := rpc.NewResult(littleendian.Uint64ToBytes(uint64(0)), nil, 10)
	err := compareEstimateGas(res, data, 0)
	if err == nil {
		t.Errorf("error must not be null")
		return
	}

	if err.typ != expectedErrorGotResult {
		t.Errorf("error must be type 'expectedErrorGotResult'; err: %v", err)
	}

}

// Test_compareCodeOK tests compare func for getCode method
// It expects no error since results are same
func Test_compareCodeOK(t *testing.T) {
	rec, _ := json.Marshal(longHexOne)

	data := &rpc.RequestAndResults{
		Query: &rpc.Body{
			Method: "eth_getCode",
		},
		Response: &rpc.Response{
			Result: rec,
		},
	}

	res := rpc.NewResult(hexutils.HexToBytes(strings.TrimPrefix(longHexOne, "0x")), nil, 10)
	err := compareCode(res, data, 0)
	if err != nil {
		t.Errorf("error must be nil; err: %v", err)
	}
}

// Test_compareCodeErrorNoMatchingResult tests compare func for getCode method
// It expects an error of no matching results since results are different
func Test_compareCodeErrorNoMatchingResult(t *testing.T) {
	rec, _ := json.Marshal(longHexOne)

	data := &rpc.RequestAndResults{
		Query: &rpc.Body{
			Method: "eth_getCode",
		},
		Response: &rpc.Response{
			Result: rec,
		},
	}

	res := rpc.NewResult(hexutils.HexToBytes(strings.TrimPrefix(longHexZero, "0x")), nil, 10)
	err := compareCode(res, data, 0)
	if err == nil {
		t.Errorf("error must not be nil; err: %v", err)
		return
	}

	if err.typ != noMatchingResult {
		t.Errorf("error must be type 'noMatchingResult'; err: %v", err)
	}

}

// Test_compareStorageAtOK tests compare func for getStorageAt method
// It expects no error since results are same
func Test_compareStorageAtOK(t *testing.T) {
	rec, _ := json.Marshal(longHexOne)

	data := &rpc.RequestAndResults{
		Query: &rpc.Body{
			Method: "eth_getStorageAt",
		},
		Response: &rpc.Response{
			Result: rec,
		},
	}

	res := rpc.NewResult(hexutils.HexToBytes(strings.TrimPrefix(longHexOne, "0x")), nil, 10)
	err := compareStorageAt(res, data, 0)
	if err != nil {
		t.Errorf("error must be nil; err: %v", err)
	}
}

// Test_compareStorageAtErrorNoMatchingResult tests compare func for getStorageAt method
// It expects an error of no matching results since results are different
func Test_compareStorageAtErrorNoMatchingResult(t *testing.T) {
	rec, _ := json.Marshal(longHexOne)

	data := &rpc.RequestAndResults{
		Query: &rpc.Body{
			Method: "eth_getStorageAt",
		},
		Response: &rpc.Response{
			Result: rec,
		},
	}

	res := rpc.NewResult(hexutils.HexToBytes(strings.TrimPrefix(longHexZero, "0x")), nil, 10)

	err := compareStorageAt(res, data, 0)
	if err == nil {
		t.Errorf("error must not be nil; err: %v", err)
		return
	}

	if err.typ != noMatchingResult {
		t.Errorf("error must be type 'noMatchingResult'; err: %v", err)
	}

}
