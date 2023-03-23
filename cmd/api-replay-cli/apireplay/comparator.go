package apireplay

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"sync"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/op/go-logging"
	"github.com/status-im/keycard-go/hexutils"
)

// nil answer from EVM is recorded as nilEVMResult, this is used this for the comparing and for more readable logMsg
const nilEVMResult = "0x0000000000000000000000000000000000000000000000000000000000000000"

// EVMErrors decode error code into string with which we compare recorded error message
var EVMErrors = map[int]string{
	-32000: "execution reverted",
	-32602: "invalid argument",
}

// Comparator compares data from StateDB and expected data recorded on API server
// This data is retrieved from Reader
type Comparator struct {
	input  chan *OutData
	log    *logging.Logger
	closed chan any
	wg     *sync.WaitGroup
}

// newComparator returns new instance of Comparator
func newComparator(input chan *OutData, log *logging.Logger, wg *sync.WaitGroup) *Comparator {
	return &Comparator{
		input:  input,
		log:    log,
		closed: make(chan any),
		wg:     wg,
	}
}

// Start the Comparator
func (c *Comparator) Start() {
	c.wg.Add(1)
	go c.compare()
}

// Stop the Comparator
func (c *Comparator) Stop() {
	select {
	case <-c.closed:
		return
	default:
		close(c.closed)
	}
}

// compare reads data from Reader and compares them. If doCompare func returns error,
// the error is logged since the results do not match
func (c *Comparator) compare() {
	var data *OutData
	var ok bool

	for {
		select {
		case <-c.closed:
			c.wg.Done()
			return
		default:
		}
		data, ok = <-c.input

		// stop Comparator if input is closed
		if !ok {
			c.Stop()
			continue
		}

		if err := c.doCompare(data); err != nil {
			c.log.Critical(err)
		}

	}
}

// doCompare calls adequate comparing function for given method
func (c *Comparator) doCompare(data *OutData) (err *comparatorError) {
	switch data.MethodBase {
	case "getBalance":
		err = compareBalance(data)
	case "getTransactionCount":
		err = compareTransactionCount(data)
	case "call":
		err = compareCall(data)
	case "estimateGas":
		err = compareEstimateGas(data)
	default:
		c.log.Debugf("unknown method in comparator: %v", data.Method)
	}

	return
}

// compareBalance compares getBalance data recorded on API server with data returned by StateDB
func compareBalance(data *OutData) *comparatorError {
	var (
		stateDBBalance, recordedBalance *big.Int
	)

	// did StateDB return a valid result?
	if v, ok := data.StateDB.Result.(*big.Int); ok {
		stateDBBalance = v

		// did we record an error?
		if data.Recorded.Error != nil {
			return newComparatorError(stateDBBalance, data.Recorded.Error.Message, data, expectedResultGotError)
		}

		recordedBalance = new(big.Int)

		// did we record a valid result?
		if data.Recorded.Result != nil {
			s := strings.TrimPrefix(string(data.Recorded.Result), "0x")
			recordedBalance.SetString(s, 16)
		}

		// matching results?
		if recordedBalance.Cmp(stateDBBalance) != 0 {
			return newComparatorError(stateDBBalance, recordedBalance, data, noMatchingResult)
		}

		return nil
	}

	return newUnexpectedDataTypeErr(data)

}

// compareTransactionCount compares getTransactionCount data recorded on API server with data returned by StateDB
func compareTransactionCount(data *OutData) *comparatorError {
	var (
		stateDBNonce, recordedNonce uint64
	)

	// did StateDB return a valid result?
	if v, ok := data.StateDB.Result.(uint64); ok {
		stateDBNonce = v

		// did we record an error?
		if data.Recorded.Error != nil {
			return newComparatorError(stateDBNonce, data.Recorded.Error.Message, data, expectedResultGotError)
		}

		// did we record a valid result?
		if data.Recorded.Result != nil {
			var (
				unmarsh, trimmed string
				b                *big.Int
			)

			trimmed = strings.TrimPrefix(string(data.Recorded.Result), "0x")
			b = new(big.Int)

			_, ok = b.SetString(trimmed, 16)
			if !ok {
				err := json.Unmarshal(data.Recorded.Result, &unmarsh)
				if err != nil {

					return &comparatorError{
						error: err,
						typ:   defaultErrorType,
					}
				}

				b.SetString(strings.TrimPrefix(unmarsh, "0x"), 16)

			}

			recordedNonce = b.Uint64()
		}

		// matching result?
		if stateDBNonce != recordedNonce {
			return newComparatorError(stateDBNonce, recordedNonce, data, noMatchingResult)
		}

		return nil
	}

	return newUnexpectedDataTypeErr(data)
}

