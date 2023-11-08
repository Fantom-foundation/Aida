package validator

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strconv"
	"strings"

	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/executor/extension"
	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/rpc"
	"github.com/Fantom-foundation/Aida/utils"
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
	return &rpcComparator{cfg: cfg, log: log, errors: make([]error, 0)}
}

type rpcComparator struct {
	extension.NilExtension[*rpc.RequestAndResults]
	cfg                     *utils.Config
	log                     logger.Logger
	errors                  []error
	numberOfRetriedRequests int
	totalNumberOfRequests   int
}

// PostTransaction compares result with recording. If ContinueOnFailure
// is enabled error is saved. Otherwise, the error is returned.
func (c *rpcComparator) PostTransaction(state executor.State[*rpc.RequestAndResults], _ *executor.Context) error {
	defer func() {
		rec := recover()
		if rec != nil {
			fmt.Println("comparator")
			fmt.Println(rec)
		}
	}()
	// StateDB can be nil if invalid block number is passed
	if state.Data.StateDB == nil {
		return nil
	}

	c.totalNumberOfRequests++

	compareErr := compare(state)
	if compareErr != nil {
		// lot errors are recorded wrongly, for this case we resend the request and compare it again
		if !state.Data.StateDB.IsRecovered && state.Data.Error != nil {
			c.log.Debugf("retrying %v request", state.Data.Query.Method)
			c.log.Debugf("current ration retried against total %v/%v", c.numberOfRetriedRequests, c.totalNumberOfRequests)
			c.numberOfRetriedRequests++
			state.Data.StateDB.IsRecovered = true
			compareErr = retryRequest(state)
			if compareErr == nil {
				return nil
			}
		}

		if compareErr.typ == cannotUnmarshalResult {
			return nil
		}

		if c.cfg.ContinueOnFailure {
			c.log.Warning(compareErr)
			c.errors = append(c.errors, compareErr)
			return nil
		}

		return compareErr
	}

	return nil
}

// PostRun prints all caught errors.
func (c *rpcComparator) PostRun(executor.State[*rpc.RequestAndResults], *executor.Context, error) error {
	// log only if continue on failure is enabled
	if !c.cfg.ContinueOnFailure {
		return nil
	}

	switch len(c.errors) {
	case 0:
		c.log.Notice("No errors found!")
		return nil
	case 1:
		c.log.Error("1 error was found:\n%v", c.errors[0].Error())
	default:
		c.log.Errorf("%v errors were found:\n%v", len(c.errors), errors.Join(c.errors...))
	}

	return nil
}

func compare(state executor.State[*rpc.RequestAndResults]) *comparatorError {
	switch state.Data.Query.MethodBase {
	case "getBalance":
		return compareBalance(state.Data, state.Block)
	case "getTransactionCount":
		return compareTransactionCount(state.Data, state.Block)
	case "call":
		return compareCall(state.Data, state.Block)
	case "estimateGas":
		// estimateGas is currently not suitable for replay since the estimation  in geth is always calculated
		// for current state that means recorded result and result returned by StateDB are not comparable
	case "getCode":
		return compareCode(state.Data, state.Block)
	case "getStorageAt":
		return compareStorageAt(state.Data, state.Block)
	}

	return nil
}

func retryRequest(state executor.State[*rpc.RequestAndResults]) *comparatorError {
	payload := utils.JsonRPCRequest{
		Method:  state.Data.Query.Method,
		Params:  state.Data.Query.Params,
		ID:      0,
		JSONRPC: "2.0",
	}

	// append correct block number
	payload.Params[len(payload.Params)-1] = "0x" + strconv.FormatInt(int64(state.Block), 16)

	// we only state on mainnet, so we can safely put mainnet chainID constant here
	m, err := utils.SendRpcRequest(payload, utils.MainnetChainID)
	if err != nil {
		return newComparatorError(nil, nil, state.Data, state.Block, cannotSendRpcRequest)
	}

	s, ok := m["result"].(string)
	if !ok {
		return newComparatorError(nil, nil, state.Data, state.Block, cannotUnmarshalResult)
	}

	result, err := json.Marshal(s)
	if err != nil {
		return newComparatorError(nil, nil, state.Data, state.Block, cannotUnmarshalResult)
	}

	state.Data.Response = &rpc.Response{
		Version:   state.Data.Error.Version,
		ID:        state.Data.Error.Id,
		BlockID:   state.Data.Error.BlockID,
		Timestamp: state.Data.Error.Timestamp,
		Result:    result,
		Payload:   state.Data.Error.Payload,
	}

	state.Data.Error = nil

	e := compare(state)
	if err != nil {
		return e
	}

	return nil
}

