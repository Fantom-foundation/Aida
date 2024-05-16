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
	"fmt"
	"math/big"
	"strconv"
	"strings"

	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/executor/extension"
	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/rpc"
	"github.com/Fantom-foundation/Aida/txcontext"
	"github.com/Fantom-foundation/Aida/utils"
	"github.com/Fantom-foundation/lachesis-base/common/littleendian"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

type comparatorErrorType byte

const (
	defaultErrorType comparatorErrorType = iota
	noMatchingResult
	noMatchingErrors
	expectedErrorGotResult
	expectedResultGotError
	unexpectedDataType
	cannotUnmarshalResult
	cannotSendRpcRequest
	internalError
)

const (
	// internalErrorCode is created when RPC-API could not execute request
	// - for purpose of replay, this error is not critical and does not cause an error
	internalErrorCode = -32603

	invalidArgumentErrCode = -32602
	// there are multiple types of execution reverted error codes
	executionRevertedA = -32603
	executionRevertedB = -32000
	executionRevertedC = 3
)

// EvmErrors decode error code into string with which is compared with recorded error message
var EvmErrors = map[int][]string{
	executionRevertedA: {"execution reverted"},

	executionRevertedB: {
		"execution reverted",
		"invalid opcode",
		"invalid code",
		"insufficient balance for transfer",
		"insufficient funds",
		"gas required exceeds allowance",
		"out of gas",
	},
	executionRevertedC: {"execution reverted"},

	invalidArgumentErrCode: {"invalid argument"},
}

// comparatorError is returned when state.Data returned by StateDB does not match recorded state.Data
type comparatorError struct {
	error
	typ comparatorErrorType
}

// MakeRpcComparator returns extension which handles comparison of result created by the StateDb and the recording.
// If ContinueOnFailure is enabled errors are being saved and printed after the whole run ends. Otherwise, error is returned.
func MakeRpcComparator(cfg *utils.Config) executor.Extension[*rpc.RequestAndResults] {
	if !cfg.Validate {
		return extension.NilExtension[*rpc.RequestAndResults]{}
	}

	log := logger.NewLogger("INFO", "state-hash-validator")

	return makeRPCComparator(cfg, log)
}

func makeRPCComparator(cfg *utils.Config, log logger.Logger) *rpcComparator {
	return &rpcComparator{cfg: cfg, log: log}
}

type rpcComparator struct {
	extension.NilExtension[*rpc.RequestAndResults]
	cfg                     *utils.Config
	log                     logger.Logger
	numberOfRetriedRequests int
	totalNumberOfRequests   int
	numberOfErrors          int
}

// PostTransaction compares result with recording. If ContinueOnFailure
// is enabled error is saved. Otherwise, the error is returned.
func (c *rpcComparator) PostTransaction(state executor.State[*rpc.RequestAndResults], ctx *executor.Context) error {
	c.totalNumberOfRequests++

	// pending block numbers are not validatable
	if state.Data.SkipValidation || ctx.ExecutionResult == nil {
		return nil
	}

	compareErr := compare(ctx.ExecutionResult, state)
	if compareErr != nil {
		// request method base 'call' cannot be resent, because we need timestamp of the block that executed
		// this request. As of right now there we cannot get the timestamp, hence we skip these requests
		if state.Data.Query.MethodBase == "call" {
			// Only requests containing an error result are not being treated as data
			// mismatch, request with a non-error result are recorded correctly
			if state.Data.Error != nil {
				return nil
			} else {
				return compareErr
			}
		}
		// lot errors are recorded wrongly, for this case we resend the request and compare it again
		if !state.Data.IsRecovered {
			c.log.Debugf("retrying %v request", state.Data.Query.Method)
			c.numberOfRetriedRequests++
			c.log.Debugf("current ration retried against total %v/%v", c.numberOfRetriedRequests, c.totalNumberOfRequests)
			state.Data.IsRecovered = true
			if err := c.resendRequest(ctx.ExecutionResult, state); err != nil {
				return err
			}
			compareErr = compare(ctx.ExecutionResult, state)
			if compareErr == nil {
				return nil
			}
		}

		if compareErr.typ == cannotUnmarshalResult {
			return nil
		}

		if !c.cfg.ContinueOnFailure {
			return compareErr
		}

		ctx.ErrorInput <- compareErr
		c.numberOfErrors++

		// endless run
		if c.cfg.MaxNumErrors == 0 {
			return nil
		}

		if c.numberOfErrors >= c.cfg.MaxNumErrors {
			return compareErr
		}
	}

	return nil
}

