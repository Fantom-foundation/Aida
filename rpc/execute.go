package rpc

import (
	"math/big"

	"github.com/Fantom-foundation/Aida/state"
	"github.com/Fantom-foundation/Aida/utils"
	"github.com/Fantom-foundation/go-opera/evmcore"
	"github.com/ethereum/go-ethereum/common"
)

// ReturnState represents data that StateDB returned for requests recorded on API server
// This is sent to Comparator and compared with RecordedData
type ReturnState struct {
	Result      any
	Error       error
	IsRecovered bool
}

func Execute(block uint64, rec *RequestAndResults, archive state.NonCommittableStateDB, cfg *utils.Config) *ReturnState {
	switch rec.Query.MethodBase {
	case "getBalance":
		return executeGetBalance(rec.Query.Params[0], archive)

	case "getTransactionCount":
		return executeGetTransactionCount(rec.Query.Params[0], archive)

	case "call":
		if rec.Timestamp == 0 {
			return nil
		}

		evm := newEvmExecutor(block, archive, cfg, rec.Query.Params[0].(map[string]interface{}), rec.Timestamp)
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
func executeGetBalance(param interface{}, archive state.VmStateDB) (out *ReturnState) {
	var (
		address common.Address
	)

	out = new(ReturnState)
	out.Result = new(big.Int)

	// decode requested address
	address = common.HexToAddress(param.(string))

	// retrieve compareBalance
	out.Result = archive.GetBalance(address)

	return
}

// executeGetTransactionCount request into given archive and send result to comparator
func executeGetTransactionCount(param interface{}, archive state.VmStateDB) (out *ReturnState) {
	var (
		address common.Address
	)

	out = new(ReturnState)

	// decode requested address
	address = common.HexToAddress(param.(string))

	// retrieve nonce
	out.Result = archive.GetNonce(address)

	return
}

// executeCall into EvmExecutor and return the result
func executeCall(evm *EvmExecutor) (out *ReturnState) {
	var (
		result *evmcore.ExecutionResult
		err    error
	)

	out = new(ReturnState)

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

// executeEstimateGas into EvmExecutor which calculates gas needed for a transaction
func executeEstimateGas(evm *EvmExecutor) (out *ReturnState) {
	out = new(ReturnState)

	out.Result, out.Error = evm.sendEstimateGas()

	return
}

// executeGetCode request into given archive and send result to comparator
func executeGetCode(param interface{}, archive state.VmStateDB) (out *ReturnState) {
	var (
		address common.Address
	)

	out = new(ReturnState)

	// decode requested address
	address = common.HexToAddress(param.(string))

	// retrieve nonce
	out.Result = archive.GetCode(address)

	return
}

// executeGetStorageAt request into given archive and send result to comparator
func executeGetStorageAt(params []interface{}, archive state.VmStateDB) (out *ReturnState) {
	var (
		address   common.Address
		hash, res common.Hash
	)

	out = new(ReturnState)

	// decode requested address and position in storage
	address = common.HexToAddress(params[0].(string))
	hash = common.HexToHash(params[1].(string))

	// retrieve nonce
	res = archive.GetState(address, hash)

	out.Result = res[:]

	return
}
