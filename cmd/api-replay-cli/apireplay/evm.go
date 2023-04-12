package apireplay

import (
	"errors"
	"fmt"
	"math"
	"math/big"
	"strings"

	"github.com/Fantom-foundation/Aida/state"
	"github.com/Fantom-foundation/go-opera-fvm/opera"
	"github.com/Fantom-foundation/go-opera/ethapi"
	"github.com/Fantom-foundation/go-opera/evmcore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core"
	eth "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/status-im/keycard-go/hexutils"
)

// EVMRequest represents data structure of requests executed in EVM
type EVMRequest struct {
	From, To             *common.Address
	Data                 hexutil.Bytes
	Gas, GasPrice, Value *big.Int
}

// EVM represents requests executed over Ethereum Virtual Machine
type EVM struct {
	req       *EVMRequest
	msg       eth.Message
	evm       *vm.EVM
	archive   state.StateDB
	timestamp uint64
	chainCfg  *params.ChainConfig
	vmImpl    string
	blockID   uint64
}

const maxGasLimit = 9995800 // used when request does not specify gas
const globalGasCap = 50000000

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

	msg = eth.NewMessage(*req.From, req.To, archive.GetNonce(*req.From), req.Value, req.Gas.Uint64(), req.GasPrice, new(big.Int), new(big.Int), req.Data, nil, true)
	txCtx = evmcore.NewEVMTxContext(msg)

	evm = vm.NewEVM(blockCtx, txCtx, archive, chainCfg, vmConfig)

	return &EVM{
		req:       req,
		msg:       msg,
		evm:       evm,
		archive:   archive,
		vmImpl:    vmImpl,
		chainCfg:  chainCfg,
		timestamp: timestamp,
		blockID:   blockID,
	}
}

