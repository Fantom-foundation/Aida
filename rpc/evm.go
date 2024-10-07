// Copyright 2024 Fantom Foundation
// This file is part of Aida Testing Infrastructure for Sonic
//
// Aida is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Aida is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with Aida. If not, see <http://www.gnu.org/licenses/>.

package rpc

import (
	"errors"
	"fmt"
	"math"
	"math/big"
	"strings"

	"github.com/Fantom-foundation/Aida/state"
	"github.com/Fantom-foundation/Aida/utils"
	"github.com/Fantom-foundation/go-opera/ethapi"
	"github.com/Fantom-foundation/go-opera/evmcore"
	"github.com/Fantom-foundation/go-opera/opera"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
	"github.com/status-im/keycard-go/hexutils"
)

// EvmExecutor represents requests executed over Ethereum Virtual Machine
type EvmExecutor struct {
	args      ethapi.TransactionArgs
	archive   state.NonCommittableStateDB
	timestamp uint64 // EVM requests require timestamp for correct execution
	chainCfg  *params.ChainConfig
	vmImpl    vm.InterpreterFactory
	blockId   *big.Int
	rules     opera.EconomyRules
}

const maxGasLimit = 9995800     // used when request does not specify gas
const globalGasCap = 50_000_000 // highest gas allowance used for estimateGas

// newEvmExecutor creates EvmExecutor for executing requests into StateDB that demand usage of EVM
func newEvmExecutor(blockID uint64, archive state.NonCommittableStateDB, cfg *utils.Config, params map[string]interface{}, timestamp uint64) (*EvmExecutor, error) {
	factory, err := cfg.GetInterpreterFactory()
	if err != nil {
		return nil, fmt.Errorf("cannot get interpreter factory: %w", err)
	}
	chainCfg, err := cfg.GetChainConfig("")
	if err != nil {
		return nil, fmt.Errorf("cannot get chain config: %w", err)
	}

	return &EvmExecutor{
		args:      newTxArgs(params),
		archive:   archive,
		timestamp: timestamp,
		chainCfg:  chainCfg,
		vmImpl:    factory,
		blockId:   new(big.Int).SetUint64(blockID),
		rules:     opera.DefaultEconomyRules(),
	}, nil
}

// newTxArgs decodes recorded params into ethapi.TransactionArgs
func newTxArgs(params map[string]interface{}) ethapi.TransactionArgs {
	var args ethapi.TransactionArgs

	if v, ok := params["from"]; ok && v != nil {
		args.From = new(common.Address)
		*args.From = common.HexToAddress(v.(string))
	}

	if v, ok := params["to"]; ok && v != nil {
		args.To = new(common.Address)
		*args.To = common.HexToAddress(v.(string))
	}

	if v, ok := params["value"]; ok && v != nil {
		value := new(big.Int)
		value.SetString(strings.TrimPrefix(v.(string), "0x"), 16)
		args.Value = (*hexutil.Big)(value)
	}

	args.Gas = new(hexutil.Uint64)
	if v, ok := params["gas"]; ok && v != nil {
		gas := new(big.Int)
		gas.SetString(strings.TrimPrefix(v.(string), "0x"), 16)
		*args.Gas = hexutil.Uint64(gas.Uint64())
	} else {
		// if gas cap is not specified, we use maxGasLimit
		*args.Gas = hexutil.Uint64(maxGasLimit)
	}

	if v, ok := params["gasPrice"]; ok && v != nil {
		gasPrice := new(big.Int)
		gasPrice.SetString(strings.TrimPrefix(v.(string), "0x"), 16)
		args.GasPrice = new(hexutil.Big)
		args.GasPrice = (*hexutil.Big)(gasPrice)
	}

	if v, ok := params["data"]; ok && v != nil {
		s := strings.TrimPrefix(v.(string), "0x")
		data := hexutils.HexToBytes(s)
		args.Data = new(hexutil.Bytes)
		args.Data = (*hexutil.Bytes)(&data)
	}

	return args
}

// newEVM creates new instance of EVM with given parameters
func (e *EvmExecutor) newEVM(msg *core.Message, hashErr *error) *vm.EVM {
	var (
		getHash  func(uint64) common.Hash
		blockCtx vm.BlockContext
		vmConfig vm.Config
		txCtx    vm.TxContext
	)

	getHash = func(_ uint64) common.Hash {
		h, err := e.archive.GetHash()
		*hashErr = err
		return h
	}

	blockCtx = vm.BlockContext{
		CanTransfer: core.CanTransfer,
		Transfer:    core.Transfer,
		Coinbase:    common.Address{}, // opera based value
		BlockNumber: e.blockId,
		Difficulty:  big.NewInt(1),  // evmcore/evm.go
		GasLimit:    math.MaxUint64, // evmcore/dummy_block.go
		GetHash:     getHash,
		BaseFee:     e.rules.MinGasPrice, // big.NewInt(1e9)
		Time:        e.timestamp,
	}

	vmConfig = opera.DefaultVMConfig
	vmConfig.NoBaseFee = true
	vmConfig.Interpreter = e.vmImpl

	txCtx = evmcore.NewEVMTxContext(msg)

	return vm.NewEVM(blockCtx, txCtx, e.archive, e.chainCfg, vmConfig)
}

