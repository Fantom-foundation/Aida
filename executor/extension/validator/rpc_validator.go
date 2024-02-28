package validator

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"reflect"
	"strconv"
	"strings"

	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/executor/extension"
	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/rpc"
	"github.com/Fantom-foundation/Aida/utils"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

type rpcValidationStatusCode byte

const (
	rpcValidationOk rpcValidationStatusCode = iota
	noMatchingResult
	noMatchingErrors
	expectedErrorGotResult
	expectedResultGotError
	unexpectedDataType
	cannotUnmarshalResult
	internalRpcError
	unsupportedMethod
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

type rpcValidationResult struct {
	code rpcValidationStatusCode
	//validationErr *rpcValidationError
	err error
}

type rpcValidationError struct {
	mismatch string
}

func (m rpcValidationError) Error() string {
	return m.mismatch
}

// MakeRpcComparator returns extension which handles comparison of result created by the StateDb and the recording.
// If ContinueOnFailure is enabled errors are being saved and printed after the whole run ends. Otherwise, error is returned.
func MakeRpcComparator(cfg *utils.Config) executor.Extension[*rpc.RequestAndResults] {
	if !cfg.Validate {
		return extension.NilExtension[*rpc.RequestAndResults]{}
	}

	log := logger.NewLogger("INFO", "rpc-validator")

	return makeRPCComparator(cfg, log)
}

func makeRPCComparator(cfg *utils.Config, log logger.Logger) *rpcValidator {
	return &rpcValidator{cfg: cfg, log: log}
}

type rpcValidator struct {
	extension.NilExtension[*rpc.RequestAndResults]
	cfg                     *utils.Config
	log                     logger.Logger
	numberOfRetriedRequests int
	totalNumberOfRequests   int
	numberOfErrors          int
}

// PostTransaction compares result with recording. If ContinueOnFailure
// is enabled error is saved. Otherwise, the error is returned.
func (v *rpcValidator) PostTransaction(state executor.State[*rpc.RequestAndResults], ctx *executor.Context) error {
	v.totalNumberOfRequests++

	// pending block numbers are not validatable
	if state.Data.SkipValidation || state.Data.StateDB == nil {
		return nil
	}

	result := v.compare(state)
	if result.err != nil {
		return v.handleError(state, ctx, result)
	}

	return nil
}

func (v *rpcValidator) compare(state executor.State[*rpc.RequestAndResults]) rpcValidationResult {
	switch state.Data.Query.MethodBase {
	case "getBalance":
		return compareBalance(state.Data)
	case "getTransactionCount":
		return compareTransactionCount(state.Data)
	case "call":
		return compareCall(state.Data)
	case "estimateGas":
		// estimateGas is currently not suitable for replay since the estimation  in geth is always calculated
		// for current state that means recorded result and result returned by StateDB are not comparable
	case "getCode":
		return compareCode(state.Data)
	case "getStorageAt":
		return compareStorageAt(state.Data)
	default:
		break
	}

	return unsupportedMethodErr(state.Data.Query.Method)
}

func (v *rpcValidator) retryRequest(state executor.State[*rpc.RequestAndResults]) rpcValidationResult {
	var payload []byte

	if state.Data.Response != nil {
		payload = state.Data.Response.Payload
		v.log.Debugf("Previously recorded: %v", state.Data.Response.Result)
	} else {
		payload = state.Data.Error.Payload
		v.log.Debugf("Previously recorded: %v", state.Data.Error.Error)
	}

	retriedReq := utils.JsonRPCRequest{
		Method:  state.Data.Query.Method,
		Params:  state.Data.Query.Params,
		ID:      0,
		JSONRPC: "2.0",
	}

	v.log.Debugf("Previous params: %v", state.Data.Query.Params)

	b := hexutil.EncodeUint64(uint64(state.Data.RequestedBlock))
	l := len(retriedReq.Params)
	// this is a corner case where request does not have block number
	if l <= 1 {
		retriedReq.Params = append(retriedReq.Params, b)
	} else {
		retriedReq.Params[len(retriedReq.Params)-1] = b
	}

	v.log.Debugf("Retried params: %v", retriedReq.Params)
	// we only record on mainnet, so we can safely put mainnet chainID constant here
	m, err := utils.SendRpcRequest(retriedReq, utils.MainnetChainID)
	if err != nil {
		// very unlikely to happen
		// ignore ths error
		return validationOk()
	}

	// remove the data
	state.Data.Response = nil
	state.Data.Error = nil

	s, ok := m["result"].(string)
	if ok { // valid result
		result, err := json.Marshal(s)
		if err != nil {
			// very unlikely to happen
			// ignore ths error
			return validationOk()
		}

		state.Data.Response = &rpc.Response{
			Version: "2.0",
			ID:      json.RawMessage{1},
			BlockID: uint64(state.Data.RequestedBlock),
			Result:  result,
			Payload: payload,
		}
	} else { // error result
		resMap, ok := m["error"].(map[string]interface{})
		if !ok {
			// very unlikely to happen
			// ignore ths error
			return validationOk()
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

	return v.compare(state)
}

// handleError decides whether error is fatal. There are some corner-cases that require special treatment.
// Any handler corner-case is explained in the implementation.
func (v *rpcValidator) handleError(state executor.State[*rpc.RequestAndResults], ctx *executor.Context, result rpcValidationResult) error {
	// request method base 'call' cannot be resent, because we need timestamp of the block that executed
	// this request. As of right now there we cannot get the timestamp, hence we skip these requests
	if state.Data.Query.MethodBase == "call" {
		// Only requests containing an error result are not being treated as data
		// mismatch, requests with a non-error result are recorded correctly
		if state.Data.Error != nil {
			return nil
		}
		return result.err
	}

	// all other requests are retried by resending them back to rpc and then comparing them again
	if !state.Data.StateDB.IsRecovered {
		v.log.Debugf("retrying %v request", state.Data.Query.Method)
		v.numberOfRetriedRequests++
		v.log.Debugf("current ration retried against total %v/%v", v.numberOfRetriedRequests, v.totalNumberOfRequests)
		state.Data.StateDB.IsRecovered = true
		result = v.retryRequest(state)
		if result.code == rpcValidationOk {
			return nil
		}
		return v.PostTransaction(state, ctx)
	}

	//if result.code == cannotUnmarshalResult { todo ?
	//	return nil
	//}

	if !v.cfg.ContinueOnFailure {
		return result.err
	}

	ctx.ErrorInput <- result.err
	v.numberOfErrors++

	// endless run
	if v.cfg.MaxNumErrors == 0 {
		return nil
	}

	if v.numberOfErrors >= v.cfg.MaxNumErrors {
		return result.err
	}

	return nil
}

// compareBalance compares getBalance data recorded on API server with data returned by StateDB
func compareBalance(data *rpc.RequestAndResults) rpcValidationResult {
	result := data.StateDB.Result
	stateBalance, ok := result.(*big.Int)
	if !ok { // this is very unlikely to happen
		return unexpectedDataTypeErr(data.ParamsToString(), reflect.TypeOf(result).String(), "big.Int")
	}

	if data.Error != nil {
		if data.Error.Error.Code == internalErrorCode {
			return internalRpcErrorResult()
		}
		want := findErrors(data)
		if want == "" {
			return validationOk()
		}

		return gotResultWantErrorErr(stateBalance.Text(16), want, data)
	}

	// no error
	var recordedString string
	err := json.Unmarshal(data.Response.Result, &recordedString)
	if err != nil {
		return newCannotUnmarshalResult(stateBalance.Text(16), string(data.Response.Result), data)
	}
	recordedString = strings.TrimPrefix(recordedString, "0x")

	recordedBalance := new(big.Int)
	recordedBalance.SetString(recordedString, 16)

	if stateBalance.Cmp(recordedBalance) != 0 {
		return noMatchingResultErr(recordedBalance.Text(16), stateBalance.Text(16), data)
	}

	return validationOk()
}

// compareTransactionCount compares getTransactionCount data recorded on API server with data returned by StateDB
func compareTransactionCount(data *rpc.RequestAndResults) rpcValidationResult {
	result := data.StateDB.Result
	stateNonce, ok := result.(uint64)
	if !ok {
		return unexpectedDataTypeErr(data.ParamsToString(), reflect.TypeOf(result).String(), "uint64")
	}

	var err error

	if data.Error != nil {
		if data.Error.Error.Code == internalErrorCode {
			return internalRpcErrorResult()
		}
		want := findErrors(data)
		if want == "" {
			return validationOk()
		}

		return gotResultWantErrorErr(strconv.FormatUint(stateNonce, 10), want, data)
	}

	var recordedString string
	// no error
	err = json.Unmarshal(data.Response.Result, &recordedString)
	if err != nil {
		return newCannotUnmarshalResult(strconv.FormatUint(stateNonce, 10), string(data.Response.Result), data)
	}

	recordedNonce, err := hexutil.DecodeUint64(recordedString)
	if err != nil {
		return newCannotUnmarshalResult(strconv.FormatUint(stateNonce, 10), string(data.Response.Result), data)
	}

	if stateNonce != recordedNonce {
		return noMatchingResultErr(strconv.FormatUint(stateNonce, 10), strconv.FormatUint(recordedNonce, 10), data)
	}

	return validationOk()
}

// compareCall compares call data recorded on API server with data returned by StateDB
func compareCall(data *rpc.RequestAndResults) rpcValidationResult {
	// do we have an error from StateDB?
	if data.StateDB.Error != nil {
		return compareEVMStateDBError(data)
	}

	return compareCallStateDbResult(data)
}

// compareCallStateDbResult compares valid call result recorded on API server with valid result returned by StateDb
func compareCallStateDbResult(data *rpc.RequestAndResults) rpcValidationResult {
	got := hexutil.Encode(data.StateDB.Result.([]byte))

	var want string
	if data.Error == nil {
		err := json.Unmarshal(data.Response.Result, &want)
		if err != nil {
			return newCannotUnmarshalResult(got, string(data.Response.Result), data)
		}

		// results do not match
		if !strings.EqualFold(want, got) {
			return noMatchingResultErr(got, want, data)
		}

		return validationOk()
	}

	// internal error?
	if data.Error.Error.Code == internalErrorCode {
		return internalRpcErrorResult()
	}

	want = findErrors(data)
	if want == "" {
		return validationOk()
	}

	return gotResultWantErrorErr(got, want, data)
}

// compareEVMStateDBError compares error returned from EvmExecutor with recorded data
func compareEVMStateDBError(data *rpc.RequestAndResults) rpcValidationResult {
	if data.Response != nil {
		want := findErrors(data)
		if want == "" {
			return validationOk()
		}
		return gotErrorWantResultErr(data.StateDB.Error.Error(), want, data)
	}

	stateDbErr := data.StateDB.Error.Error()

	for _, e := range EvmErrors[data.Error.Error.Code] {
		if strings.Contains(stateDbErr, e) {
			return validationOk()
		}
	}

	if data.Error.Error.Code == internalErrorCode {
		return internalRpcErrorResult()
	}

	builder := new(strings.Builder)

	builder.WriteString("one of these error messages: ")

	for i, e := range EvmErrors[data.Error.Error.Code] {
		builder.WriteString(e)
		if i < len(EvmErrors[data.Error.Error.Code]) {
			builder.WriteString(" or ")
		}
	}

	return noMatchingErrorsErr(stateDbErr, builder.String(), data)
}

// compareEstimateGas compares recorded data for estimateGas method with result from StateDB
func compareEstimateGas(data *rpc.RequestAndResults, block int) rpcValidationResult {

	// StateDB returned an error
	if data.StateDB.Error != nil {
		return compareEVMStateDBError(data)
	}

	// StateDB returned a result
	return compareEstimateGasStateDBResult(data)
}

// compareEstimateGasStateDBResult compares estimateGas data recorded on API server with data returned by StateDB
func compareEstimateGasStateDBResult(data *rpc.RequestAndResults) rpcValidationResult {
	got, ok := data.StateDB.Result.(hexutil.Uint64)
	if !ok {
		return unexpectedDataTypeErr(data.ParamsToString(), reflect.TypeOf(data.StateDB.Result).String(), "hexutil.Uint64")
	}

	// did we receive an error
	if data.Error != nil {
		// internal error?
		if data.Error.Error.Code == internalErrorCode {
			return internalRpcErrorResult()
		}

		want := findErrors(data)
		if want == "" {
			return validationOk()
		}

		return gotResultWantErrorErr(got.String(), want, data)
	}

	var str string
	// no error
	err := json.Unmarshal(data.Response.Result, &str)
	if err != nil {
		return newCannotUnmarshalResult(got.String(), string(data.Response.Result), data)
	}

	want, err := hexutil.DecodeUint64(str)
	if err != nil {
		return newCannotUnmarshalResult(got.String(), string(data.Response.Result), data)
	}

	if uint64(got) != want {
		return noMatchingResultErr(got.String(), strconv.FormatUint(want, 16), data)
	}

	return validationOk()
}

// compareCode compares getCode data recorded on API server with data returned by StateDB
func compareCode(data *rpc.RequestAndResults) rpcValidationResult {
	got := hexutil.Encode(data.StateDB.Result.([]byte))

	// did we data an error?
	if data.Error != nil {
		// internal error?
		if data.Error.Error.Code == internalErrorCode {
			return internalRpcErrorResult()
		}
		want := findErrors(data)
		if want == "" {
			return validationOk()
		}
		return gotResultWantErrorErr(got, want, data)
	}

	var want string

	// no error
	err := json.Unmarshal(data.Response.Result, &want)
	if err != nil {
		return newCannotUnmarshalResult(got, string(data.Response.Result), data)
	}

	if !strings.EqualFold(want, got) {
		return noMatchingResultErr(got, want, data)
	}

	return validationOk()
}

// compareStorageAt compares getStorageAt data recorded on API server with data returned by StateDB
func compareStorageAt(data *rpc.RequestAndResults) rpcValidationResult {
	got := hexutil.Encode(data.StateDB.Result.([]byte))

	if data.Error != nil {
		// internal error?
		if data.Error.Error.Code == internalErrorCode {
			return internalRpcErrorResult()
		}
		want := findErrors(data)
		if want == "" {
			return validationOk()
		}
		return gotResultWantErrorErr(got, want, data)
	}

	var want string

	// no error
	err := json.Unmarshal(data.Response.Result, &want)
	if err != nil {
		return newCannotUnmarshalResult(got, string(data.Response.Result), data)
	}

	if !strings.EqualFold(want, got) {
		return noMatchingResultErr(got, want, data)
	}

	return validationOk()
}

// newCannotUnmarshalResult returns new rpcValidationResult
// It is returned when json.Unmarshal returned an error
func newCannotUnmarshalResult(got, want string, data *rpc.RequestAndResults) rpcValidationResult {
	return rpcValidationResult{
		err: fmt.Errorf("cannot unmarshal result, \ngot: %v\n want: %v\n"+
			"\nMethod: %v"+
			"\nRequested block: %v"+
			"\nRecorded block: %v"+
			"\n\nParams: %v", got, want, data.Query.Method, data.RequestedBlock, data.RecordedBlock, data.ParamsToString()),
		code: cannotUnmarshalResult,
	}
}

// gotErrorWantResultErr returns new rpcValidationResult
// It is returned when StateDB returns an error but expected return is a valid result
func gotErrorWantResultErr(got, want string, data *rpc.RequestAndResults) rpcValidationResult {
	return rpcValidationResult{
		err: fmt.Errorf("got error: %v\nwant result: %v\n"+
			"\nMethod: %v"+
			"\nRecorded Block: %v"+
			"\nRequested Block: %v"+
			"\n\nParams: %v", got, want, data.Query.Method, data.RecordedBlock, data.RequestedBlock, data.ParamsToString()),
		code: expectedResultGotError,
	}
}

// gotResultWantErrorErr returns new rpcValidationResult
// It is returned when StateDB returns a valid error but expected return is an error
func gotResultWantErrorErr(got, want string, data *rpc.RequestAndResults) rpcValidationResult {
	return rpcValidationResult{
		err: fmt.Errorf("got result: %v\nwant error: %v\n"+
			"\nMethod: %v"+
			"\nRecorded Block: %v"+
			"\nRequested Block: %v"+
			"\n\nParams: %v", got, want, data.Query.Method, data.RecordedBlock, data.RequestedBlock, data.ParamsToString()),
		code: expectedErrorGotResult,
	}
}

func unexpectedDataTypeErr(params, got, want string) rpcValidationResult {
	return rpcValidationResult{
		code: unexpectedDataType,
		err:  fmt.Errorf("unexpected data type:\ngot: %v\nwant: %v\nparams:%v", got, want, params),
	}
}

func validationOk() rpcValidationResult {
	return rpcValidationResult{code: rpcValidationOk}
}

func internalRpcErrorResult() rpcValidationResult {
	return rpcValidationResult{
		code: internalRpcError,
		err:  errors.New("internal error"),
	}
}

// noMatchingResultErr returns new rpcValidationResult
// It is returned when StateDB result does not match with recorded result
func noMatchingResultErr(got, want string, data *rpc.RequestAndResults) rpcValidationResult {
	return rpcValidationResult{
		err: fmt.Errorf("results do not match\ngot: %v\nwant: %v\n"+
			"\nMethod: %v"+
			"\nRequested block: %v"+
			"\nRecorded block: %v"+
			"\n\n\tParams: %v", got, want, data.Query.Method, data.RequestedBlock, data.RecordedBlock, data.ParamsToString()),
		code: noMatchingResult,
	}
}

// noMatchingErrorsErr returns new rpcValidationResult
// It is returned when StateDB error does not match with recorded error
func noMatchingErrorsErr(got, want any, data *rpc.RequestAndResults) rpcValidationResult {
	return rpcValidationResult{
		err: fmt.Errorf("error does not contain wanted message\ngot: %v\nwant: %v"+
			"\nMethod: %v"+
			"\nRecorded block: %v"+
			"\nRequested block: %v"+
			"\n\nParams: %v", got, want, data.Query.Method, data.RecordedBlock, data.RequestedBlock, data.ParamsToString()),
		code: noMatchingErrors,
	}
}

func findErrors(data *rpc.RequestAndResults) string {
	var want string
	errs, ok := EvmErrors[data.Error.Error.Code]
	if !ok {
		want = fmt.Sprintf("unknown error code: %v", data.Error.Error.Code)
	} else {

		// we could have potentially recorded a request with invalid arguments
		// - this is not checked in execution, hence StateDB returns a valid result.
		// For this we exclude any invalid requests when getting unmatched results
		if data.Error.Error.Code == invalidArgumentErrCode {
			return ""
		}

		builder := new(strings.Builder)

		// more error messages for one code?
		for i, e := range errs {
			builder.WriteString(e)
			if len(errs) > i {
				builder.WriteString(" or ")
			}
		}
		want = builder.String()
	}

	return want
}

func unsupportedMethodErr(method string) rpcValidationResult {
	return rpcValidationResult{
		err:  fmt.Errorf("method %v is not (yet) supported", method),
		code: unsupportedMethod,
	}
}
