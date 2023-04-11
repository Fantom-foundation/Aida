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

// executeCall into EVM and return the result
func executeCall(evm *EVM) (out *StateDBData) {
	var (
		result *evmcore.ExecutionResult
		err    error
	)

	out = new(StateDBData)

	// get the result from EVM
	result, err = evm.sendCall()
	if err != nil {
		out.Error = err
		return
	}

	// If the result contains a revert reason, try to unpack and return it.
	if len(result.Revert()) > 0 {
		out.Error = newRevertError(result)
	} else {
		out.Result = result.Return()
		out.Error = result.Err
	}

	return
}

// executeEstimateGas into EVM which calculates gas needed for a transaction
func executeEstimateGas(evm *EVM) (out *StateDBData) {
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
