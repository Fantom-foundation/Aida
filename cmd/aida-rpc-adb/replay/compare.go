package replay

import (
	"encoding/json"
	"fmt"
	"math/big"
	"strconv"
	"strings"

	"github.com/Fantom-foundation/Aida/rpc_iterator"
	"github.com/Fantom-foundation/Aida/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

const (
	// internalErrorCode is created when RPC-API could not execute request
	// - for purpose of replay, this error is not critical and is only logged into DEBUG level
	internalErrorCode = -32603

	invalidArgumentErrCode = -32602
	// there are multiple types of execution reverted error codes
	executionRevertedA = -32603
	executionRevertedB = -32000
	executionRevertedC = 3
)

// EVMErrors decode error code into string with which is compared with recorded error message
var EVMErrors = map[int][]string{
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

type comparisonData struct {
	block       uint64
	record      *rpc_iterator.RequestWithResponse
	StateDB     *StateDBData
	isRecovered bool
}

// compareBalance compares getBalance data recorded on API server with data returned by StateDB
func compareBalance(data comparisonData) *comparatorError {
	stateBalance, ok := data.StateDB.Result.(*big.Int)
	if !ok {
		return newUnexpectedDataTypeErr(data)
	}

	// did we record an error?
	if data.record.Error != nil {
		if data.record.Error.Error.Code == internalErrorCode {
			return newComparatorError(stateBalance.Text(16), data.record.Error.Error.Message, data, internalError)
		}
		return newComparatorError(stateBalance.Text(16), data.record.Error.Error.Message, data, internalError)
	}

	// no error
	var recordedString string
	err := json.Unmarshal(data.record.Response.Result, &recordedString)
	if err != nil {
		return newComparatorError(stateBalance.Text(16), string(data.record.Response.Result), data, cannotUnmarshalResult)
	}
	recordedString = strings.TrimPrefix(recordedString, "0x")

	recordedBalance, ok := new(big.Int).SetString(recordedString, 16)
	if !ok {
		return newComparatorError(stateBalance.Text(16), string(data.record.Response.Result), data, cannotUnmarshalResult)
	}

	if err != nil {
		return newComparatorError(stateBalance.Text(16), string(data.record.Response.Result), data, cannotUnmarshalResult)
	}

	if stateBalance.Cmp(recordedBalance) != 0 {
		return newComparatorError(stateBalance.Text(16), recordedBalance, data, noMatchingResult)
	}

	return nil

}

// compareTransactionCount compares getTransactionCount data recorded on API server with data returned by StateDB
func compareTransactionCount(data comparisonData) *comparatorError {
	stateNonce, ok := data.StateDB.Result.(uint64)
	if !ok {
		return newUnexpectedDataTypeErr(data)
	}

	var err error

	// did we record an error?
	if data.record.Error != nil {
		if data.record.Error.Error.Code == internalErrorCode {
			return newComparatorError(stateNonce, data.record.Error.Error.Message, data, internalError)
		}
		return newComparatorError(stateNonce, data.record.Error.Error.Message, data, internalError)
	}

	var recordedString string
	// no error
	err = json.Unmarshal(data.record.Response.Result, &recordedString)
	if err != nil {
		return newComparatorError(stateNonce, string(data.record.Response.Result), data, cannotUnmarshalResult)
	}

	recordedNonce, err := hexutil.DecodeUint64(recordedString)
	if err != nil {
		return newComparatorError(recordedNonce, string(data.record.Response.Result), data, cannotUnmarshalResult)
	}

	if stateNonce != recordedNonce {
		return newComparatorError(recordedNonce, recordedString, data, noMatchingResult)
	}

	return nil
}

// compareCall compares call data recorded on API server with data returned by StateDB
func compareCall(data comparisonData) *comparatorError {
	// do we have an error from StateDB?
	if data.StateDB.Error != nil {
		return compareEVMStateDBError(data)
	}

	// did StateDB return a valid result?
	if data.StateDB.Result != nil {
		return compareCallStateDBResult(data)
	}

	return newUnexpectedDataTypeErr(data)
}

// compareCallStateDBResult compares valid call result recorded on API server with valid result returned by StateDB
func compareCallStateDBResult(data comparisonData) *comparatorError {
	dbString := "0x" + common.Bytes2Hex(data.StateDB.Result.([]byte))

	// did we record an error
	if data.record.Error == nil {
		var recordedString string
		err := json.Unmarshal(data.record.Response.Result, &recordedString)
		if err != nil {
			return newComparatorError(dbString, string(data.record.Response.Result), data, cannotUnmarshalResult)
		}

		// results do not match
		if !strings.EqualFold(recordedString, dbString) {
			return newComparatorError(
				dbString,
				recordedString,
				data,
				noMatchingResult)
		}

		return nil
	}

	// we did record an error
	// internal error?
	if data.record.Error.Error.Code == internalErrorCode {
		return newComparatorError(dbString, data.record.Error.Error, data, internalError)
	}

	var msg string

	// do we know the error?
	errs, ok := EVMErrors[data.record.Error.Error.Code]
	if !ok {
		msg = fmt.Sprintf("unknown error code: %v", data.record.Error.Error.Code)
	} else {

		// we could have potentially recorded a request with invalid arguments
		// - this is not checked in execution, hence StateDB returns a valid result.
		// For this we exclude any invalid requests when getting unmatched results
		if data.record.Error.Error.Code == invalidArgumentErrCode {
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

	return newComparatorError(
		dbString,
		msg,
		data,
		expectedErrorGotResult)
}

func (p rpcProcessor) tryRecovery(data comparisonData) *comparatorError {
	payload := utils.JsonRPCRequest{
		Method:  data.record.Query.Method,
		Params:  data.record.Query.Params,
		ID:      0,
		JSONRPC: "2.0",
	}

	// append correct block number
	payload.Params[len(payload.Params)-1] = "0x" + strconv.FormatUint(data.block, 16)

	// we only record on mainnet, so we can safely put mainnet chainID constant here
	m, err := utils.SendRPCRequest(payload, 250)
	if err != nil {
		return newComparatorError(nil, nil, data, cannotSendRPCRequest)
	}

	s, ok := m["result"].(string)
	if !ok {
		return newComparatorError(nil, nil, data, cannotUnmarshalResult)
	}

	result, err := json.Marshal(s)
	if err != nil {
		return newComparatorError(nil, nil, data, cannotUnmarshalResult)
	}

	data.record.Response = &rpc_iterator.Response{
		Version:   data.record.Error.Version,
		ID:        data.record.Error.Id,
		BlockID:   data.record.Error.BlockID,
		Timestamp: data.record.Error.Timestamp,
		Result:    result,
		Payload:   data.record.Error.Payload,
	}

	data.record.Error = nil

	return p.compare(data)
}

// compareEVMStateDBError compares error returned from EVMExecutor with recorded data
func compareEVMStateDBError(data comparisonData) *comparatorError {
	// did we record an error?
	if data.record.Error == nil {
		return newComparatorError(
			data.StateDB.Error,
			data.record.Response.Result,
			data,
			expectedResultGotError)
	}

	// we did record an error
	for _, e := range EVMErrors[data.record.Error.Error.Code] {
		if strings.Contains(data.StateDB.Error.Error(), e) {
			return nil
		}
	}

	// internal error?
	if data.record.Error.Error.Code == internalErrorCode {
		return newComparatorError(data.StateDB.Error, data.record.Error.Error, data, internalError)
	}

	builder := new(strings.Builder)

	builder.WriteString("one of these error messages: ")

	for i, e := range EVMErrors[data.record.Error.Error.Code] {
		builder.WriteString(e)
		if i < len(EVMErrors[data.record.Error.Error.Code]) {
			builder.WriteString(" or ")
		}
	}

	msg := builder.String()

	return newComparatorError(
		data.StateDB.Error,
		msg,
		data,
		noMatchingErrors)
}

// compareEstimateGas compares recorded data for estimateGas method with result from StateDB
func compareEstimateGas(data comparisonData) *comparatorError {

	// StateDB returned an error
	if data.StateDB.Error != nil {
		return compareEVMStateDBError(data)
	}

	// StateDB returned a result
	if data.StateDB.Result != nil {
		return compareEstimateGasStateDBResult(data)
	}

	return nil
}

// compareEstimateGasStateDBResult compares estimateGas data recorded on API server with data returned by StateDB
func compareEstimateGasStateDBResult(data comparisonData) *comparatorError {
	stateDBGas, ok := data.StateDB.Result.(hexutil.Uint64)
	if !ok {
		return newUnexpectedDataTypeErr(data)
	}

	// did we record an error
	if data.record.Error != nil {
		// internal error?
		if data.record.Error.Error.Code == internalErrorCode {
			return newComparatorError(stateDBGas, data.record.Error.Error, data, internalError)
		}

		return newComparatorError(
			stateDBGas,
			EVMErrors[data.record.Error.Error.Code],
			data,
			expectedErrorGotResult)
	}

	var (
		err            error
		recordedString string
	)

	// no error
	err = json.Unmarshal(data.record.Response.Result, &recordedString)
	if err != nil {
		return newComparatorError(stateDBGas, string(data.record.Response.Result), data, cannotUnmarshalResult)
	}

	recordedResult, err := hexutil.DecodeUint64(recordedString)
	if err != nil {
		return newComparatorError(recordedResult, string(data.record.Response.Result), data, cannotUnmarshalResult)
	}

	if uint64(stateDBGas) != recordedResult {
		return newComparatorError(recordedResult, recordedString, data, noMatchingResult)
	}

	return nil
}

// compareCode compares getCode data recorded on API server with data returned by StateDB
func compareCode(data comparisonData) *comparatorError {
	dbString := "0x" + common.Bytes2Hex(data.StateDB.Result.([]byte))

	// did we record an error?
	if data.record.Error != nil {
		// internal error?
		if data.record.Error.Error.Code == internalErrorCode {
			return newComparatorError(dbString, data.record.Error.Error, data, internalError)
		}
		return newComparatorError(dbString, data.record.Error.Error, data, internalError)
	}

	var recordedString string

	// no error
	err := json.Unmarshal(data.record.Response.Result, &recordedString)
	if err != nil {
		return newComparatorError(dbString, string(data.record.Response.Result), data, cannotUnmarshalResult)
	}

	if !strings.EqualFold(recordedString, dbString) {
		return newComparatorError(dbString, recordedString, data, noMatchingResult)
	}

	return nil
}

// compareStorageAt compares getStorageAt data recorded on API server with data returned by StateDB
func compareStorageAt(data comparisonData) *comparatorError {
	dbString := "0x" + common.Bytes2Hex(data.StateDB.Result.([]byte))

	// did we record an error?
	if data.record.Error != nil {
		// internal error?
		if data.record.Error.Error.Code == internalErrorCode {
			return newComparatorError(dbString, data.record.Error.Error, data, internalError)
		}
		return newComparatorError(dbString, data.record.Error.Error, data, internalError)
	}

	var recordedString string

	// no error
	err := json.Unmarshal(data.record.Response.Result, &recordedString)
	if err != nil {
		return newComparatorError(dbString, string(data.record.Response.Result), data, cannotUnmarshalResult)
	}

	if !strings.EqualFold(recordedString, dbString) {
		return newComparatorError(dbString, recordedString, data, noMatchingResult)
	}

	return nil
}
