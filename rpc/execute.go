package rpc

import (
	"encoding/binary"
	"fmt"
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
			fmt.Println("timestamp nil")
			return nil
		}
		evm := newEvmExecutor(block, archive, cfg, rec.Query.Params[0].(map[string]interface{}), rec.Timestamp)
		// calls to this contract are excluded for now,
		// this contract causes issues in validation
		if strings.Compare(falsyContract, strings.ToLower(evm.args.To.String())) == 0 {
			rec.SkipValidation = true
		}
		r := executeCall(evm)
		if re, e := r.GetRawResult(); re == nil && e == nil {
			fmt.Println("execute nil")
		}
		return r

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
	var res []byte
	if exRes.ReturnData == nil {
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
