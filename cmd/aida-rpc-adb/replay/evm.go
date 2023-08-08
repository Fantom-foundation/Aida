package replay

import (
	"errors"
	"fmt"
	"math"
	"math/big"
	"strings"

	"github.com/Fantom-foundation/Aida/state"
	"github.com/Fantom-foundation/go-opera/ethapi"
	"github.com/Fantom-foundation/go-opera/evmcore"
	"github.com/Fantom-foundation/go-opera/opera"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core"
	eth "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
	"github.com/op/go-logging"
	"github.com/status-im/keycard-go/hexutils"
)

// EVMExecutor represents requests executed over Ethereum Virtual Machine
type EVMExecutor struct {
	args      ethapi.TransactionArgs
	archive   state.StateDB
	timestamp uint64 // EVM requests require timestamp for correct execution
	chainCfg  *params.ChainConfig
	vmImpl    string
	blockID   *big.Int
	rules     opera.EconomyRules
	log       *logging.Logger
}

const maxGasLimit = 9995800   // used when request does not specify gas
const globalGasCap = 50000000 // highest gas allowance used for estimateGas

// newEVMExecutor creates EVMExecutor for executing requests into StateDB that demand usage of EVM
func newEVMExecutor(blockID uint64, archive state.StateDB, vmImpl string, chainCfg *params.ChainConfig, params map[string]interface{}, timestamp uint64, log *logging.Logger) *EVMExecutor {
	return &EVMExecutor{
		args:      newTxArgs(params),
		archive:   archive,
		timestamp: timestamp,
		chainCfg:  chainCfg,
		vmImpl:    vmImpl,
		blockID:   new(big.Int).SetUint64(blockID),
		rules:     opera.DefaultEconomyRules(),
		log:       log,
	}
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
func (e *EVMExecutor) newEVM(msg eth.Message) *vm.EVM {
	var (
		getHash  func(uint64) common.Hash
		blockCtx vm.BlockContext
		vmConfig vm.Config
		txCtx    vm.TxContext
	)

	// for purpose of comparing, we need not a hash func
	getHash = func(_ uint64) common.Hash {
		return common.Hash{}
	}

	blockCtx = vm.BlockContext{
		CanTransfer: core.CanTransfer,
		Transfer:    core.Transfer,
		Coinbase:    common.Address{}, // opera based value
		BlockNumber: e.blockID,
		Difficulty:  big.NewInt(1),  // evmcore/evm.go
		GasLimit:    math.MaxUint64, // evmcore/dummy_block.go
		GetHash:     getHash,
		BaseFee:     e.rules.MinGasPrice, // big.NewInt(1e9)
		Time:        new(big.Int).SetUint64(e.timestamp),
	}

	vmConfig = opera.DefaultVMConfig
	vmConfig.NoBaseFee = true
	vmConfig.InterpreterImpl = e.vmImpl

	txCtx = evmcore.NewEVMTxContext(msg)

	return vm.NewEVM(blockCtx, txCtx, e.archive, e.chainCfg, vmConfig)
}

// sendCall executes the call method in the EVMExecutor with given archive
func (e *EVMExecutor) sendCall() (*evmcore.ExecutionResult, error) {
	var (
		gp     *evmcore.GasPool
		result *evmcore.ExecutionResult
		err    error
		msg    eth.Message
		evm    *vm.EVM
	)

	gp = new(evmcore.GasPool).AddGas(math.MaxUint64) // based in opera
	msg, err = e.args.ToMessage(globalGasCap, e.rules.MinGasPrice)
	if err != nil {
		return nil, err
	}

	evm = e.newEVM(msg)

	result, err = evmcore.ApplyMessage(evm, msg, gp)

	// If the timer caused an abort, return an appropriate error message
	if evm.Cancelled() {
		return nil, fmt.Errorf("execution aborted: timeout")
	}
	if err != nil {
		return result, fmt.Errorf("err: %v (supplied gas %v)", err, e.args.Gas)
	}
	return result, nil

}

// sendEstimateGas executes estimateGas method in the EVMExecutor
// It calculates how much gas would transaction need if it was executed
func (e *EVMExecutor) sendEstimateGas() (hexutil.Uint64, error) {
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
					return 0, newRevertError(result)
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
func (e *EVMExecutor) executable(gas uint64) (bool, *evmcore.ExecutionResult, error) {
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

func (e *EVMExecutor) findHiLoCap() (uint64, uint64, uint64, error) {
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
		available := new(big.Int).Set(balance)
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
