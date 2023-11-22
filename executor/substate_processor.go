package executor

import (
	"errors"
	"fmt"
	"math/big"
	"sync/atomic"

	"github.com/Fantom-foundation/Aida/state"
	"github.com/Fantom-foundation/Aida/utils"
	substate "github.com/Fantom-foundation/Substate"
	"github.com/Fantom-foundation/go-opera/evmcore"
	"github.com/Fantom-foundation/go-opera/opera"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
)

func MakeLiveDbProcessor(cfg *utils.Config) *LiveDbProcessor {
	return &LiveDbProcessor{&SubstateProcessor{cfg: cfg}}
}

type LiveDbProcessor struct {
	*SubstateProcessor
}

func (p *LiveDbProcessor) Process(state State[*substate.Substate], ctx *Context) error {
	var err error

	//_, err = utils.ProcessTx(ctx.State, p.cfg, uint64(state.Block), state.Transaction, state.Data)
	//return err

	err = p.ProcessTransaction(ctx.State, state.Block, state.Transaction, state.Data)
	if err == nil {
		return nil
	}

	if !p.isErrFatal() {
		ctx.ErrorInput <- err
		return nil
	}

	return err
}

func MakeArchiveDbProcessor(cfg *utils.Config) *ArchiveDbProcessor {
	return &ArchiveDbProcessor{&SubstateProcessor{cfg: cfg}}
}

type ArchiveDbProcessor struct {
	*SubstateProcessor
}

func (p *ArchiveDbProcessor) Process(state State[*substate.Substate], ctx *Context) error {
	var err error

	//_, err = utils.ProcessTx(ctx.State, p.cfg, uint64(state.Block), state.Transaction, state.Data)
	//return err

	err = p.ProcessTransaction(ctx.Archive, state.Block, state.Transaction, state.Data)
	if err == nil {
		return nil
	}

	if !p.isErrFatal() {
		ctx.ErrorInput <- err
		return nil
	}

	return err
}

type SubstateProcessor struct {
	cfg       *utils.Config
	numErrors *atomic.Int32 // transactions can be processed in parallel, so this needs to be thread safe

}

func MakeSubstateProcessor(cfg *utils.Config) *SubstateProcessor {
	return &SubstateProcessor{cfg: cfg}
}

func (s *SubstateProcessor) isErrFatal() bool {
	if !s.cfg.ContinueOnFailure {
		return true
	}

	// check this first, so we don't have to access atomic value
	if s.cfg.MaxNumErrors <= 0 {
		return false
	}

	if s.numErrors.Load() < int32(s.cfg.MaxNumErrors) {
		s.numErrors.Add(1)
		return false
	}

	return true
}

func (s *SubstateProcessor) ProcessTransaction(db state.VmStateDB, block int, tx int, st *substate.Substate) error {
	if tx >= utils.PseudoTx {
		s.processPseudoTx(st.OutputAlloc, db)
		return nil
	}
	return s.processRegularTx(db, block, tx, st)
}

// processRegularTx executes VM on a chosen storage system.
func (s *SubstateProcessor) processRegularTx(db state.VmStateDB, block int, tx int, st *substate.Substate) (finalError error) {
	db.BeginTransaction(uint32(tx))
	defer db.EndTransaction()

	var (
		gaspool   = new(evmcore.GasPool)
		txHash    = common.HexToHash(fmt.Sprintf("0x%016d%016d", block, tx))
		inputEnv  = st.Env
		hashError error
		validate  = s.cfg.ValidateTxState
	)

	// create vm config
	vmConfig := opera.DefaultVMConfig
	vmConfig.InterpreterImpl = s.cfg.VmImpl
	vmConfig.NoBaseFee = true
	vmConfig.Tracer = nil
	vmConfig.Debug = false

	chainConfig := utils.GetChainConfig(s.cfg.ChainID)

	// prepare tx
	gaspool.AddGas(inputEnv.GasLimit)
	msg := st.Message.AsMessage()
	db.Prepare(txHash, tx)
	blockCtx := prepareBlockCtx(inputEnv, &hashError)
	txCtx := evmcore.NewEVMTxContext(msg)
	evm := vm.NewEVM(*blockCtx, txCtx, db, chainConfig, vmConfig)
	snapshot := db.Snapshot()

	// apply
	msgResult, err := evmcore.ApplyMessage(evm, msg, gaspool)
	if err != nil {
		// if transaction fails, revert to the first snapshot.
		db.RevertToSnapshot(snapshot)
		finalError = errors.Join(fmt.Errorf("block: %v transaction: %v", block, tx), err)
		// discontinue output alloc validation on error
		validate = false
	}

	// check whether getHash func produced an error
	if hashError != nil {
		finalError = errors.Join(finalError, hashError)
		// discontinue output alloc validation on error
		validate = false
	}

	// check whether the outputAlloc substate is contained in the world-state db.
	// todo this should be move to extension
	if validate {
		blockHash := common.HexToHash(fmt.Sprintf("0x%016d", block))

		// validate result
		logs := db.GetLogs(txHash, blockHash)
		var contract common.Address
		if to := msg.To(); to == nil {
			contract = crypto.CreateAddress(evm.TxContext.Origin, msg.Nonce())
		}
		vmResult := compileVMResult(logs, msgResult, contract)
		if err = validateVMResult(vmResult, st.Result); err != nil {
			finalError = errors.Join(finalError, err)
		}
	}

	return
}

// processPseudoTx processes pseudo transactions in Lachesis by applying the change in db state.
// The pseudo transactions includes Lachesis SFC, lachesis genesis and lachesis-opera transition.
func (s *SubstateProcessor) processPseudoTx(sa substate.SubstateAlloc, db state.VmStateDB) {
	db.BeginTransaction(utils.PseudoTx)
	defer db.EndTransaction()

	for addr, account := range sa {
		db.SubBalance(addr, db.GetBalance(addr))
		db.AddBalance(addr, account.Balance)
		db.SetNonce(addr, account.Nonce)
		db.SetCode(addr, account.Code)
		for key, value := range account.Storage {
			db.SetState(addr, key, value)
		}
	}
}

// prepareBlockCtx creates a block context for evm call from an environment of a substate.
func prepareBlockCtx(inputEnv *substate.SubstateEnv, hashError *error) *vm.BlockContext {
	getHash := func(num uint64) common.Hash {
		if inputEnv.BlockHashes == nil {
			*hashError = fmt.Errorf("getHash(%d) invoked, no blockhashes provided", num)
			return common.Hash{}
		}
		h, ok := inputEnv.BlockHashes[num]
		if !ok {
			*hashError = fmt.Errorf("getHash(%d) invoked, blockhash for that block not provided", num)
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
	if !expectedResult.Equal(vmResult) {
		return fmt.Errorf("inconsistent output; %v", vmResult)
	}
	return nil
}
