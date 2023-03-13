package utils

import (
	"bytes"
	"fmt"
	"log"
	"math/big"
	"strings"

	"github.com/Fantom-foundation/Aida/state"
	substate "github.com/Fantom-foundation/Substate"
	"github.com/Fantom-foundation/go-opera/evmcore"
	"github.com/Fantom-foundation/go-opera/opera"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
)

// Count total errors occured while processing transactions
const (
	MaxErrors       = 50   // maximum number of errors before terminating program
	UpdateOnFailure = true // update when statedb validation detects discrepancy
)

var (
	NumErrors int   // total number of errors across processed transactions
	hashError error // error when retriving block hashes
)

// runVMTask executes VM on a chosen storage system.
func ProcessTx(db state.StateDB, cfg *Config, block uint64, txIndex int, tx *substate.Substate) (txerr error) {

	inputEnv := tx.Env

	var (
		gaspool   = new(evmcore.GasPool)
		blockHash = common.HexToHash(fmt.Sprintf("0x%016d", block))
		txHash    = common.HexToHash(fmt.Sprintf("0x%016d%016d", block, txIndex))
		newErrors int
		errMsg    strings.Builder
	)
	defer handleErrorOnExit(&txerr, &errMsg, &newErrors, cfg.ContinueOnFailure)
	vmConfig := opera.DefaultVMConfig
	vmConfig.NoBaseFee = true
	vmConfig.InterpreterImpl = cfg.VmImpl
	hashError = nil
	errMsg.WriteString(fmt.Sprintf("Block: %v Transaction: %v\n", block, txIndex))
	// get chain configuration
	chainConfig := GetChainConfig(cfg.ChainID)

	// validate whether the input alloc is contained in the db
	if cfg.ValidateTxState {
		if err := ValidateStateDB(tx.InputAlloc, db, UpdateOnFailure); err != nil {
			newErrors++
			errMsg.WriteString("Input alloc is not contained in the stateDB.\n")
			errMsg.WriteString(err.Error())
			if !cfg.ContinueOnFailure {
				return
			}
		}
	}

	// prepare tx
	gaspool.AddGas(inputEnv.GasLimit)
	msg := tx.Message.AsMessage()
	db.Prepare(txHash, txIndex)
	blockCtx := prepareBlockCtx(inputEnv)
	txCtx := evmcore.NewEVMTxContext(msg)
	evm := vm.NewEVM(*blockCtx, txCtx, db, chainConfig, vmConfig)
	snapshot := db.Snapshot()
	// call ApplyMessage
	msgResult, err := evmcore.ApplyMessage(evm, msg, gaspool)

	// if transaction fails, revert to the first snapshot.
	if err != nil {
		db.RevertToSnapshot(snapshot)
		newErrors++
		errMsg.WriteString(err.Error())
		if !cfg.ContinueOnFailure {
			return
		}
	}
	if hashError != nil {
		newErrors++
		errMsg.WriteString(hashError.Error())
		if !cfg.ContinueOnFailure {
			return
		}
	}
	if chainConfig.IsByzantium(blockCtx.BlockNumber) {
		db.Finalise(true)
	} else {
		db.IntermediateRoot(chainConfig.IsEIP158(blockCtx.BlockNumber))
	}

	// check whether the outputAlloc substate is contained in the world-state db.
	if cfg.ValidateTxState {
		// validate result
		logs := db.GetLogs(txHash, blockHash)
		if cfg.DbImpl == "carmen" {
			//ignore log comparison in carmen
			logs = tx.Result.Logs
		}
		var contract common.Address
		if to := msg.To(); to == nil {
			contract = crypto.CreateAddress(evm.TxContext.Origin, msg.Nonce())
		}
		vmResult := compileVMResult(logs, msgResult, contract)
		if err := validateVMResult(vmResult, tx.Result); err != nil {
			newErrors++
			errMsg.WriteString(err.Error())
			if !cfg.ContinueOnFailure {
				return
			}
		}

		// validate state
		if err := validateVMAlloc(db, tx.OutputAlloc, cfg.StateValidationMode); err != nil {
			newErrors++
			errMsg.WriteString("Output alloc is not contained in the stateDB.\n")
			errMsg.WriteString(err.Error())
			if !cfg.ContinueOnFailure {
				return
			}
		}
	}
	return
}

