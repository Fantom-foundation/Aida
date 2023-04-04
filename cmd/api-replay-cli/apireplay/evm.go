package apireplay

import (
	"errors"
	"fmt"
	"math"
	"math/big"
	"strings"

	"github.com/Fantom-foundation/Aida/state"
	"github.com/Fantom-foundation/go-opera-fvm/opera"
	"github.com/Fantom-foundation/go-opera/evmcore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core"
	eth "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
	"github.com/status-im/keycard-go/hexutils"
)

// EVMRequest represents data structure of requests executed in EVM
type EVMRequest struct {
	From, To             common.Address
	Data                 hexutil.Bytes
	Gas, GasPrice, Value *big.Int
}

// EVM represents requests executed over Ethereum Virtual Machine
type EVM struct {
	req     *EVMRequest
	msg     eth.Message
	evm     *vm.EVM
	archive state.StateDB
}

const globalGasCap = 50000000 // used when request does not specify gas

// newEVM creates EVM for comparing data recorded on API with StateDB
func newEVM(blockID uint64, archive state.StateDB, vmImpl string, chainCfg *params.ChainConfig, req *EVMRequest, timestamp uint64) *EVM {
	var (
		bigBlockId *big.Int
		getHash    func(uint64) common.Hash
		rules      opera.EconomyRules
		blockCtx   vm.BlockContext
		vmConfig   vm.Config
		msg        eth.Message
		txCtx      vm.TxContext
		evm        *vm.EVM
	)

	bigBlockId = new(big.Int).SetUint64(blockID)

	// for purpose of comparing, we need not a hash func
	getHash = func(_ uint64) common.Hash {
		return common.Hash{}
	}

	rules = opera.DefaultEconomyRules()

	blockCtx = vm.BlockContext{
		CanTransfer: core.CanTransfer,
		Transfer:    core.Transfer,
		Coinbase:    common.Address{}, // opera based value
		BlockNumber: bigBlockId,
		Difficulty:  big.NewInt(1),  // evmcore/evm.go
		GasLimit:    math.MaxUint64, // evmcore/dummy_block.go
		GetHash:     getHash,
		BaseFee:     rules.MinGasPrice, // big.NewInt(1e9)
		Time:        new(big.Int).SetUint64(timestamp),
	}

	vmConfig = opera.DefaultVMConfig
	vmConfig.NoBaseFee = true
	vmConfig.InterpreterImpl = vmImpl

	msg = eth.NewMessage(req.From, &req.To, archive.GetNonce(req.From), req.Value, req.Gas.Uint64(), req.GasPrice, new(big.Int), new(big.Int), req.Data, nil, true)
	txCtx = evmcore.NewEVMTxContext(msg)

	evm = vm.NewEVM(blockCtx, txCtx, archive, chainCfg, vmConfig)

	return &EVM{
		req:     req,
		msg:     msg,
		evm:     evm,
		archive: archive,
	}
}

// newEVMRequest decodes recorded params into a structure
func newEVMRequest(params map[string]interface{}) *EVMRequest {
	req := new(EVMRequest)

	if v, ok := params["from"]; ok {
		req.From = common.HexToAddress(v.(string))
	}

	if v, ok := params["to"]; ok {
		req.To = common.HexToAddress(v.(string))
	}

	req.Value = new(big.Int)
	if v, ok := params["value"]; ok {
		req.Value, _ = req.Value.SetString(strings.TrimPrefix(v.(string), "0x"), 16)
	}

	req.Gas = new(big.Int)
	if v, ok := params["gas"]; ok {
		req.Gas.SetString(strings.TrimPrefix(v.(string), "0x"), 16)
	} else {
		// if gas cap is not specified, we use globalGasCap
		req.Gas.SetUint64(globalGasCap)
	}

	req.GasPrice = new(big.Int)
	if v, ok := params["gasPrice"]; ok {
		req.GasPrice, _ = new(big.Int).SetString(strings.TrimPrefix(v.(string), "0x"), 16)
	}

	if v, ok := params["data"]; ok {
		s := strings.TrimPrefix(v.(string), "0x")
		req.Data = hexutils.HexToBytes(s)
	}

	return req
}