func compare(result txcontext.Result, state executor.State[*rpc.RequestAndResults]) *comparatorError {
	switch state.Data.Query.MethodBase {
	case "getBalance":
		return compareBalance(result, state.Data, state.Block)
	case "getTransactionCount":
		return compareTransactionCount(result, state.Data, state.Block)
	case "call":
		return compareCall(result, state.Data, state.Block)
	case "estimateGas":
		// estimateGas is currently not suitable for replay since the estimation  in geth is always calculated
		// for current state that means recorded result and result returned by StateDB are not comparable
	case "getCode":
		return compareCode(result, state.Data, state.Block)
	case "getStorageAt":
		return compareStorageAt(result, state.Data, state.Block)
	}

	return nil
}

func (c *rpcComparator) resendRequest(result txcontext.Result, state executor.State[*rpc.RequestAndResults]) *comparatorError {
	var payload []byte

	if state.Data.Response != nil {
		payload = state.Data.Response.Payload
		c.log.Debugf("Previously recorded: %v", state.Data.Response.Result)
	} else {
		payload = state.Data.Error.Payload
		c.log.Debugf("Previously recorded: %v", state.Data.Error.Error)
	}

	retriedReq := utils.JsonRPCRequest{
		Method:  state.Data.Query.Method,
		Params:  state.Data.Query.Params,
		ID:      0,
		JSONRPC: "2.0",
	}

	c.log.Debugf("Previous params: %v", state.Data.Query.Params)

	b := hexutil.EncodeUint64(uint64(state.Data.RequestedBlock))
	l := len(retriedReq.Params)
	// this is a corner case where request does not have block number
	if l <= 1 {
		retriedReq.Params = append(retriedReq.Params, b)
	} else {
		retriedReq.Params[len(retriedReq.Params)-1] = b
	}

	c.log.Debugf("Retried params: %v", retriedReq.Params)
	// we only record on mainnet, so we can safely put mainnet chainID constant here
	m, err := utils.SendRpcRequest(retriedReq, utils.MainnetChainID)
	if err != nil {
		return newComparatorError(result, nil, nil, state.Data, state.Block, cannotSendRpcRequest)
	}

	// remove the data
	state.Data.Response = nil
	state.Data.Error = nil

	s, ok := m["result"].(string)
	if ok { // valid result
		resentResult, err := json.Marshal(s)
		if err != nil {
			return newComparatorError(result, nil, nil, state.Data, state.Block, cannotUnmarshalResult)
		}

		state.Data.Response = &rpc.Response{
			Version: "2.0",
			ID:      json.RawMessage{1},
			BlockID: uint64(state.Data.RequestedBlock),
			Result:  resentResult,
			Payload: payload,
		}
	} else { // error result
		resMap, ok := m["error"].(map[string]interface{})
		if !ok {
			return newComparatorError(result, nil, nil, state.Data, state.Block, cannotUnmarshalResult)
		}

		// rpc sometimes returns float
		var code int
		c, ok := resMap["code"].(float64)
		if ok {
			code = int(c)
		} else {
			code = resMap["code"].(int)
		}

		state.Data.Error = &rpc.ErrorResponse{
			Version: "2.0",
			Id:      json.RawMessage{1},
			BlockID: uint64(state.Data.RequestedBlock),
			Error: rpc.ErrorMessage{
				Code:    code,
				Message: resMap["message"].(string),
			},
			Payload: payload,
		}
	}

	return nil
}