// prepareBlockCtx creates a block context for evm call from an environment of a substate.
func prepareBlockCtx(inputEnv *substate.SubstateEnv) *vm.BlockContext {
	getHash := func(num uint64) common.Hash {
		if inputEnv.BlockHashes == nil {
			hashError = fmt.Errorf("getHash(%d) invoked, no blockhashes provided", num)
			return common.Hash{}
		}
		h, ok := inputEnv.BlockHashes[num]
		if !ok {
			hashError = fmt.Errorf("getHash(%d) invoked, blockhash for that block not provided", num)
		}
		return h
	}
	blockCtx := &vm.BlockContext{
		CanTransfer: core.CanTransfer,
		Transfer:    core.Transfer,
		Coinbase:    inputEnv.Coinbase,
		BlockNumber: new(big.Int).SetUint64(inputEnv.Number),
		Time:        new(big.Int).SetUint64(inputEnv.Timestamp),
		Difficulty:  inputEnv.Difficulty,
		GasLimit:    inputEnv.GasLimit,
		GetHash:     getHash,
	}
	// If currentBaseFee is defined, add it to the vmContext.
	if inputEnv.BaseFee != nil {
		blockCtx.BaseFee = new(big.Int).Set(inputEnv.BaseFee)
	}
	return blockCtx
}

// compileVMResult creates a result of a transaction as SubstateResult struct.
func compileVMResult(logs []*types.Log, reciept *evmcore.ExecutionResult, contract common.Address) *substate.SubstateResult {
	vmResult := &substate.SubstateResult{
		ContractAddress: contract,
		GasUsed:         reciept.UsedGas,
		Logs:            logs,
		Bloom:           types.BytesToBloom(types.LogsBloom(logs)),
	}
	if reciept.Failed() {
		vmResult.Status = types.ReceiptStatusFailed
	} else {
		vmResult.Status = types.ReceiptStatusSuccessful
	}
	return vmResult
}

// validateVMResult compares the result of a transaction to an expected value.
func validateVMResult(vmResult, expectedResult *substate.SubstateResult) error {
	r := expectedResult.Equal(vmResult)
	if !r {
		log.Printf("inconsistent output: result\n")
		PrintResultDiffSummary(expectedResult, vmResult)
		return fmt.Errorf("inconsistent output")
	}
	return nil
}

// validateVMAlloc compares states of accounts in stateDB to an expected set of states.
// If fullState mode, check if expected stae is contained in stateDB.
// If partialState mode, check for equality of sets.
func validateVMAlloc(db state.StateDB, expectedAlloc substate.SubstateAlloc, mode ValidationMode) error {
	var err error
	switch mode {
	case SubsetCheck:
		err = ValidateStateDB(expectedAlloc, db, !UpdateOnFailure)
	case EqualityCheck:
		vmAlloc := db.GetSubstatePostAlloc()
		isEqual := expectedAlloc.Equal(vmAlloc)
		if !isEqual {
			err = fmt.Errorf("inconsistent output: alloc\n")
			PrintAllocationDiffSummary(&expectedAlloc, &vmAlloc)
		}
	}
	return err
}

// handleErrorOnExit reports error appropiately based on continue-on-failure option.
func handleErrorOnExit(err *error, errMsg *strings.Builder, newErrors *int, continueOnFailure bool) {
	if *newErrors > 0 {
		if continueOnFailure {
			log.Println(errMsg.String())
		} else {
			*err = fmt.Errorf(errMsg.String())
		}
	}
	NumErrors += *newErrors
	if NumErrors > MaxErrors {
		*err = fmt.Errorf("%w\nToo many errors...", *err)
	}
}

// printIfDifferent compares two values of any types and reports differences if any.
func printIfDifferent[T comparable](label string, want, have T) bool {
	if want != have {
		fmt.Printf("  Different %s:\n", label)
		fmt.Printf("    want: %v\n", want)
		fmt.Printf("    have: %v\n", have)
		return true
	}
	return false
}

