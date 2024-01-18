package rpc

import (
	"math/big"
	"time"

	"github.com/Fantom-foundation/Aida/state"
	"github.com/Fantom-foundation/Aida/utils"
	"github.com/Fantom-foundation/go-opera/evmcore"
	"github.com/ethereum/go-ethereum/common"
)

// StateDBData represents data that StateDB returned for requests recorded on API server
// This is sent to Comparator and compared with RecordedData
type StateDBData struct {
	Result      any
	Error       error
	IsRecovered bool
}

func Execute(block uint64, rec *RequestAndResults, archive state.NonCommittableStateDB, cfg *utils.Config) *StateDBData {
	switch rec.Query.MethodBase {
	case "getBalance":
		return executeGetBalance(rec.Query.Params[0], archive)

	case "getTransactionCount":
		return executeGetTransactionCount(rec.Query.Params[0], archive)

	case "call":
		var timestamp uint64

		// first try to extract timestamp from response
		if rec.Response != nil {
			if rec.Response.Timestamp != 0 {
				timestamp = uint64(time.Unix(0, int64(rec.Response.Timestamp)).Unix())
			}
		} else if rec.Error != nil {
			if rec.Error.Timestamp != 0 {

				timestamp = uint64(time.Unix(0, int64(rec.Error.Timestamp)).Unix())
			}
		}

		if timestamp == 0 {
			return nil
		}

		evm := newEvmExecutor(block, archive, cfg, rec.Query.Params[0].(map[string]interface{}), timestamp)
		return executeCall(evm)

	case "estimateGas":
		// estimateGas is currently not suitable for rpc replay since the estimation  in geth is always calculated for current state
		// that means recorded result and result returned by StateDB are not comparable
	case "getCode":
		return executeGetCode(rec.Query.Params[0], archive)

	case "getStorageAt":
		return executeGetStorageAt(rec.Query.Params, archive)

	default:
		break
	}
	return nil
}

// executeGetBalance request into given archive and send result to comparator
func executeGetBalance(param interface{}, archive state.VmStateDB) (out *StateDBData) {
	var (
		address common.Address
	)

	out = new(StateDBData)
	out.Result = new(big.Int)

	// decode requested address
	address = common.HexToAddress(param.(string))

	// retrieve compareBalance
	out.Result = archive.GetBalance(address)

	return
}

// executeGetTransactionCount request into given archive and send result to comparator
func executeGetTransactionCount(param interface{}, archive state.VmStateDB) (out *StateDBData) {
	var (
		address common.Address
	)

	out = new(StateDBData)

	// decode requested address
	address = common.HexToAddress(param.(string))

	// retrieve nonce
	out.Result = archive.GetNonce(address)

	return
}

// executeCall into EvmExecutor and return the result
func executeCall(evm *EvmExecutor) (out *StateDBData) {
	var (
		result *evmcore.ExecutionResult
		err    error
	)

	out = new(StateDBData)

	// get the result from EvmExecutor
	result, err = evm.sendCall()
	if err != nil {
		out.Error = err
		return
	}

	out.Error = result.Err
	out.Result = result.Return()

	return
}

// executeEstimateGas into EvmExecutor which calculates gas needed for a txcontext
func executeEstimateGas(evm *EvmExecutor) (out *StateDBData) {
	out = new(StateDBData)

	out.Result, out.Error = evm.sendEstimateGas()

	return
}

// executeGetCode request into given archive and send result to comparator
func executeGetCode(param interface{}, archive state.VmStateDB) (out *StateDBData) {
	var (
		address common.Address
	)

	out = new(StateDBData)

	// decode requested address
	address = common.HexToAddress(param.(string))

	// retrieve nonce
	out.Result = archive.GetCode(address)

	return
}

// executeGetStorageAt request into given archive and send result to comparator
func executeGetStorageAt(params []interface{}, archive state.VmStateDB) (out *StateDBData) {
	var (
		address   common.Address
		hash, res common.Hash
	)

	out = new(StateDBData)

	// decode requested address and position in storage
	address = common.HexToAddress(params[0].(string))
	hash = common.HexToHash(params[1].(string))

	// retrieve nonce
	res = archive.GetState(address, hash)

	out.Result = res[:]

	return
}