// compareBalance compares getBalance data recorded on API server with data returned by StateDB
func compareBalance(result txcontext.Result, data *rpc.RequestAndResults, block int) *comparatorError {
	res, _ := result.GetRawResult()
	stateBalance := new(big.Int).SetBytes(res)

	if data.Error != nil {
		if data.Error.Error.Code == internalErrorCode {
			return newComparatorError(result, stateBalance.Text(16), data.Error.Error.Message, data, block, internalError)
		}
		return newComparatorError(result, stateBalance.Text(16), data.Error.Error.Message, data, block, expectedErrorGotResult)
	}

	// no error
	var recordedString string
	err := json.Unmarshal(data.Response.Result, &recordedString)
	if err != nil {
		return newComparatorError(result, stateBalance.Text(16), string(data.Response.Result), data, block, cannotUnmarshalResult)
	}
	recordedString = strings.TrimPrefix(recordedString, "0x")

	recordedBalance, ok := new(big.Int).SetString(recordedString, 16)
	if !ok {
		return newComparatorError(result, stateBalance.Text(16), string(data.Response.Result), data, block, cannotUnmarshalResult)
	}

	if stateBalance.Cmp(recordedBalance) != 0 {
		return newComparatorError(result, stateBalance.Text(16), recordedBalance.Text(16), data, block, noMatchingResult)
	}

	return nil

}

// compareTransactionCount compares getTransactionCount data recorded on API server with data returned by StateDB
func compareTransactionCount(result txcontext.Result, data *rpc.RequestAndResults, block int) *comparatorError {
	res, _ := result.GetRawResult()
	stateNonce := littleendian.BytesToUint64(res)

	if data.Error != nil {
		if data.Error.Error.Code == internalErrorCode {
			return newComparatorError(result, stateNonce, data.Error.Error.Message, data, block, internalError)
		}
		return newComparatorError(result, stateNonce, data.Error.Error.Message, data, block, expectedErrorGotResult)
	}

	var recordedString string
	// no error
	err := json.Unmarshal(data.Response.Result, &recordedString)
	if err != nil {
		return newComparatorError(result, stateNonce, string(data.Response.Result), data, block, cannotUnmarshalResult)
	}

	recordedNonce, err := hexutil.DecodeUint64(recordedString)
	if err != nil {
		return newComparatorError(result, recordedNonce, string(data.Response.Result), data, block, cannotUnmarshalResult)
	}

	if stateNonce != recordedNonce {
		return newComparatorError(result, recordedNonce, recordedString, data, block, noMatchingResult)
	}

	return nil
}

// compareCall compares call data recorded on API server with data returned by StateDB
func compareCall(result txcontext.Result, data *rpc.RequestAndResults, block int) *comparatorError {
	res, err := result.GetRawResult()
	if res != nil {
		return compareCallStateDbResult(result, res, data, block)
	}

	if err != nil {
		return compareEVMStateDBError(result, err, data, block)
	}

	return newUnexpectedDataTypeErr(data)
}