// printIfDifferentBytes compares two values of byte type and reports differences if any.
func printIfDifferentBytes(label string, want, have []byte) bool {
	if !bytes.Equal(want, have) {
		fmt.Printf("  Different %s:\n", label)
		fmt.Printf("    want: %v\n", want)
		fmt.Printf("    have: %v\n", have)
		return true
	}
	return false
}

// printIfDifferentBigInt compares two values of big int type and reports differences if any.
func printIfDifferentBigInt(label string, want, have *big.Int) bool {
	if want == nil && have == nil {
		return false
	}
	if want == nil || have == nil || want.Cmp(have) != 0 {
		fmt.Printf("  Different %s:\n", label)
		fmt.Printf("    want: %v\n", want)
		fmt.Printf("    have: %v\n", have)
		return true
	}
	return false
}

// PrintResultDiffSummary compares two tx results and reports differences if any.
func PrintResultDiffSummary(want, have *substate.SubstateResult) {
	printIfDifferent("status", want.Status, have.Status)
	printIfDifferent("contract address", want.ContractAddress, have.ContractAddress)
	printIfDifferent("gas usage", want.GasUsed, have.GasUsed)
	printIfDifferent("log bloom filter", want.Bloom, have.Bloom)
	if !printIfDifferent("log size", len(want.Logs), len(have.Logs)) {
		for i := range want.Logs {
			printLogDiffSummary(fmt.Sprintf("log[%d]", i), want.Logs[i], have.Logs[i])
		}
	}
}

// printLogDiffSummary compares two tx logs and reports differences if any.
func printLogDiffSummary(label string, want, have *types.Log) {
	printIfDifferent(fmt.Sprintf("%s.address", label), want.Address, have.Address)
	if !printIfDifferent(fmt.Sprintf("%s.Topics size", label), len(want.Topics), len(have.Topics)) {
		for i := range want.Topics {
			printIfDifferent(fmt.Sprintf("%s.Topics[%d]", label, i), want.Topics[i], have.Topics[i])
		}
	}
	printIfDifferentBytes(fmt.Sprintf("%s.data", label), want.Data, have.Data)
}

// PrintAllocationDiffSummary compares atrributes and existence of accounts and reports differences if any.
func PrintAllocationDiffSummary(want, have *substate.SubstateAlloc) {
	printIfDifferent("substate alloc size", len(*want), len(*have))
	for key := range *want {
		_, present := (*have)[key]
		if !present {
			fmt.Printf("    missing key=%v\n", key)
		}
	}

	for key := range *have {
		_, present := (*want)[key]
		if !present {
			fmt.Printf("    extra key=%v\n", key)
		}
	}

	for key, is := range *have {
		should, present := (*want)[key]
		if present {
			printAccountDiffSummary(fmt.Sprintf("key=%v:", key), should, is)
		}
	}
}

// PrintAccountDiffSummary compares attributes of two accounts and reports differences if any.
func printAccountDiffSummary(label string, want, have *substate.SubstateAccount) {
	printIfDifferent(fmt.Sprintf("%s.Nonce", label), want.Nonce, have.Nonce)
	printIfDifferentBigInt(fmt.Sprintf("%s.Balance", label), want.Balance, have.Balance)
	printIfDifferentBytes(fmt.Sprintf("%s.Code", label), want.Code, have.Code)

	printIfDifferent(fmt.Sprintf("len(%s.Storage)", label), len(want.Storage), len(have.Storage))
	for key := range want.Storage {
		_, present := have.Storage[key]
		if !present {
			fmt.Printf("    %s.Storage misses key %v\n", label, key)
		}
	}

	for key := range have.Storage {
		_, present := want.Storage[key]
		if !present {
			fmt.Printf("    %s.Storage has extra key %v\n", label, key)
		}
	}

	for key, is := range have.Storage {
		should, present := want.Storage[key]
		if present {
			printIfDifferent(fmt.Sprintf("%s.Storage[%v]", label, key), should, is)
		}
	}
}
