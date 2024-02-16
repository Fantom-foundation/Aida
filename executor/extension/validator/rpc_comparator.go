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

type comparatorErrorStatus byte

const (
	statusOk comparatorErrorStatus = iota
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
	typ comparatorErrorStatus
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
		// some records are recorded wrongly, for this case we resend the request and compare it again
		if !state.Data.IsRecovered {
			c.log.Debugf("retrying %v request", state.Data.Query.Method)
			c.numberOfRetriedRequests++
			c.log.Debugf("current ration retried against total %v/%v", c.numberOfRetriedRequests, c.totalNumberOfRequests)
			state.Data.IsRecovered = true

			if err := resendRequest(ctx.ExecutionResult, state); err != nil {
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

func compare(result txcontext.Receipt, state executor.State[*rpc.RequestAndResults]) *comparatorError {
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

func resendRequest(result txcontext.Receipt, state executor.State[*rpc.RequestAndResults]) *comparatorError {
	var payload []byte

	if state.Data.Response != nil {
		payload = state.Data.Response.Payload
	} else {
		payload = state.Data.Error.Payload
	}

	retriedReq := utils.JsonRPCRequest{
		Method:  state.Data.Query.Method,
		Params:  state.Data.Query.Params,
		ID:      0,
		JSONRPC: "2.0",
	}

	// append correct block number
	retriedReq.Params[len(retriedReq.Params)-1] = hexutil.EncodeUint64(uint64(state.Data.RequestedBlock))

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
		res, err := json.Marshal(s)
		if err != nil {
			return newComparatorError(result, nil, nil, state.Data, state.Block, cannotUnmarshalResult)
		}

		state.Data.Response = &rpc.Response{
			Version:   "2.0",
			ID:        json.RawMessage{1},
			BlockID:   uint64(state.Data.RequestedBlock),
			Timestamp: state.Data.RecordedTimestamp,
			Result:    res,
			Payload:   payload,
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
			Version:   "2.0",
			Id:        json.RawMessage{1},
			BlockID:   uint64(state.Data.RequestedBlock),
			Timestamp: state.Data.RecordedTimestamp,
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
func compareBalance(result txcontext.Receipt, data *rpc.RequestAndResults, block int) *comparatorError {
	res := result.GetResult()
	stateBalance := new(big.Int).SetBytes(res.Message)

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
func compareTransactionCount(result txcontext.Receipt, data *rpc.RequestAndResults, block int) *comparatorError {
	res := result.GetResult()
	stateNonce := littleendian.BytesToUint64(res.Message)

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
func compareCall(result txcontext.Receipt, data *rpc.RequestAndResults, block int) *comparatorError {
	res := result.GetResult()
	if res.Message != nil {
		return compareCallStateDbResult(result, res.Message, data, block)
	}

	if res.Err != nil {
		return compareEVMStateDBError(result, res.Err, data, block)
	}

	return newUnexpectedDataTypeErr(data)
}

// compareCallStateDbResult compares valid call result recorded on API server with valid result returned by StateDb
func compareCallStateDbResult(result txcontext.Receipt, res []byte, data *rpc.RequestAndResults, block int) *comparatorError {
	dbString := hexutil.Encode(res)

	if data.Error == nil {
		var recordedString string
		err := json.Unmarshal(data.Response.Result, &recordedString)
		if err != nil {
			return newComparatorError(result, dbString, string(data.Response.Result), data, block, cannotUnmarshalResult)
		}

		// results do not match
		if !strings.EqualFold(recordedString, dbString) {
			return newComparatorError(
				result,
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

	return newComparatorError(
		result,
		dbString,
		msg,
		data,
		block,
		expectedErrorGotResult)
}

// compareEVMStateDBError compares error returned from EvmExecutor with recorded data
func compareEVMStateDBError(result txcontext.Receipt, err error, data *rpc.RequestAndResults, block int) *comparatorError {
	if data.Error == nil {
		return newComparatorError(
			result,
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

	return newComparatorError(
		result,
		err,
		msg,
		data,
		block,
		noMatchingErrors)
}

// compareEstimateGas compares recorded data for estimateGas method with result from StateDB
func compareEstimateGas(result txcontext.Receipt, data *rpc.RequestAndResults, block int) *comparatorError {
	res := result.GetResult()
	if res.Message != nil {
		return compareEstimateGasStateDBResult(result, res.Message, data, block)
	}

	if res.Err != nil {
		return compareEVMStateDBError(result, res.Err, data, block)
	}

	return nil
}

// compareEstimateGasStateDBResult compares estimateGas data recorded on API server with data returned by StateDB
func compareEstimateGasStateDBResult(result txcontext.Receipt, res []byte, data *rpc.RequestAndResults, block int) *comparatorError {
	stateDBGas := littleendian.BytesToUint64(res)

	if data.Error != nil {
		if data.Error.Error.Code == internalErrorCode {
			return newComparatorError(nil, stateDBGas, data.Error.Error, data, block, internalError)
		}

		return newComparatorError(
			result,
			stateDBGas,
			EvmErrors[data.Error.Error.Code],
			data,
			block,
			expectedErrorGotResult)
	}

	var recordedString string

	err := json.Unmarshal(data.Response.Result, &recordedString)
	if err != nil {
		return newComparatorError(nil, stateDBGas, string(data.Response.Result), data, block, cannotUnmarshalResult)
	}

	recordedResult, err := hexutil.DecodeUint64(recordedString)
	if err != nil {
		return newComparatorError(nil, recordedResult, string(data.Response.Result), data, block, cannotUnmarshalResult)
	}

	if stateDBGas != recordedResult {
		return newComparatorError(nil, recordedResult, recordedString, data, block, noMatchingResult)
	}

	return nil
}

// compareCode compares getCode data recorded on API server with data returned by StateDB
func compareCode(result txcontext.Receipt, data *rpc.RequestAndResults, block int) *comparatorError {
	res := result.GetResult()
	dbString := hexutil.Encode(res.Message)

	if data.Error != nil {
		if data.Error.Error.Code == internalErrorCode {
			return newComparatorError(result, dbString, data.Error.Error, data, block, internalError)
		}
		return newComparatorError(result, dbString, data.Error.Error, data, block, expectedErrorGotResult)
	}

	var recordedString string

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
func compareStorageAt(result txcontext.Receipt, data *rpc.RequestAndResults, block int) *comparatorError {
	res := result.GetResult()
	dbString := hexutil.Encode(res.Message)

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
func newComparatorError(result txcontext.Receipt, stateDB, expected any, data *rpc.RequestAndResults, block int, typ comparatorErrorStatus) *comparatorError {
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

func newCannotSendRPCRequestErr(result txcontext.Receipt, data *rpc.RequestAndResults, block int) *comparatorError {
	res := result.GetResult()
	return &comparatorError{
		error: fmt.Errorf("could not resend request to rpc:"+
			"\nMethod: %v"+
			"\nBlockID: 0x%v"+
			"\n\tStateDB result: %v"+
			"\n\tStateDB err: %v"+
			"\n\tExpected result: %v"+
			"\n\tExpected err: %v"+
			"\n\nParams: %v", data.Query.Method, strconv.FormatInt(int64(block), 16), hexutil.Encode(res.Message), res.Err, data.Response, data.Error, string(data.ParamsRaw)),
		typ: cannotSendRpcRequest,
	}
}

// newCannotUnmarshalResult returns new comparatorError
// It is returned when json.Unmarshal returned an error
func newCannotUnmarshalResult(result txcontext.Receipt, data *rpc.RequestAndResults, block int) *comparatorError {
	res := result.GetResult()
	return &comparatorError{
		error: fmt.Errorf("cannot unmarshal result, returning every possible data"+
			"\nMethod: %v"+
			"\nBlockID: 0x%v"+
			"\n\tStateDB result: %v"+
			"\n\tStateDB err: %v"+
			"\n\tRecorded result: %v"+
			"\n\tRecorded err: %v"+
			"\n\nParams: %v", data.Query.Method, strconv.FormatInt(int64(block), 16), hexutil.Encode(res.Message), res.Err, data.Response, data.Error, string(data.ParamsRaw)),
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
