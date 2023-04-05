package apireplay

import (
	"encoding/json"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

// EVMErrors decode error code into string with which is compared with recorded error message
var EVMErrors = map[int]string{
	-32000: "execution reverted",
	-32602: "invalid argument",
	3:      "execution reverted",
}

//todo
//var EVMErrors = map[int][]string{
//	-32000: {"execution reverted", "invalid opcode"},
//	-32602: {"invalid argument"},
//	3:      {"execution reverted"},
//}

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

	if len(data.Recorded.Result) > 5 {
		fmt.Println("a")
	}

	// did we record an error?
	if data.Recorded.Error != nil {
		return newComparatorError(hexBalance, data.Recorded.Error.Message, data, expectedResultGotError)
	}

	var (
		err            error
		recordedString string
	)

	err = json.Unmarshal(data.Recorded.Result, &recordedString)
	if err != nil {
		return &comparatorError{
			error: err,
			typ:   defaultErrorType,
		}
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
		hexNonce, recordedString string
		err                      error
	)

	bigNonce = new(big.Int).SetUint64(stateNonce)

	builder.WriteString("0x")
	builder.WriteString(bigNonce.Text(16))

	hexNonce = builder.String()

	builder.Reset()

	err = json.Unmarshal(data.Recorded.Result, &recordedString)
	if err != nil {
		return &comparatorError{
			error: err,
			typ:   defaultErrorType,
		}
	}

	if !strings.EqualFold(recordedString, hexNonce) {
		return newComparatorError(hexNonce, recordedString, data, noMatchingResult)
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
	var recordedString, stateString string

	stateString = common.Bytes2Hex(data.StateDB.Result.([]byte))

	builder.WriteString("0x")
	builder.WriteString(stateString)
	stateString = builder.String()

	builder.Reset()

	if data.Recorded.Error != nil {
		return newComparatorError(
			stateString,
			EVMErrors[data.Recorded.Error.Code],
			data,
			expectedErrorGotResult)
	}

	err := json.Unmarshal(data.Recorded.Result, &recordedString)
	if err != nil {
		return &comparatorError{
			error: err,
			typ:   defaultErrorType,
		}
	}

	if !strings.EqualFold(recordedString, stateString) {
		return newComparatorError(
			stateString,
			recordedString,
			data,
			noMatchingResult)
	}

	return nil
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

	if !strings.Contains(data.StateDB.Error.Error(), EVMErrors[data.Recorded.Error.Code]) {
		return newComparatorError(
			data.StateDB.Error,
			EVMErrors[data.Recorded.Error.Code],
			data,
			noMatchingErrors)
	}

	return nil
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
		bigGas *big.Int
		hexGas string
	)

	bigGas = new(big.Int).SetUint64(uint64(stateDBGas))

	builder.WriteString("0x")
	builder.WriteString(bigGas.Text(16))

	hexGas = builder.String()

	builder.Reset()

	// did we record a valid result?
	if data.Recorded.Error != nil {
		return newComparatorError(
			hexGas,
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
		return &comparatorError{
			error: err,
			typ:   defaultErrorType,
		}
	}

	if !strings.EqualFold(recordedString, hexGas) {
		return newComparatorError(hexGas, recordedString, data, noMatchingResult)
	}

	return nil
}

// compareCode compares getCode data recorded on API server with data returned by StateDB
func compareCode(data *OutData, builder *strings.Builder) *comparatorError {
	var (
		recordedString, stateString string
	)

	err := json.Unmarshal(data.Recorded.Result, &recordedString)
	if err != nil {
		return &comparatorError{
			error: err,
			typ:   defaultErrorType,
		}
	}

	stateString = common.Bytes2Hex(data.StateDB.Result.([]byte))

	builder.WriteString("0x")
	builder.WriteString(stateString)
	stateString = builder.String()

	builder.Reset()

	if !strings.EqualFold(recordedString, stateString) {
		return newComparatorError(stateString, recordedString, data, noMatchingResult)
	}

	return nil
}

// compareStorageAt compares getStorageAt data recorded on API server with data returned by StateDB
func compareStorageAt(data *OutData, builder *strings.Builder) *comparatorError {
	var (
		recordedString, stateString string
		err                         error
	)

	err = json.Unmarshal(data.Recorded.Result, &recordedString)
	if err != nil {
		return &comparatorError{
			error: err,
			typ:   defaultErrorType,
		}
	}

	stateString = common.Bytes2Hex(data.StateDB.Result.([]byte))

	builder.WriteString("0x")
	builder.WriteString(stateString)
	stateString = builder.String()

	builder.Reset()

	if !strings.EqualFold(recordedString, stateString) {
		return newComparatorError(stateString, recordedString, data, noMatchingResult)
	}

	return nil
}
