// Copyright 2024 Fantom Foundation
// This file is part of Aida Testing Infrastructure for Sonic
//
// Aida is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Aida is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with Aida. If not, see <http://www.gnu.org/licenses/>.

package rpc

import (
	"encoding/binary"
	"strings"
	"unsafe"

	"github.com/Fantom-foundation/Aida/state"
	"github.com/Fantom-foundation/Aida/txcontext"
	"github.com/Fantom-foundation/Aida/utils"
	"github.com/Fantom-foundation/lachesis-base/common/littleendian"
	"github.com/ethereum/go-ethereum/common"
)

// TODO FIX!
const falsyContract = "0xe0c38b2a8d09aad53f1c67734b9a95e43d5981c0"

func Execute(block uint64, rec *RequestAndResults, archive state.NonCommittableStateDB, cfg *utils.Config) txcontext.Result {
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
		// calls to this contract are excluded for now,
		// this contract causes issues in validation
		if strings.Compare(falsyContract, strings.ToLower(evm.args.To.String())) == 0 {
			rec.SkipValidation = true
		}
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
func executeGetBalance(param interface{}, archive state.VmStateDB) *result {
	address := common.HexToAddress(param.(string))

	return &result{
		result: archive.GetBalance(address).Bytes(),
	}
}

// executeGetTransactionCount request into given archive and send result to comparator
func executeGetTransactionCount(param interface{}, archive state.VmStateDB) *result {
	address := common.HexToAddress(param.(string))
	nonce := archive.GetNonce(address)
	res := &result{result: make([]byte, unsafe.Sizeof(nonce))}
	binary.LittleEndian.PutUint64(res.result, nonce)

	return res
}

// executeCall into EvmExecutor and return the result
func executeCall(evm *EvmExecutor) *result {
	var gasUsed uint64

	exRes, err := evm.sendCall()
	if exRes != nil {
		gasUsed = exRes.UsedGas
	}

	// this situation can happen if request is valid but the response from EVM is empty
	// EVM returns nil instead of an empty result
	var res []byte
	if exRes.ReturnData == nil && err == nil && exRes.Err == nil {
		res = []byte{}
	} else {
		res = exRes.ReturnData
	}

	return &result{
		gasUsed: gasUsed,
		result:  res,
		err:     err,
	}
}

// executeEstimateGas into EvmExecutor which calculates gas needed for a transaction
func executeEstimateGas(evm *EvmExecutor) *result {
	gas, err := evm.sendEstimateGas()
	return &result{
		result: littleendian.Uint64ToBytes(uint64(gas)),
		err:    err,
	}
}

// executeGetCode request into given archive and send result to comparator
func executeGetCode(param interface{}, archive state.VmStateDB) *result {
	address := common.HexToAddress(param.(string))
	return &result{
		result: archive.GetCode(address),
	}
}

// executeGetStorageAt request into given archive and send result to comparator
func executeGetStorageAt(params []interface{}, archive state.VmStateDB) *result {
	address := common.HexToAddress(params[0].(string))
	hash := common.HexToHash(params[1].(string))

	return &result{
		result: archive.GetState(address, hash).Bytes(),
	}
}