// sendCall executes the call method in the EvmExecutor with given archive
func (e *EvmExecutor) sendCall() (*core.ExecutionResult, error) {
	var (
		gp              *core.GasPool
		executionResult *core.ExecutionResult
		err             error
		msg             *core.Message
		evm             *vm.EVM
	)

	gp = new(core.GasPool).AddGas(math.MaxUint64) // based in opera
	msg, err = e.args.ToMessage(globalGasCap, e.rules.MinGasPrice)
	if err != nil {
		return nil, err
	}

	var hashErr *error
	evm = e.newEVM(msg, hashErr)

	executionResult, err = core.ApplyMessage(evm, msg, gp)
	if executionResult.Err != nil {
		return nil, fmt.Errorf("execution returned err; %w", executionResult.Err)
	}

	if hashErr != nil {
		return nil, fmt.Errorf("cannot get state hash; %w", *hashErr)
	}

	// If the timer caused an abort, return an appropriate error message
	if evm.Cancelled() {
		return nil, fmt.Errorf("execution aborted: timeout")
	}
	if err != nil {
		return executionResult, fmt.Errorf("err: %v (supplied gas %v)", err, e.args.Gas)
	}
	return executionResult, nil

}

// sendEstimateGas executes estimateGas method in the EvmExecutor
// It calculates how much gas would transaction need if it was executed
func (e *EvmExecutor) sendEstimateGas() (hexutil.Uint64, error) {
	hi, lo, cap, err := e.findHiLoCap()
	if err != nil {
		return 0, err
	}

	// Execute the binary search and hone in on an executable gas limit
	for lo+1 < hi {
		mid := (hi + lo) / 2
		failed, _, err := e.executable(mid)

		// If the error is not nil(consensus error), it means the provided message
		// call or transaction will never be accepted no matter how much gas it is
		// assigned. Return the error directly, don't struggle anymore.
		if err != nil {
			return 0, err
		}
		if failed {
			lo = mid
		} else {
			hi = mid
		}
	}
	// Reject the transaction as invalid if it still fails at the highest allowance
	if hi == cap {
		failed, result, err := e.executable(hi)
		if err != nil {
			return 0, err
		}
		if failed {
			if result != nil && result.Err != vm.ErrOutOfGas {
				if len(result.Revert()) > 0 {
					return 0, result.Err
				}
				return 0, result.Err
			}
			// Otherwise, the specified gas cap is too low
			return 0, fmt.Errorf("gas required exceeds allowance (%d)", cap)
		}
	}
	return hexutil.Uint64(hi), nil
}

// executable tries to execute call with given gas into EVM. This func is used for estimateGas calculation
func (e *EvmExecutor) executable(gas uint64) (bool, *core.ExecutionResult, error) {
	e.args.Gas = (*hexutil.Uint64)(&gas)

	result, err := e.sendCall()

	if err != nil {
		if strings.Contains(err.Error(), "intrinsic gas too low") {
			return true, nil, nil // Special case, raise gas limit
		}
		return true, nil, err // Bail out
	}
	return result.Failed(), result, nil
}

func (e *EvmExecutor) findHiLoCap() (uint64, uint64, uint64, error) {
	// Binary search the gas requirement, as it may be higher than the amount used
	var (
		lo  = params.TxGas - 1
		hi  uint64
		cap uint64
	)

	// Use zero address if sender unspecified.
	if e.args.From == nil {
		e.args.From = new(common.Address)
	}
	// Determine the highest gas limit can be used during the estimation.
	if e.args.Gas != nil && uint64(*e.args.Gas) >= params.TxGas {
		hi = uint64(*e.args.Gas)
	} else {
		hi = maxGasLimit
	}
	// Normalize the max fee per gas the call is willing to spend.
	var feeCap *big.Int
	if e.args.GasPrice != nil && (e.args.MaxFeePerGas != nil || e.args.MaxPriorityFeePerGas != nil) {
		return 0, 0, 0, errors.New("both gasPrice and (maxFeePerGas or maxPriorityFeePerGas) specified")
	} else if e.args.GasPrice != nil {
		feeCap = e.args.GasPrice.ToInt()
	} else if e.args.MaxFeePerGas != nil {
		feeCap = e.args.MaxFeePerGas.ToInt()
	} else {
		feeCap = common.Big0
	}
	// Recap the highest gas limit with account's available balance.
	if feeCap.BitLen() != 0 {
		balance := e.archive.GetBalance(*e.args.From) // from can't be nil
		available := balance.ToBig()
		if e.args.Value != nil {
			if e.args.Value.ToInt().Cmp(available) >= 0 {
				return 0, 0, 0, errors.New("insufficient funds for transfer")
			}
			available.Sub(available, e.args.Value.ToInt())
		}
		allowance := new(big.Int).Div(available, feeCap)

		// If the allowance is larger than maximum uint64, skip checking
		if allowance.IsUint64() && hi > allowance.Uint64() {
			transfer := e.args.Value
			if transfer == nil {
				transfer = new(hexutil.Big)
			}
			hi = allowance.Uint64()
		}
	}

	// Recap the highest gas allowance with specified gascap.
	if hi > globalGasCap {
		hi = globalGasCap
	}
	cap = hi

	return hi, lo, cap, nil
}