// newEVMRequest decodes recorded params into a structure
func newEVMRequest(params map[string]interface{}) *EVMRequest {
	req := new(EVMRequest)

	req.From = new(common.Address)
	if v, ok := params["from"]; ok && v != nil {
		*req.From = common.HexToAddress(v.(string))
	}

	req.To = new(common.Address)
	if v, ok := params["to"]; ok && v != nil {
		*req.To = common.HexToAddress(v.(string))
	}

	req.Value = new(big.Int)
	if v, ok := params["value"]; ok && v != nil {
		req.Value, _ = req.Value.SetString(strings.TrimPrefix(v.(string), "0x"), 16)
	}

	req.Gas = new(big.Int)
	if v, ok := params["gas"]; ok && v != nil {
		req.Gas.SetString(strings.TrimPrefix(v.(string), "0x"), 16)
	} else {
		// if gas cap is not specified, we use maxGasLimit
		req.Gas.SetUint64(maxGasLimit)
	}

	req.GasPrice = new(big.Int)
	if v, ok := params["gasPrice"]; ok && v != nil {
		req.GasPrice, _ = new(big.Int).SetString(strings.TrimPrefix(v.(string), "0x"), 16)
	}

	if v, ok := params["data"]; ok && v != nil {
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
		err            error
	)

	// todo try
	hi, lo = findHiLo(evm.req.Gas)
	//hi, lo, err = hilo(evm.req, evm.archive)
	if err != nil {
		return 0, err
	}

	gasCap = hi

	fmt.Printf("lo: %v\n", lo)
	fmt.Printf("hi: %v\n", hi)

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

func (evm *EVM) newEstimateGas(args *ethapi.TransactionArgs) (hexutil.Uint64, error) {
	// Binary search the gas requirement, as it may be higher than the amount used
	var (
		lo  uint64 = params.TxGas - 1
		hi  uint64
		cap uint64
	)
	if args.GasPrice.ToInt().Uint64() == 0 {
		args.GasPrice = nil
	}

	// Use zero address if sender unspecified.
	if args.From == nil {
		args.From = new(common.Address)
	}
	// Determine the highest gas limit can be used during the estimation.
	if args.Gas != nil && uint64(*args.Gas) >= params.TxGas {
		hi = uint64(*args.Gas)
	} else {
		hi = maxGasLimit
	}
	// Normalize the max fee per gas the call is willing to spend.
	var feeCap *big.Int
	if args.GasPrice != nil && (args.MaxFeePerGas != nil || args.MaxPriorityFeePerGas != nil) {
		return 0, errors.New("both gasPrice and (maxFeePerGas or maxPriorityFeePerGas) specified")
	} else if args.GasPrice != nil {
		feeCap = args.GasPrice.ToInt()
	} else if args.MaxFeePerGas != nil {
		feeCap = args.MaxFeePerGas.ToInt()
	} else {
		feeCap = common.Big0
	}
	// Recap the highest gas limit with account's available balance.
	if feeCap.BitLen() != 0 {
		balance := evm.archive.GetBalance(*args.From) // from can't be nil
		available := new(big.Int).Set(balance)
		if args.Value != nil {
			if args.Value.ToInt().Cmp(available) >= 0 {
				return 0, errors.New("insufficient funds for transfer")
			}
			available.Sub(available, args.Value.ToInt())
		}
		allowance := new(big.Int).Div(available, feeCap)

		// If the allowance is larger than maximum uint64, skip checking
		if allowance.IsUint64() && hi > allowance.Uint64() {
			transfer := args.Value
			if transfer == nil {
				transfer = new(hexutil.Big)
			}
			log.Warn("Gas estimation capped by limited funds", "original", hi, "balance", balance,
				"sent", transfer.ToInt(), "maxFeePerGas", feeCap, "fundable", allowance)
			hi = allowance.Uint64()
		}
	}

	// Recap the highest gas allowance with specified gascap.
	if hi > globalGasCap {
		log.Warn("Caller gas above allowance, capping", "requested", hi, "cap", globalGasCap)
		hi = globalGasCap
	}
	cap = hi

	fmt.Printf("lo: %v\n", lo)
	fmt.Printf("hi: %v\n", hi)

	// Create a helper to check if a gas allowance results in an executable transaction
	executable := func(gas uint64, evm *EVM) (bool, *evmcore.ExecutionResult, error) {
		args.Gas = (*hexutil.Uint64)(&gas)
		evm.req.Gas.SetUint64(gas)

		result, err := newEVM(evm.blockID, evm.archive, evm.vmImpl, evm.chainCfg, evm.req, evm.timestamp).sendCall()

		if err != nil {
			if strings.Contains(err.Error(), "intrinsic gas too low") {
				return true, nil, nil // Special case, raise gas limit
			}
			return true, nil, err // Bail out
		}
		return result.Failed(), result, nil
	}
	// Execute the binary search and hone in on an executable gas limit
	for lo+1 < hi {
		mid := (hi + lo) / 2
		failed, _, err := executable(mid, evm)

		// If the error is not nil(consensus error), it means the provided message
		// call or transaction will never be accepted no matter how much gas it is
		// assigned. Return the error directly, don't struggle anymore.
		if err != nil {
			return 0, err
		}
		if failed {
			fmt.Printf("fail; gas: %v\n", mid)
			lo = mid
		} else {
			fmt.Printf("no fail; gas: %v\n", mid)
			hi = mid
		}
	}
	// Reject the transaction as invalid if it still fails at the highest allowance
	if hi == cap {
		failed, result, err := executable(hi, evm)
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

// findHiLo finds the lowest and the highest gas amount possible
func findHiLo(gas *big.Int) (hi, lo uint64) {
	// do we have a gas limit in the request?
	if gas != nil && gas.Uint64() >= params.TxGas {
		hi = gas.Uint64()
	} else {
		hi = maxGasLimit
	}

	lo = params.TxGas - 1

	return
}

//func hilo(args *EVMRequest, archive state.StateDB) (uint64, uint64, error) {
//	// Binary search the gas requirement, as it may be higher than the amount used
//	var (
//		lo uint64 = params.TxGas - 1
//		hi uint64
//	)
//	// Use zero address if sender unspecified.
//	if args.From == nil {
//		args.From = new(common.Address)
//	}
//	// Determine the highest gas limit can be used during the estimation.
//	if args.Gas != nil && args.Gas.Uint64() >= params.TxGas {
//		hi = args.Gas.Uint64()
//	} else {
//		hi = maxGasLimit()
//	}
//	// todo necessary?
//	// Normalize the max fee per gas the call is willing to spend.
//	//var feeCap *big.Int
//	//if args.GasPrice != nil && (args.MaxFeePerGas != nil || args.MaxPriorityFeePerGas != nil) {
//	//	return 0, errors.New("both gasPrice and (maxFeePerGas or maxPriorityFeePerGas) specified")
//	//} else if args.GasPrice != nil {
//	//	feeCap = args.GasPrice.ToInt()
//	//} else if args.MaxFeePerGas != nil {
//	//	feeCap = args.MaxFeePerGas.ToInt()
//	//} else {
//	//	feeCap = common.Big0
//	//}
//
//	// todo remove if necessary
//	feeCap := args.GasPrice
//	// Recap the highest gas limit with account's available balance.
//	if feeCap.BitLen() != 0 {
//		balance := archive.GetBalance(*args.From) // from can't be nil
//		available := new(big.Int).Set(balance)
//		if args.Value != nil {
//			if args.Value.Cmp(available) >= 0 {
//				return 0, 0, errors.New("insufficient funds for transfer")
//			}
//			available.Sub(available, args.Value)
//		}
//		allowance := new(big.Int).Div(available, feeCap)
//
//		// If the allowance is larger than maximum uint64, skip checking
//		if allowance.IsUint64() && hi > allowance.Uint64() {
//			transfer := args.Value
//			if transfer == nil {
//				transfer = new(big.Int)
//			}
//			log.Warn("Gas estimation capped by limited funds", "original", hi, "balance", balance,
//				"sent", transfer, "maxFeePerGas", feeCap, "fundable", allowance)
//			hi = allowance.Uint64()
//		}
//	}
//
//	// todo I guess?
//	var gasCap uint64 = math.MaxUint64
//	// Recap the highest gas allowance with specified gascap.
//	if gasCap != 0 && hi > gasCap {
//		log.Warn("Caller gas above allowance, capping", "requested", hi, "cap", gasCap)
//		hi = gasCap
//	}
//	// todo whats up with cap???
//	//cap = hi
//
//	return hi, lo, nil
//}

// isExecutable tries if transaction is executable with given gas
func isExecutable(gas uint64, evm *EVM) (bool, *evmcore.ExecutionResult, error) {
	evm.req.Gas.SetUint64(gas)

	if gas == 59340 {
		fmt.Println("")
	}

	evmRes, err := newEVM(evm.blockID, evm.archive, evm.vmImpl, evm.chainCfg, evm.req, evm.timestamp).sendCall()
	if err != nil {
		if strings.Contains(err.Error(), "intrinsic gas too low") {
			return true, nil, nil // Special case, raise gas limit
		}
		return true, nil, err // Bailout
	}
	if !evmRes.Failed() {
		fmt.Println(gas)
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
