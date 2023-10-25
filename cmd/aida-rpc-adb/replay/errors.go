package replay

import (
	"encoding/json"
	"fmt"
	"strconv"
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
	cannotSendRPCRequest
	internalError
)

// comparatorError is returned when data returned by StateDB does not match recorded data
type comparatorError struct {
	error
	typ comparatorErrorType
}

// newComparatorError returns new comparatorError with given StateDB and recorded data based on the typ.
func newComparatorError(stateDB, expected any, data comparisonData, typ comparatorErrorType) *comparatorError {
	switch typ {
	case noMatchingResult:
		return newNoMatchingResultErr(stateDB, expected, data)
	case noMatchingErrors:
		return newNoMatchingErrorsErr(stateDB, expected, data)
	case expectedResultGotError:
		return newExpectedResultGotErrorErr(stateDB, expected, data)
	case expectedErrorGotResult:
		return newExpectedErrorGotResultErr(stateDB, expected, data)
	case unexpectedDataType:
		return newUnexpectedDataTypeErr(data)
	case cannotUnmarshalResult:
		return newCannotUnmarshalResult(data)
	case internalError:
		return newInternalError(data)
	case cannotSendRPCRequest:
		return newCannotSendRPCRequestErr(data)
	default:
		return &comparatorError{
			error: fmt.Errorf("default error:\n%v", data),
			typ:   0,
		}
	}
}

func newCannotSendRPCRequestErr(data comparisonData) *comparatorError {
	return &comparatorError{
		error: fmt.Errorf("could not resend request to rpc:"+
			"\nMethod: %v"+
			"\nBlockID: 0x%v"+
			"\n\tStateDB result: %v"+
			"\n\tStateDB err: %v"+
			"\n\tExpected result: %v"+
			"\n\tExpected err: %v"+
			"\n\nParams: %v", data.record.Query.Method, strconv.FormatUint(data.block, 16), data.StateDB.Result, data.StateDB.Error, data.record.Response.Result, data.record.Error.Error, string(data.record.ParamsRaw)),
		typ: cannotSendRPCRequest,
	}
}

// newNoMatchingResultErr returns new comparatorError
// It is returned when recording has an internal error code - this error is logged to level DEBUG and
// is not related to StateDB
func newInternalError(data comparisonData) *comparatorError {
	var stateDbRes string
	if data.StateDB.Result != nil {
		stateDbRes = fmt.Sprintf("%v", data.StateDB.Result)
	} else {
		stateDbRes = fmt.Sprintf("%v", data.StateDB.Error)
	}

	var recordedRes string
	if data.record.Response != nil {
		err := json.Unmarshal(data.record.Response.Result, &recordedRes)
		if err != nil {
			return newComparatorError(data.record.Response.Result, string(data.record.Response.Result), data, cannotUnmarshalResult)
		}
	} else {
		recordedRes = fmt.Sprintf("err: %v", data.record.Error.Error)
	}

	return &comparatorError{
		error: fmt.Errorf("recording with internal error for request:"+
			"\nMethod: %v"+
			"\nBlockID: 0x%v"+
			"\n\tStateDB: %v"+
			"\n\tExpected: %v"+
			"\n\nParams: %v", data.record.Query.Method, strconv.FormatUint(data.block, 16), stateDbRes, recordedRes, string(data.record.ParamsRaw)),
		typ: internalError,
	}
}

// newCannotUnmarshalResult returns new comparatorError
// It is returned when json.Unmarshal returned an error
func newCannotUnmarshalResult(data comparisonData) *comparatorError {
	return &comparatorError{
		error: fmt.Errorf("cannot unmarshal result, returning every possible data"+
			"\nMethod: %v"+
			"\nBlockID: 0x%v"+
			"\n\tStateDB result: %v"+
			"\n\tStateDB err: %v"+
			"\n\tRecorded result: %v"+
			"\n\tRecorded err: %v"+
			"\n\nParams: %v", data.record.Query.Method, strconv.FormatUint(data.block, 16), data.StateDB.Result, data.StateDB.Error, data.record.Response.Result, data.record.Error.Error, string(data.record.ParamsRaw)),
		typ: cannotUnmarshalResult,
	}
}

// newNoMatchingResultErr returns new comparatorError
// It is returned when StateDB result does not match with recorded result
func newNoMatchingResultErr(stateDBData, expectedData any, data comparisonData) *comparatorError {
	return &comparatorError{
		error: fmt.Errorf("result do not match"+
			"\nMethod: %v"+
			"\nBlockID: 0x%v"+
			"\n\tCarmen: %v"+
			"\n\tRecorded: %v"+
			"\n\n\tParams: %v", data.record.Query.Method, strconv.FormatUint(data.block, 16), stateDBData, expectedData, string(data.record.ParamsRaw)),
		typ: noMatchingResult,
	}
}

// newNoMatchingErrorsErr returns new comparatorError
// It is returned when StateDB error does not match with recorded error
func newNoMatchingErrorsErr(stateDBError, expectedError any, data comparisonData) *comparatorError {
	return &comparatorError{
		error: fmt.Errorf("errors do not match"+
			"\nMethod: %v"+
			"\nBlockID: 0x%v"+
			"\n\tCarmen: %v"+
			"\n\tRecorded: %v"+
			"\n\nParams: %v", data.record.Query.Method, strconv.FormatUint(data.block, 16), stateDBError, expectedError, string(data.record.ParamsRaw)),
		typ: noMatchingErrors,
	}
}

// newExpectedResultGotErrorErr returns new comparatorError
// It is returned when StateDB returns an error but expected return is a valid result
func newExpectedResultGotErrorErr(stateDBError, expectedResult any, data comparisonData) *comparatorError {
	return &comparatorError{
		error: fmt.Errorf("expected valid result but StateDB returned err"+
			"\nMethod: %v"+
			"\nBlockID: 0x%v"+
			"\n\tCarmen: %v"+
			"\n\tRecorded: %v"+
			"\n\nParams: %v", data.record.Query.Method, strconv.FormatUint(data.block, 16), stateDBError, expectedResult, string(data.record.ParamsRaw)),
		typ: expectedResultGotError,
	}
}

// newExpectedErrorGotResultErr returns new comparatorError
// It is returned when StateDB returns a valid error but expected return is an error
func newExpectedErrorGotResultErr(stateDBResult, expectedError any, data comparisonData) *comparatorError {
	return &comparatorError{
		error: fmt.Errorf("expected error but StateDB returned valid result"+
			"\nMethod: %v"+
			"\nBlockID: 0x%v"+
			"\n\tCarmen: %v"+
			"\n\tRecorded: %v"+
			"\n\nParams: %v", data.record.Query.Method, strconv.FormatUint(data.block, 16), stateDBResult, expectedError, string(data.record.ParamsRaw)),
		typ: expectedErrorGotResult,
	}
}

// newUnexpectedDataTypeErr returns comparatorError
// It is returned when Comparator is given unexpected data type in result from StateDB
func newUnexpectedDataTypeErr(data comparisonData) *comparatorError {
	return &comparatorError{
		error: fmt.Errorf("unexpected data type:\n%v", data),
		typ:   unexpectedDataType,
	}
}
