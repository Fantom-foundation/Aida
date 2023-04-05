package apireplay

import (
	"encoding/json"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

// internalErrorCode is created when RPC-API could not execute request
// - for purpose of replay, this error is not critical and is only logged into DEBUG level
const internalErrorCode = -32603

// EVMErrors decode error code into string with which is compared with recorded error message
var EVMErrors = map[int][]string{
	-32000: {"execution reverted", "invalid opcode"},
	-32602: {"invalid argument"},
	3:      {"execution reverted"},
}

// compareBalance compares getBalance data recorded on API server with data returned by StateDB
func compareBalance(data *OutData, builder *strings.Builder) *comparatorError {
	var (
		bigBalance *big.Int
		hexBalance string
		ok         bool
	)

	bigBalance, ok = data.StateDB.Result.(*big.Int)

	if !ok {
		return newUnexpectedDataTypeErr(data)
	}

	builder.WriteString("0x")
	builder.WriteString(bigBalance.Text(16))

	hexBalance = builder.String()

	builder.Reset()

	// did we record an error?
	if data.Recorded.Error != nil {
		// internal error?
		if data.Recorded.Error.Code == internalErrorCode {
			return newComparatorError(hexBalance, data.Recorded.Error.Message, data, internalError)
		}
		return newComparatorError(hexBalance, data.Recorded.Error.Message, data, internalError)
	}

	var (
		err            error
		recordedString string
	)

	err = json.Unmarshal(data.Recorded.Result, &recordedString)
	if err != nil {
		return newComparatorError(hexBalance, string(data.Recorded.Result), data, cannotUnmarshalResult)
	}

	if !strings.EqualFold(recordedString, hexBalance) {
		return newComparatorError(hexBalance, recordedString, data, noMatchingResult)
	}

	return nil

}

// compareTransactionCount compares getTransactionCount data recorded on API server with data returned by StateDB
func compareTransactionCount(data *OutData, builder *strings.Builder) *comparatorError {
	var (
		stateNonce uint64
		ok         bool
	)

	stateNonce, ok = data.StateDB.Result.(uint64)

	if !ok {
		return newUnexpectedDataTypeErr(data)
	}

	var (
		bigNonce                 *big.Int
		dbString, recordedString string
		err                      error
	)

	bigNonce = new(big.Int).SetUint64(stateNonce)

	builder.WriteString("0x")
	builder.WriteString(bigNonce.Text(16))

	dbString = builder.String()

	builder.Reset()

	// did we record an error?
	if data.Recorded.Error != nil {
		// internal error?
		if data.Recorded.Error.Code == internalErrorCode {
			return newComparatorError(dbString, data.Recorded.Error.Message, data, internalError)
		}
		return newComparatorError(dbString, data.Recorded.Error.Message, data, internalError)
	}

	err = json.Unmarshal(data.Recorded.Result, &recordedString)
	if err != nil {
		return newComparatorError(dbString, string(data.Recorded.Result), data, cannotUnmarshalResult)
	}

	if !strings.EqualFold(recordedString, dbString) {
		return newComparatorError(dbString, recordedString, data, noMatchingResult)
	}

	return nil
}

// compareCall compares call data recorded on API server with data returned by StateDB
func compareCall(data *OutData, builder *strings.Builder) *comparatorError {
	// do we have an error from StateDB?
	if data.StateDB.Error != nil {
		return compareEVMStateDBError(data, builder)
	}

	// did StateDB return a valid result?
	if data.StateDB.Result != nil {
		return compareCallStateDBResult(data, builder)
	}

	return newUnexpectedDataTypeErr(data)
}

// compareCallStateDBResult compares valid call result recorded on API server with valid result returned by StateDB
func compareCallStateDBResult(data *OutData, builder *strings.Builder) *comparatorError {
	var recordedString, dbString string

	// create proper hex string from statedb result

	dbString = common.Bytes2Hex(data.StateDB.Result.([]byte))

	builder.WriteString("0x")
	builder.WriteString(dbString)
	dbString = builder.String()

	builder.Reset()

	// we did not record an error
	if data.Recorded.Error == nil {

		err := json.Unmarshal(data.Recorded.Result, &recordedString)
		if err != nil {
			return newComparatorError(dbString, string(data.Recorded.Result), data, cannotUnmarshalResult)
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
	if data.Recorded.Error.Code == internalErrorCode {
		return newComparatorError(dbString, data.Recorded.Error.Message, data, internalError)
	}

	var msg string

	// do we know the error?
	errs, ok := EVMErrors[data.Recorded.Error.Code]
	if !ok {
		msg = fmt.Sprintf("unknown error code: %v", data.Recorded.Error.Code)
	} else {

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

// compareEVMStateDBError compares error returned from EVM with recorded data
func compareEVMStateDBError(data *OutData, builder *strings.Builder) *comparatorError {

	// did we record an error?
	if data.Recorded.Error == nil {
		builder.Write(data.Recorded.Result)
		r := builder.String()

		builder.Reset()

		return newComparatorError(
			data.StateDB.Error,
			r,
			data,
			expectedResultGotError)
	}

	// we did record an error
	for _, e := range EVMErrors[data.Recorded.Error.Code] {
		if strings.Contains(data.StateDB.Error.Error(), e) {
			return nil
		}
	}

	// internal error?
	if data.Recorded.Error.Code == internalErrorCode {
		return newComparatorError(data.StateDB.Error, data.Recorded.Error.Message, data, internalError)
	}

	return newComparatorError(
		data.StateDB.Error,
		EVMErrors[data.Recorded.Error.Code],
		data,
		noMatchingErrors)
}

// compareEstimateGas compares recorded data for estimateGas method with result from StateDB
func compareEstimateGas(data *OutData, builder *strings.Builder) *comparatorError {

	// StateDB returned an error
	if data.StateDB.Error != nil {
		return compareEVMStateDBError(data, builder)
	}

	// StateDB returned a result
	if data.StateDB.Result != nil {
		return compareEstimateGasStateDBResult(data, builder)
	}

	return nil
}

// compareEstimateGasStateDBResult compares estimateGas data recorded on API server with data returned by StateDB
func compareEstimateGasStateDBResult(data *OutData, builder *strings.Builder) *comparatorError {

	stateDBGas, ok := data.StateDB.Result.(hexutil.Uint64)
	if !ok {
		return newUnexpectedDataTypeErr(data)
	}

	var (
		bigGas   *big.Int
		dbString string
	)

	bigGas = new(big.Int).SetUint64(uint64(stateDBGas))

	builder.WriteString("0x")
	builder.WriteString(bigGas.Text(16))

	dbString = builder.String()

	builder.Reset()

	// did we record a valid result?
	if data.Recorded.Error != nil {
		// internal error?
		if data.Recorded.Error.Code == internalErrorCode {
			return newComparatorError(dbString, data.Recorded.Error.Message, data, internalError)
		}

		return newComparatorError(
			dbString,
			EVMErrors[data.Recorded.Error.Code],
			data,
			expectedErrorGotResult)
	}

	var (
		err            error
		recordedString string
	)

	err = json.Unmarshal(data.Recorded.Result, &recordedString)
	if err != nil {
		return newComparatorError(dbString, string(data.Recorded.Result), data, cannotUnmarshalResult)
	}

	if !strings.EqualFold(recordedString, dbString) {
		return newComparatorError(dbString, recordedString, data, noMatchingResult)
	}

	return nil
}

// compareCode compares getCode data recorded on API server with data returned by StateDB
func compareCode(data *OutData, builder *strings.Builder) *comparatorError {
	var (
		recordedString, dbString string
	)

	dbString = common.Bytes2Hex(data.StateDB.Result.([]byte))

	builder.WriteString("0x")
	builder.WriteString(dbString)
	dbString = builder.String()

	builder.Reset()

	// did we record an error?
	if data.Recorded.Error != nil {
		// internal error?
		if data.Recorded.Error.Code == internalErrorCode {
			return newComparatorError(dbString, data.Recorded.Error.Message, data, internalError)
		}
		return newComparatorError(dbString, data.Recorded.Error.Message, data, internalError)
	}

	err := json.Unmarshal(data.Recorded.Result, &recordedString)
	if err != nil {
		return newComparatorError(dbString, string(data.Recorded.Result), data, cannotUnmarshalResult)
	}

	if !strings.EqualFold(recordedString, dbString) {
		return newComparatorError(dbString, recordedString, data, noMatchingResult)
	}

	return nil
}

// compareStorageAt compares getStorageAt data recorded on API server with data returned by StateDB
func compareStorageAt(data *OutData, builder *strings.Builder) *comparatorError {
	var (
		recordedString, dbString string
		err                      error
	)

	dbString = common.Bytes2Hex(data.StateDB.Result.([]byte))

	builder.WriteString("0x")
	builder.WriteString(dbString)
	dbString = builder.String()

	builder.Reset()

	// did we record an error?
	if data.Recorded.Error != nil {
		// internal error?
		if data.Recorded.Error.Code == internalErrorCode {
			return newComparatorError(dbString, data.Recorded.Error.Message, data, internalError)
		}
		return newComparatorError(dbString, data.Recorded.Error.Message, data, internalError)
	}

	err = json.Unmarshal(data.Recorded.Result, &recordedString)
	if err != nil {
		return newComparatorError(dbString, string(data.Recorded.Result), data, cannotUnmarshalResult)
	}

	// no err
	if !strings.EqualFold(recordedString, dbString) {
		return newComparatorError(dbString, recordedString, data, noMatchingResult)
	}

	return nil
}