// compareCallStateDbResult compares valid call result recorded on API server with valid result returned by StateDb
func compareCallStateDbResult(result txcontext.Result, res []byte, data *rpc.RequestAndResults, block int) *comparatorError {
	dbString := hexutil.Encode(res)

	if data.Error == nil {
		var recordedString string
		err := json.Unmarshal(data.Response.Result, &recordedString)
		if err != nil {
			return newComparatorError(result, dbString, string(data.Response.Result), data, block, cannotUnmarshalResult)
		}

		// results do not match
		if !strings.EqualFold(recordedString, dbString) {
			return newComparatorError(result,
				dbString,
				recordedString,
				data,
				block,
				noMatchingResult)
		}

		return nil
	}

	// internal error?
	if data.Error.Error.Code == internalErrorCode {
		return newComparatorError(result, dbString, data.Error.Error, data, block, internalError)
	}

	var msg string

	// do we know the error?
	errs, ok := EvmErrors[data.Error.Error.Code]
	if !ok {
		msg = fmt.Sprintf("unknown error code: %v", data.Error.Error.Code)
	} else {

		// we could have potentially recorded a request with invalid arguments
		// - this is not checked in execution, hence StateDB returns a valid result.
		// For this we exclude any invalid requests when getting unmatched results
		if data.Error.Error.Code == invalidArgumentErrCode {
			return nil
		}

		builder := new(strings.Builder)

		// more error messages for one code?
		for i, e := range errs {
			builder.WriteString(e)
			if len(errs) > i {
				builder.WriteString(" or ")
			}
		}
		msg = builder.String()
	}

	return newComparatorError(result,
		dbString,
		msg,
		data,
		block,
		expectedErrorGotResult)
}

// compareEVMStateDBError compares error returned from EvmExecutor with recorded data
func compareEVMStateDBError(result txcontext.Result, err error, data *rpc.RequestAndResults, block int) *comparatorError {
	if data.Error == nil {
		return newComparatorError(result,
			err,
			data.Response.Result,
			data,
			block,
			expectedResultGotError)
	}

	for _, e := range EvmErrors[data.Error.Error.Code] {
		if strings.Contains(err.Error(), e) {
			return nil
		}
	}

	if data.Error.Error.Code == internalErrorCode {
		return newComparatorError(result, err, data.Error.Error, data, block, internalError)
	}

	builder := new(strings.Builder)

	builder.WriteString("one of these error messages: ")

	for i, e := range EvmErrors[data.Error.Error.Code] {
		builder.WriteString(e)
		if i < len(EvmErrors[data.Error.Error.Code]) {
			builder.WriteString(" or ")
		}
	}

	msg := builder.String()

	return newComparatorError(result,
		err,
		msg,
		data,
		block,
		noMatchingErrors)
}

// compareEstimateGas compares recorded data for estimateGas method with result from StateDB
func compareEstimateGas(result txcontext.Result, data *rpc.RequestAndResults, block int) *comparatorError {
	res, err := result.GetRawResult()
	if res != nil {
		return compareEstimateGasStateDBResult(result, res, data, block)
	}

	if err != nil {
		return compareEVMStateDBError(result, err, data, block)
	}

	return nil
}

// compareEstimateGasStateDBResult compares estimateGas data recorded on API server with data returned by StateDB
func compareEstimateGasStateDBResult(result txcontext.Result, res []byte, data *rpc.RequestAndResults, block int) *comparatorError {
	stateDBGas := littleendian.BytesToUint64(res)

	// did we receive an error
	if data.Error != nil {
		// internal error?
		if data.Error.Error.Code == internalErrorCode {
			return newComparatorError(result, stateDBGas, data.Error.Error, data, block, internalError)
		}

		return newComparatorError(result,
			stateDBGas,
			EvmErrors[data.Error.Error.Code],
			data,
			block,
			expectedErrorGotResult)
	}

	var (
		err            error
		recordedString string
	)

	// no error
	err = json.Unmarshal(data.Response.Result, &recordedString)
	if err != nil {
		return newComparatorError(result, stateDBGas, string(data.Response.Result), data, block, cannotUnmarshalResult)
	}

	recordedResult, err := hexutil.DecodeUint64(recordedString)
	if err != nil {
		return newComparatorError(result, recordedResult, string(data.Response.Result), data, block, cannotUnmarshalResult)
	}

	if stateDBGas != recordedResult {
		return newComparatorError(result, recordedResult, recordedString, data, block, noMatchingResult)
	}

	return nil
}