// compareBalance compares getBalance data recorded on API server with data returned by StateDB
func compareBalance(data *rpc.RequestAndResults, block int) *comparatorError {
	stateBalance, ok := data.StateDB.Result.(*big.Int)
	if !ok {
		return newUnexpectedDataTypeErr(data)
	}

	if data.Error != nil {
		if data.Error.Error.Code == internalErrorCode {
			return newComparatorError(stateBalance.Text(16), data.Error.Error.Message, data, block, internalError)
		}
		return newComparatorError(stateBalance.Text(16), data.Error.Error.Message, data, block, expectedErrorGotResult)
	}

	// no error
	var recordedString string
	err := json.Unmarshal(data.Response.Result, &recordedString)
	if err != nil {
		return newComparatorError(stateBalance.Text(16), string(data.Response.Result), data, block, cannotUnmarshalResult)
	}
	recordedString = strings.TrimPrefix(recordedString, "0x")

	recordedBalance, ok := new(big.Int).SetString(recordedString, 16)
	if !ok {
		return newComparatorError(stateBalance.Text(16), string(data.Response.Result), data, block, cannotUnmarshalResult)
	}

	if stateBalance.Cmp(recordedBalance) != 0 {
		return newComparatorError(stateBalance.Text(16), recordedBalance.Text(16), data, block, noMatchingResult)
	}

	return nil

}

// compareTransactionCount compares getTransactionCount data recorded on API server with data returned by StateDB
func compareTransactionCount(data *rpc.RequestAndResults, block int) *comparatorError {
	stateNonce, ok := data.StateDB.Result.(uint64)
	if !ok {
		return newUnexpectedDataTypeErr(data)
	}

	var err error

	if data.Error != nil {
		if data.Error.Error.Code == internalErrorCode {
			return newComparatorError(stateNonce, data.Error.Error.Message, data, block, internalError)
		}
		return newComparatorError(stateNonce, data.Error.Error.Message, data, block, expectedErrorGotResult)
	}

	var recordedString string
	// no error
	err = json.Unmarshal(data.Response.Result, &recordedString)
	if err != nil {
		return newComparatorError(stateNonce, string(data.Response.Result), data, block, cannotUnmarshalResult)
	}

	recordedNonce, err := hexutil.DecodeUint64(recordedString)
	if err != nil {
		return newComparatorError(recordedNonce, string(data.Response.Result), data, block, cannotUnmarshalResult)
	}

	if stateNonce != recordedNonce {
		return newComparatorError(recordedNonce, recordedString, data, block, noMatchingResult)
	}

	return nil
}

// compareCall compares call data recorded on API server with data returned by StateDB
func compareCall(data *rpc.RequestAndResults, block int) *comparatorError {
	// do we have an error from StateDB?
	if data.StateDB.Error != nil {
		return compareEVMStateDBError(data, block)
	}

	// did StateDB return a valid result?
	if data.StateDB.Result != nil {
		return compareCallStateDbResult(data, block)
	}

	return newUnexpectedDataTypeErr(data)
}