// compareCall compares call data recorded on API server with data returned by StateDB
func compareCall(data *OutData) *comparatorError {

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
func compareCallStateDBResult(data *OutData) *comparatorError {
	var recordedStr, stateStr string

	if isZeroAnswer(data.StateDB.Result.([]byte)) {
		stateStr = nilEVMResult
	} else {
		stateStr = fmt.Sprintf("0x%v", hexutils.BytesToHex(data.StateDB.Result.([]byte)))
	}

	// did we record a valid result?
	if data.Recorded.Result != nil {

		err := json.Unmarshal(data.Recorded.Result, &recordedStr)
		if err != nil {
			return newComparatorError(data.Recorded.Result, data.StateDB.Result, data, cannotUnmarshalResult)
		}

		expectedResult := hexutils.HexToBytes(strings.TrimPrefix(recordedStr, "0x"))

		if bytes.Compare(data.StateDB.Result.([]byte), expectedResult) != 0 {

			return newComparatorError(
				stateStr,
				recordedStr,
				data,
				noMatchingResult)
		}
		return nil
	}

	// did we record an error?
	if data.Recorded.Error != nil {
		var returned string
		if v, ok := EVMErrors[data.Recorded.Error.Code]; ok {
			returned = v
		} else {
			returned = fmt.Sprintf("Error code: %v", data.Recorded.Error.Code)
		}
		return newComparatorError(
			stateStr,
			returned,
			data,
			expectedErrorGotResult)
	}
	return nil
}

// isZeroAnswer looks at StateDB result for call and if all bytes are 0, it returns true since it is a zero answer,
// if one byte that is not 0 is found, false is returned immediately
func isZeroAnswer(result json.RawMessage) bool {
	for _, b := range result {
		if b != 0 {
			return false
		}
	}
	return true
}

// compareEVMStateDBError compares error returned from EVM with recorded data
func compareEVMStateDBError(data *OutData) *comparatorError {

	// did we record an error?
	if data.Recorded.Error != nil {
		if !(data.StateDB.Error.Error() == EVMErrors[data.Recorded.Error.Code]) {
			return newComparatorError(
				data.StateDB.Error,
				EVMErrors[data.Recorded.Error.Code],
				data,
				noMatchingErrors)
		}
		return nil
	}

	// did we record a valid result?
	if data.Recorded.Result != nil {
		return newComparatorError(
			data.StateDB.Error,
			string(data.Recorded.Result),
			data,
			expectedResultGotError)
	}

	return nil
}

// compareEstimateGas compares recorded data for estimateGas method with result from StateDB
func compareEstimateGas(data *OutData) *comparatorError {

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
func compareEstimateGasStateDBResult(data *OutData) *comparatorError {

	stateDBGas, ok := data.StateDB.Result.(hexutil.Uint64)
	if !ok {
		return newUnexpectedDataTypeErr(data)
	}

	uintGas := uint64(stateDBGas)

	// did we record a valid result?
	if data.Recorded.Result != nil {
		var (
			trimmed, unmarsh string
			b                *big.Int
		)

		// first try to extract the number the easier way
		trimmed = strings.TrimPrefix(string(data.Recorded.Result), "0x")
		b = new(big.Int)

		// if we fail, we must unmarshal
		_, ok = b.SetString(trimmed, 16)
		if !ok {
			err := json.Unmarshal(data.Recorded.Result, &unmarsh)
			if err != nil {

				return &comparatorError{
					error: err,
					typ:   defaultErrorType,
				}
			}

			b.SetString(strings.TrimPrefix(unmarsh, "0x"), 16)

		}

		if uintGas != b.Uint64() {
			return newComparatorError(
				fmt.Sprintf("0x%v", strconv.FormatUint(uintGas, 16)),
				string(data.Recorded.Result),
				data,
				noMatchingResult)
		}
	}

	// did we record an error?
	if data.Recorded.Error != nil {
		return newComparatorError(
			fmt.Sprintf("0x%v", strconv.FormatUint(uintGas, 16)),
			EVMErrors[data.Recorded.Error.Code],
			data,
			expectedErrorGotResult)
	}

	return nil
}
