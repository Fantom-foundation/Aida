package apireplay

import (
	"math/big"

	"github.com/Fantom-foundation/Aida/state"
	"github.com/Fantom-foundation/go-opera/evmcore"
	"github.com/ethereum/go-ethereum/common"
)

// executeGetBalance request into given archive and send result to comparator
func executeGetBalance(param interface{}, archive state.StateDB) (out *StateDBData) {
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
func executeGetTransactionCount(param interface{}, archive state.StateDB) (out *StateDBData) {
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

// executeCall into EVMExecutor and return the result
func executeCall(evm *EVMExecutor) (out *StateDBData) {
	var (
		result *evmcore.ExecutionResult
		err    error
	)

	out = new(StateDBData)

	// get the result from EVMExecutor
	result, err = evm.sendCall()
	if err != nil {
		out.Error = err
		return
	}

	out.Error = result.Err
	out.Result = result.Return()

	return
}

// executeEstimateGas into EVMExecutor which calculates gas needed for a transaction
func executeEstimateGas(evm *EVMExecutor) (out *StateDBData) {
	out = new(StateDBData)

	out.Result, out.Error = evm.sendEstimateGas()

	return
}

// executeGetCode request into given archive and send result to comparator
func executeGetCode(param interface{}, archive state.StateDB) (out *StateDBData) {
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
func executeGetStorageAt(params []interface{}, archive state.StateDB) (out *StateDBData) {
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