// compareCallStateDbResult compares valid call result recorded on API server with valid result returned by StateDb
func compareCallStateDbResult(data *rpc.RequestAndResults, block int) *comparatorError {
	dbString := hexutil.Encode(data.StateDB.Result.([]byte))

	if data.Error == nil {
		var recordedString string
		err := json.Unmarshal(data.Response.Result, &recordedString)
		if err != nil {
			return newComparatorError(dbString, string(data.Response.Result), data, block, cannotUnmarshalResult)
		}

		// results do not match
		if !strings.EqualFold(recordedString, dbString) {
			return newComparatorError(
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
		return newComparatorError(dbString, data.Error.Error, data, block, internalError)
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

	return newComparatorError(
		dbString,
		msg,
		data,
		block,
		expectedErrorGotResult)
}

// compareEVMStateDBError compares error returned from EvmExecutor with recorded data
func compareEVMStateDBError(data *rpc.RequestAndResults, block int) *comparatorError {
	if data.Error == nil {
		return newComparatorError(
			data.StateDB.Error,
			data.Response.Result,
			data,
			block,
			expectedResultGotError)
	}

	for _, e := range EvmErrors[data.Error.Error.Code] {
		if strings.Contains(data.StateDB.Error.Error(), e) {
			return nil
		}
	}

	if data.Error.Error.Code == internalErrorCode {
		return newComparatorError(data.StateDB.Error, data.Error.Error, data, block, internalError)
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

	return newComparatorError(
		data.StateDB.Error,
		msg,
		data,
		block,
		noMatchingErrors)
}

// compareEstimateGas compares recorded data for estimateGas method with result from StateDB
func compareEstimateGas(data *rpc.RequestAndResults, block int) *comparatorError {

	// StateDB returned an error
	if data.StateDB.Error != nil {
		return compareEVMStateDBError(data, block)
	}

	// StateDB returned a result
	if data.StateDB.Result != nil {
		return compareEstimateGasStateDBResult(data, block)
	}

	return nil
}

// compareEstimateGasStateDBResult compares estimateGas data recorded on API server with data returned by StateDB
func compareEstimateGasStateDBResult(data *rpc.RequestAndResults, block int) *comparatorError {
	stateDBGas, ok := data.StateDB.Result.(hexutil.Uint64)
	if !ok {
		return newUnexpectedDataTypeErr(data)
	}

	// did we receive an error
	if data.Error != nil {
		// internal error?
		if data.Error.Error.Code == internalErrorCode {
			return newComparatorError(stateDBGas, data.Error.Error, data, block, internalError)
		}

		return newComparatorError(
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
		return newComparatorError(stateDBGas, string(data.Response.Result), data, block, cannotUnmarshalResult)
	}

	recordedResult, err := hexutil.DecodeUint64(recordedString)
	if err != nil {
		return newComparatorError(recordedResult, string(data.Response.Result), data, block, cannotUnmarshalResult)
	}

	if uint64(stateDBGas) != recordedResult {
		return newComparatorError(recordedResult, recordedString, data, block, noMatchingResult)
	}

	return nil
}

// compareCode compares getCode data recorded on API server with data returned by StateDB
func compareCode(data *rpc.RequestAndResults, block int) *comparatorError {
	dbString := hexutil.Encode(data.StateDB.Result.([]byte))

	// did we data an error?
	if data.Error != nil {
		// internal error?
		if data.Error.Error.Code == internalErrorCode {
			return newComparatorError(dbString, data.Error.Error, data, block, internalError)
		}
		return newComparatorError(dbString, data.Error.Error, data, block, expectedErrorGotResult)
	}

	var recordedString string

	// no error
	err := json.Unmarshal(data.Response.Result, &recordedString)
	if err != nil {
		return newComparatorError(dbString, string(data.Response.Result), data, block, cannotUnmarshalResult)
	}

	if !strings.EqualFold(recordedString, dbString) {
		return newComparatorError(dbString, recordedString, data, block, noMatchingResult)
	}

	return nil
}

// compareStorageAt compares getStorageAt data recorded on API server with data returned by StateDB
func compareStorageAt(data *rpc.RequestAndResults, block int) *comparatorError {
	dbString := hexutil.Encode(data.StateDB.Result.([]byte))

	if data.Error != nil {
		// internal error?
		if data.Error.Error.Code == internalErrorCode {
			return newComparatorError(dbString, data.Error, data, block, internalError)
		}
		return newComparatorError(dbString, data.Error, data, block, internalError)
	}

	var recordedString string

	// no error
	err := json.Unmarshal(data.Response.Result, &recordedString)
	if err != nil {
		return newComparatorError(dbString, string(data.Response.Result), data, block, cannotUnmarshalResult)
	}

	if !strings.EqualFold(recordedString, dbString) {
		return newComparatorError(dbString, recordedString, data, block, noMatchingResult)
	}

	return nil
}

// newComparatorError returns new comparatorError with given StateDB and recorded data based on the typ.
func newComparatorError(stateDB, expected any, data *rpc.RequestAndResults, block int, typ comparatorErrorType) *comparatorError {
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
		return newCannotUnmarshalResult(data, block)
	case internalError:
		// internalError is caused by opera, adding this to the error list does not make sense
		return nil
	case cannotSendRpcRequest:
		return newCannotSendRPCRequestErr(data, block)
	default:
		return &comparatorError{
			error: fmt.Errorf("default error:\n%v", data),
			typ:   0,
		}
	}
}

func newCannotSendRPCRequestErr(data *rpc.RequestAndResults, block int) *comparatorError {
	return &comparatorError{
		error: fmt.Errorf("could not resend request to rpc:"+
			"\nMethod: %v"+
			"\nBlockID: 0x%v"+
			"\n\tStateDB result: %v"+
			"\n\tStateDB err: %v"+
			"\n\tExpected result: %v"+
			"\n\tExpected err: %v"+
			"\n\nParams: %v", data.Query.Method, strconv.FormatInt(int64(block), 16), data.StateDB.Result, data.StateDB.Error, data.Response, data.Error, string(data.ParamsRaw)),
		typ: cannotSendRpcRequest,
	}
}

// newNoMatchingResultErr returns new comparatorError
// It is returned when recording has an internal error code - this error is logged to level DEBUG and
// is not related to StateDB
func newInternalError(data *rpc.RequestAndResults, block int) *comparatorError {
	var stateDbRes string
	if data.StateDB.Result != nil {
		stateDbRes = fmt.Sprintf("%v", data.StateDB.Result)
	} else {
		stateDbRes = fmt.Sprintf("%v", data.StateDB.Error)
	}

	var recordedRes string
	if data.Response != nil {
		err := json.Unmarshal(data.Response.Result, &recordedRes)
		if err != nil {
			return newComparatorError(data.Response.Result, string(data.Response.Result), data, block, cannotUnmarshalResult)
		}
	} else {
		recordedRes = fmt.Sprintf("err: %v", data.Error.Error)
	}

	return &comparatorError{
		error: fmt.Errorf("recording with internal error for request:"+
			"\nMethod: %v"+
			"\nBlockID: 0x%v"+
			"\n\tStateDB: %v"+
			"\n\tExpected: %v"+
			"\n\nParams: %v", data.Query.Method, strconv.FormatInt(int64(block), 16), stateDbRes, recordedRes, string(data.ParamsRaw)),
		typ: internalError,
	}
}

// newCannotUnmarshalResult returns new comparatorError
// It is returned when json.Unmarshal returned an error
func newCannotUnmarshalResult(data *rpc.RequestAndResults, block int) *comparatorError {
	return &comparatorError{
		error: fmt.Errorf("cannot unmarshal result, returning every possible data"+
			"\nMethod: %v"+
			"\nBlockID: 0x%v"+
			"\n\tStateDB result: %v"+
			"\n\tStateDB err: %v"+
			"\n\tRecorded result: %v"+
			"\n\tRecorded err: %v"+
			"\n\nParams: %v", data.Query.Method, strconv.FormatInt(int64(block), 16), data.StateDB.Result, data.StateDB.Error, data.Response, data.Error, string(data.ParamsRaw)),
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
	return &comparatorError{
		error: fmt.Errorf("unexpected data type:\n%v", data),
		typ:   unexpectedDataType,
	}
}
