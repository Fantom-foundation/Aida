package apireplay

import (
	"errors"
	"fmt"

	"github.com/Fantom-foundation/go-opera/evmcore"
	"github.com/ethereum/go-ethereum/accounts/abi"
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
	internalError
)

// comparatorError is returned when data returned by StateDB does not match recorded data
type comparatorError struct {
	error
	typ comparatorErrorType
}

// revertError is returned by when transaction execution needs to be reverted by the EVM
type revertError struct {
	error
	reason string // revert reason hex encoded
}

// newRevertError creates new revertError based on given result
func newRevertError(result *evmcore.ExecutionResult) revertError {
	reason, errUnpack := abi.UnpackRevert(result.Revert())
	err := errors.New("execution reverted")
	if errUnpack == nil {
		err = fmt.Errorf("execution reverted: %v", reason)
	}
	return revertError{
		error:  err,
		reason: hexutil.Encode(result.Revert()),
	}
}

// newComparatorError returns new comparatorError with given StateDB and recorded data based on the typ.
func newComparatorError(stateDB, expected any, data *OutData, typ comparatorErrorType) *comparatorError {
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
	default:
		return &comparatorError{
			error: fmt.Errorf("default error:\n%v", data),
			typ:   0,
		}
	}
}

// newNoMatchingResultErr returns new comparatorError
// It is returned when recording has an internal error code - this error is logged to level DEBUG and
// is not related to StateDB
func newInternalError(data *OutData) *comparatorError {
	return &comparatorError{
		error: fmt.Errorf("recording with internal error for request:"+
			"\nMethod: %v"+
			"\nBlockID: %v"+
			"\n\tStateDB result: %v"+
			"\n\tStateDB err: %v"+
			"\n\tExpected result: %v"+
			"\n\tExpected err: %v"+
			"\n\tParams: %v", data.Method, data.BlockID, data.StateDB.Result, data.StateDB.Error, data.Recorded.Result, data.Recorded.Error, string(data.ParamsRaw)),
		typ: internalError,
	}
}

// newCannotUnmarshalResult returns new comparatorError
// It is returned when json.Unmarshal returned an error
func newCannotUnmarshalResult(data *OutData) *comparatorError {
	return &comparatorError{
		error: fmt.Errorf("cannot unmarshal result, returning every possible data"+
			"\nMethod: %v"+
			"\nBlockID: %v"+
			"\n\tStateDB result: %v"+
			"\n\tStateDB err: %v"+
			"\n\tExpected result: %v"+
			"\n\tExpected err: %v"+
			"\n\tParams: %v", data.Method, data.BlockID, data.StateDB.Result, data.StateDB.Error, data.Recorded.Result, data.Recorded.Error, string(data.ParamsRaw)),
		typ: cannotUnmarshalResult,
	}
}

// newNoMatchingResultErr returns new comparatorError
// It is returned when StateDB result does not match with recorded result
func newNoMatchingResultErr(stateDBData, expectedData any, data *OutData) *comparatorError {
	return &comparatorError{
		error: fmt.Errorf("result do not match"+
			"\nMethod: %v"+
			"\nBlockID: %v"+
			"\n\tStateDB: %v"+
			"\n\tExpected: %v"+
			"\n\tParams: %v", data.Method, data.BlockID, stateDBData, expectedData, string(data.ParamsRaw)),
		typ: noMatchingResult,
	}
}

// newNoMatchingErrorsErr returns new comparatorError
// It is returned when StateDB error does not match with recorded error
func newNoMatchingErrorsErr(stateDBError, expectedError any, data *OutData) *comparatorError {
	return &comparatorError{
		error: fmt.Errorf("errors do not match"+
			"\nMethod: %v"+
			"\nBlockID: %v"+
			"\n\tStateDB: %v"+
			"\n\tExpected: %v"+
			"\n\tParams: %v", data.Method, data.BlockID, stateDBError, expectedError, string(data.ParamsRaw)),
		typ: noMatchingErrors,
	}
}

// newExpectedResultGotErrorErr returns new comparatorError
// It is returned when StateDB returns an error but expected return is a valid result
func newExpectedResultGotErrorErr(stateDBError, expectedResult any, data *OutData) *comparatorError {
	return &comparatorError{
		error: fmt.Errorf("expected valid result but StateDB returned err"+
			"\nMethod: %v"+
			"\nBlockID: %v"+
			"\n\tStateDB: %v"+
			"\n\tExpected: %v"+
			"\n\tParams: %v", data.Method, data.BlockID, stateDBError, expectedResult, string(data.ParamsRaw)),
		typ: expectedResultGotError,
	}
}

// newExpectedErrorGotResultErr returns new comparatorError
// It is returned when StateDB returns a valid error but expected return is an error
func newExpectedErrorGotResultErr(stateDBResult, expectedError any, data *OutData) *comparatorError {
	return &comparatorError{
		error: fmt.Errorf("expected error but StateDB returned valid result"+
			"\nMethod: %v"+
			"\nBlockID: %v"+
			"\n\tStateDB: %v"+
			"\n\tExpected: %v"+
			"\n\tParams: %v", data.Method, data.BlockID, stateDBResult, expectedError, string(data.ParamsRaw)),
		typ: expectedErrorGotResult,
	}
}

// newUnexpectedDataTypeErr returns comparatorError
// It is returned when Comparator is given unexpected data type in result from StateDB
func newUnexpectedDataTypeErr(data *OutData) *comparatorError {
	return &comparatorError{
		error: fmt.Errorf("unexpected data type:\n%v", data),
		typ:   unexpectedDataType,
	}
}