// compareCode compares getCode data recorded on API server with data returned by StateDB
func compareCode(result txcontext.Result, data *rpc.RequestAndResults, block int) *comparatorError {
	res, _ := result.GetRawResult()
	dbString := hexutil.Encode(res)

	// did we data an error?
	if data.Error != nil {
		// internal error?
		if data.Error.Error.Code == internalErrorCode {
			return newComparatorError(result, dbString, data.Error.Error, data, block, internalError)
		}
		return newComparatorError(result, dbString, data.Error.Error, data, block, expectedErrorGotResult)
	}

	var recordedString string

	// no error
	err := json.Unmarshal(data.Response.Result, &recordedString)
	if err != nil {
		return newComparatorError(result, dbString, string(data.Response.Result), data, block, cannotUnmarshalResult)
	}

	if !strings.EqualFold(recordedString, dbString) {
		return newComparatorError(result, dbString, recordedString, data, block, noMatchingResult)
	}

	return nil
}

// compareStorageAt compares getStorageAt data recorded on API server with data returned by StateDB
func compareStorageAt(result txcontext.Result, data *rpc.RequestAndResults, block int) *comparatorError {
	res, _ := result.GetRawResult()
	dbString := hexutil.Encode(res)

	if data.Error != nil {
		// internal error?
		if data.Error.Error.Code == internalErrorCode {
			return newComparatorError(result, dbString, data.Error, data, block, internalError)
		}
		return newComparatorError(result, dbString, data.Error, data, block, internalError)
	}

	var recordedString string

	// no error
	err := json.Unmarshal(data.Response.Result, &recordedString)
	if err != nil {
		return newComparatorError(result, dbString, string(data.Response.Result), data, block, cannotUnmarshalResult)
	}

	if !strings.EqualFold(recordedString, dbString) {
		return newComparatorError(result, dbString, recordedString, data, block, noMatchingResult)
	}

	return nil
}

// newComparatorError returns new comparatorError with given StateDB and recorded data based on the typ.
func newComparatorError(result txcontext.Result, stateDB, expected any, data *rpc.RequestAndResults, block int, typ comparatorErrorType) *comparatorError {
	switch typ {
	case noMatchingResult:
		return newNoMatchingResultErr(stateDB, expected, data, block)
	case noMatchingErrors:
		return newNoMatchingErrorsErr(stateDB, expected, data, block)
	case expectedResultGotError:
		return newExpectedResultGotErrorErr(stateDB, expected, data, block)
	case expectedErrorGotResult:
		return newExpectedErrorGotResultErr(stateDB, expected, data, block)
	case unexpectedDataType:
		return newUnexpectedDataTypeErr(data)
	case cannotUnmarshalResult:
		return newCannotUnmarshalResult(result, data, block)
	case internalError:
		// internalError is caused by opera, adding this to the error list does not make sense
		return nil
	case cannotSendRpcRequest:
		return newCannotSendRPCRequestErr(result, data, block)
	default:
		return &comparatorError{
			error: fmt.Errorf("default error:\n%v", data),
			typ:   0,
		}
	}
}

func newCannotSendRPCRequestErr(result txcontext.Result, data *rpc.RequestAndResults, block int) *comparatorError {
	res, err := result.GetRawResult()
	return &comparatorError{
		error: fmt.Errorf("could not resend request to rpc:"+
			"\nMethod: %v"+
			"\nBlockID: 0x%v"+
			"\n\tStateDB result: %v"+
			"\n\tStateDB err: %v"+
			"\n\tExpected result: %v"+
			"\n\tExpected err: %v"+
			"\n\nParams: %v", data.Query.Method, strconv.FormatInt(int64(block), 16), hexutil.Encode(res), err, data.Response, data.Error, string(data.ParamsRaw)),
		typ: cannotSendRpcRequest,
	}
}

