package rpc

import (
	"strings"

	"github.com/Fantom-foundation/Aida/state"
	"github.com/Fantom-foundation/Aida/txcontext"
	"github.com/Fantom-foundation/Aida/utils"
	"github.com/Fantom-foundation/lachesis-base/common/littleendian"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// TODO FIX!
const falsyContract = "0xe0c38b2a8d09aad53f1c67734b9a95e43d5981c0"

// StateDBData represents data that StateDB returned for requests recorded on API server
// This is sent to Comparator and compared with RecordedData
//type StateDBData struct {
//	Message      any
//	Error       error
//	IsRecovered bool
//	GasUsed     uint64
//}

func Execute(block uint64, rec *RequestAndResults, archive state.NonCommittableStateDB, cfg *utils.Config) txcontext.Receipt {
	switch rec.Query.MethodBase {
	case "getBalance":
		return executeGetBalance(rec.Query.Params[0], archive)

	case "getTransactionCount":
		return executeGetTransactionCount(rec.Query.Params[0], archive)

	case "call":
		if rec.RecordedTimestamp == 0 {
			return nil
		}

		evm := newEvmExecutor(block, archive, cfg, rec.Query.Params[0].(map[string]interface{}), rec.RecordedTimestamp)

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

	return &result{
		result: littleendian.Uint64ToBytes(nonce),
	}
}

// executeCall into EvmExecutor and return the result
func executeCall(evm *EvmExecutor) *result {
	var (
		gasUsed uint64
		status  uint64
	)

	exRes, err := evm.sendCall()
	if exRes != nil {
		gasUsed = exRes.UsedGas
		if exRes.Failed() {
			status = types.ReceiptStatusFailed
		} else {
			status = types.ReceiptStatusSuccessful
		}
	}

	return &result{
		status:  status,
		gasUsed: gasUsed,
		result:  exRes.ReturnData,
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