// sendCall executes the call method in the EVM with given archive
func (evm *EVM) sendCall() (*evmcore.ExecutionResult, error) {
	var (
		gp     *evmcore.GasPool
		result *evmcore.ExecutionResult
		err    error
	)

	gp = new(evmcore.GasPool).AddGas(math.MaxUint64) // based in opera

	result, err = evmcore.ApplyMessage(evm.evm, evm.msg, gp)

	// If the timer caused an abort, return an appropriate error message
	if evm.evm.Cancelled() {
		return nil, fmt.Errorf("execution aborted: timeout")
	}
	if err != nil {
		return result, fmt.Errorf("err: %v (supplied gas %v)", err, evm.msg.Gas())
	}
	return result, nil

}

// sendEstimateGas executes estimateGas method in the EVM
// It calculates how much gas would transaction need if it was executed
func (evm *EVM) sendEstimateGas() (hexutil.Uint64, error) {
	var (
		lo, hi, gasCap uint64
	)

	hi, lo = findHiLo(evm.req.Gas)

	gasCap = hi

	// Execute the binary search and hone in on an isExecutable gas limit
	for lo+1 < hi {
		mid := (hi + lo) / 2
		failed, _, err := isExecutable(mid, evm)

		// If the error is not nil(consensus error), it means the provided message
		// compareCall or transaction will never be accepted no matter how much gas it is
		// assigned. Return the error directly, don't struggle anymore.
		if err != nil {
			return 0, err
		}

		if failed {
			// the given gas was not enough - raise it
			lo = mid
		} else {
			// the given gas was enough - lower it
			hi = mid
		}
	}

	// Reject the transaction as invalid if it still fails at the highest allowance
	if err := compareHiAndCap(hi, gasCap, evm); err != nil {
		return 0, err
	}
	return hexutil.Uint64(hi), nil
}

// findHiLo finds the lowest and the highest gas amount possible
func findHiLo(gas *big.Int) (hi, lo uint64) {
	// do we have a gas limit in the request?
	if gas != nil && gas.Uint64() >= params.TxGas {
		hi = gas.Uint64()
	} else {
		hi = maxGasLimit()
	}

	lo = params.TxGas - 1

	return
}

// isExecutable tries if transaction is executable with given gas
func isExecutable(gas uint64, evm *EVM) (bool, *evmcore.ExecutionResult, error) {
	evm.req.Gas.SetUint64(gas)

	evmRes, err := evm.sendCall()
	if err != nil {
		if errors.Is(err, evmcore.ErrIntrinsicGas) {
			return true, nil, nil // Special case, raise gas limit
		}
		return true, nil, err // Bailout
	}
	return evmRes.Failed(), evmRes, nil
}

// compareHiAndCap so we know whether transaction fails with the highest possible gas
func compareHiAndCap(hi, cap uint64, evm *EVM) error {
	if hi == cap {
		failed, result, err := isExecutable(hi, evm)
		if err != nil {
			return err
		}
		if failed {
			if result != nil && result.Err != vm.ErrOutOfGas {
				if len(result.Revert()) > 0 {
					return newRevertError(result)
				}
				return result.Err
			}
			// Otherwise, the specified gas cap is too low
			return fmt.Errorf("gas required exceeds allowance (%d)", cap)
		}
	}

	return nil
}

// maxGasLimit returns the maximum gas limit for current rules
func maxGasLimit() uint64 {
	dag := opera.DefaultDagRules()
	economy := opera.DefaultEconomyRules()
	maxEmptyEventGas := economy.Gas.EventGas +
		uint64(dag.MaxParents-dag.MaxFreeParents)*economy.Gas.ParentGas +
		uint64(dag.MaxExtraData)*economy.Gas.ExtraDataGas
	if economy.Gas.MaxEventGas < maxEmptyEventGas {
		return 0
	}
	return economy.Gas.MaxEventGas - maxEmptyEventGas
}