// newCannotUnmarshalResult returns new comparatorError
// It is returned when json.Unmarshal returned an error
func newCannotUnmarshalResult(result txcontext.Result, data *rpc.RequestAndResults, block int) *comparatorError {
	res, err := result.GetRawResult()
	return &comparatorError{
		error: fmt.Errorf("cannot unmarshal result, returning every possible data"+
			"\nMethod: %v"+
			"\nBlockID: 0x%v"+
			"\n\tStateDB result: %v"+
			"\n\tStateDB err: %v"+
			"\n\tRecorded result: %v"+
			"\n\tRecorded err: %v"+
			"\n\nParams: %v", data.Query.Method, strconv.FormatInt(int64(block), 16), hexutil.Encode(res), err, data.Response, data.Error, string(data.ParamsRaw)),
		typ: cannotUnmarshalResult,
	}
}

// newNoMatchingResultErr returns new comparatorError
// It is returned when StateDB result does not match with recorded result
func newNoMatchingResultErr(stateDBData, expectedData any, data *rpc.RequestAndResults, block int) *comparatorError {
	return &comparatorError{
		error: fmt.Errorf("result do not match"+
			"\nMethod: %v"+
			"\nBlockID: 0x%v"+
			"\n\tCarmen: %v"+
			"\n\tRecorded: %v"+
			"\n\n\tParams: %v", data.Query.Method, strconv.FormatInt(int64(block), 16), stateDBData, expectedData, string(data.ParamsRaw)),
		typ: noMatchingResult,
	}
}

// newNoMatchingErrorsErr returns new comparatorError
// It is returned when StateDB error does not match with recorded error
func newNoMatchingErrorsErr(stateDBError, expectedError any, data *rpc.RequestAndResults, block int) *comparatorError {
	return &comparatorError{
		error: fmt.Errorf("errors do not match"+
			"\nMethod: %v"+
			"\nBlockID: 0x%v"+
			"\n\tCarmen: %v"+
			"\n\tRecorded: %v"+
			"\n\nParams: %v", data.Query.Method, strconv.FormatInt(int64(block), 16), stateDBError, expectedError, string(data.ParamsRaw)),
		typ: noMatchingErrors,
	}
}

// newExpectedResultGotErrorErr returns new comparatorError
// It is returned when StateDB returns an error but expected return is a valid result
func newExpectedResultGotErrorErr(stateDBError, expectedResult any, data *rpc.RequestAndResults, block int) *comparatorError {
	return &comparatorError{
		error: fmt.Errorf("expected valid result but StateDB returned err"+
			"\nMethod: %v"+
			"\nBlockID: 0x%v"+
			"\n\tCarmen: %v"+
			"\n\tRecorded: %v"+
			"\n\nParams: %v", data.Query.Method, strconv.FormatInt(int64(block), 16), stateDBError, expectedResult, string(data.ParamsRaw)),
		typ: expectedResultGotError,
	}
}

// newExpectedErrorGotResultErr returns new comparatorError
// It is returned when StateDB returns a valid error but expected return is an error
func newExpectedErrorGotResultErr(stateDBResult, expectedError any, data *rpc.RequestAndResults, block int) *comparatorError {
	return &comparatorError{
		error: fmt.Errorf("expected error but StateDB returned valid result"+
			"\nMethod: %v"+
			"\nBlockID: 0x%v"+
			"\n\tCarmen: %v"+
			"\n\tRecorded: %v"+
			"\n\nParams: %v", data.Query.Method, strconv.FormatInt(int64(block), 16), stateDBResult, expectedError, string(data.ParamsRaw)),
		typ: expectedErrorGotResult,
	}
}

// newUnexpectedDataTypeErr returns comparatorError
// It is returned when Comparator is given unexpected data type in result from StateDB
func newUnexpectedDataTypeErr(data *rpc.RequestAndResults) *comparatorError {
	var res any
	if data.Response != nil {
		res = string(data.Response.Result)
	} else {
		res = string(rune(data.Error.Error.Code)) + ": " + data.Error.Error.Message
	}
	return &comparatorError{
		error: fmt.Errorf("unexpected data type:\n"+
			"params: %v\n"+
			"method: %v\n"+
			"response: %v\n", string(data.ParamsRaw), data.Query.Method, res),
		typ: unexpectedDataType,
	}
}
