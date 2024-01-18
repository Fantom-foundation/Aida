package executor

import (
	"errors"
	"fmt"
	"math/big"
	"sync/atomic"

	"github.com/Fantom-foundation/Aida/state"
	"github.com/Fantom-foundation/Aida/txcontext"
	substatecontext "github.com/Fantom-foundation/Aida/txcontext/substate"
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

// MakeLiveDbProcessor creates a executor.Processor which processes txcontext into LIVE StateDb.
func MakeLiveDbProcessor(cfg *utils.Config) *LiveDbProcessor {
	return &LiveDbProcessor{&SubstateProcessor{cfg: cfg}}
}

type LiveDbProcessor struct {
	*SubstateProcessor
}

// Process txcontext inside state into given LIVE StateDb
func (p *LiveDbProcessor) Process(state State[txcontext.WithValidation], ctx *Context) error {
	var err error

	err = p.ProcessTransaction(ctx.State, state.Block, state.Transaction, state.Data)
	if err == nil {
		return nil
	}

	if !p.isErrFatal() {
		ctx.ErrorInput <- fmt.Errorf("live-db processor failed; %v", err)
		return nil
	}

	return err
}

// MakeArchiveDbProcessor creates a executor.Processor which processes txcontext into ARCHIVE StateDb.
func MakeArchiveDbProcessor(cfg *utils.Config) *ArchiveDbProcessor {
	return &ArchiveDbProcessor{&SubstateProcessor{cfg: cfg}}
}

type ArchiveDbProcessor struct {
	*SubstateProcessor
}

// Process txcontext inside state into given ARCHIVE StateDb
func (p *ArchiveDbProcessor) Process(state State[txcontext.WithValidation], ctx *Context) error {
	var err error

	err = p.ProcessTransaction(ctx.Archive, state.Block, state.Transaction, state.Data)
	if err == nil {
		return nil
	}

	if !p.isErrFatal() {
		ctx.ErrorInput <- fmt.Errorf("archive-db processor failed; %v", err)
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

func (s *SubstateProcessor) ProcessTransaction(db state.VmStateDB, block int, tx int, st txcontext.WithValidation) error {
	if tx >= utils.PseudoTx {
		s.processPseudoTx(st.GetOutputState(), db)
		return nil
	}
	return s.processRegularTx(db, block, tx, st)
}

// processRegularTx executes VM on a chosen storage system.
func (s *SubstateProcessor) processRegularTx(db state.VmStateDB, block int, tx int, st txcontext.WithValidation) (finalError error) {
	db.BeginTransaction(uint32(tx))
	defer db.EndTransaction()

	if s.cfg.ChainID == utils.EthereumChainID {
		return s.ethereumTx(db, block, tx, st)
	} else {
		return s.fantomTx(db, block, tx, st)
	}
}

// fantomTx processes a txcontext in Fantom Opera EVM configuration
func (s *SubstateProcessor) fantomTx(db state.VmStateDB, block int, tx int, st txcontext.WithValidation) (finalError error) {
	var (
		gaspool   = new(evmcore.GasPool)
		txHash    = common.HexToHash(fmt.Sprintf("0x%016d%016d", block, tx))
		inputEnv  = st.GetBlockEnvironment()
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
	gaspool.AddGas(inputEnv.GetGasLimit())
	msg := st.GetMessage()
	db.Prepare(txHash, tx)
	blockCtx := prepareBlockCtx(inputEnv)
	txCtx := evmcore.NewEVMTxContext(msg)
	evm := vm.NewEVM(*blockCtx, txCtx, db, chainConfig, vmConfig)
	snapshot := db.Snapshot()

	// apply
	msgResult, err := evmcore.ApplyMessage(evm, msg, gaspool)
	if err != nil {
		// if txcontext fails, revert to the first snapshot.
		db.RevertToSnapshot(snapshot)
		finalError = errors.Join(fmt.Errorf("block: %v txcontext: %v", block, tx), err)
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
		vmResult := compileVMResult(logs, msgResult.UsedGas, msgResult.Failed(), contract)
		if err = validateVMResult(vmResult, st.GetReceipt()); err != nil {
			finalError = errors.Join(finalError, err)
		}
	}
	return
}

// ethereumTx processes a txcontext in Ethereum EVM configuration
func (s *SubstateProcessor) ethereumTx(db state.VmStateDB, block int, tx int, st txcontext.WithValidation) (finalError error) {
	var (
		gaspool  = new(core.GasPool)
		txHash   = common.HexToHash(fmt.Sprintf("0x%016d%016d", block, tx))
		inputEnv = st.GetBlockEnvironment()
		validate = s.cfg.ValidateTxState
	)

	// create vm config
	var vmConfig vm.Config
	vmConfig.InterpreterImpl = s.cfg.VmImpl
	//vmConfig.NoBaseFee = true
	vmConfig.Tracer = nil
	vmConfig.Debug = false

	chainConfig := utils.GetChainConfig(s.cfg.ChainID)

	// prepare tx
	gaspool.AddGas(inputEnv.GetGasLimit())
	msg := st.GetMessage()
	db.Prepare(txHash, tx)
	blockCtx := prepareBlockCtx(inputEnv)
	txCtx := core.NewEVMTxContext(msg)
	evm := vm.NewEVM(*blockCtx, txCtx, db, chainConfig, vmConfig)
	snapshot := db.Snapshot()

	// apply
	msgResult, err := core.ApplyMessage(evm, msg, gaspool)
	if err != nil {
		// if txcontext fails, revert to the first snapshot.
		db.RevertToSnapshot(snapshot)
		finalError = errors.Join(fmt.Errorf("block: %v txcontext: %v", block, tx), err)
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
		vmResult := compileVMResult(logs, msgResult.UsedGas, msgResult.Failed(), contract)
		if err = validateVMResult(vmResult, st.GetReceipt()); err != nil {
			finalError = errors.Join(finalError, err)
		}
	}
	return
}

// processPseudoTx processes pseudo transactions in Lachesis by applying the change in db state.
// The pseudo transactions includes Lachesis SFC, lachesis genesis and lachesis-opera transition.
func (s *SubstateProcessor) processPseudoTx(ws txcontext.WorldState, db state.VmStateDB) {
	db.BeginTransaction(utils.PseudoTx)
	defer db.EndTransaction()

	ws.ForEachAccount(func(addr common.Address, acc txcontext.Account) {
		db.SubBalance(addr, db.GetBalance(addr))
		db.AddBalance(addr, acc.GetBalance())
		db.SetNonce(addr, acc.GetNonce())
		db.SetCode(addr, acc.GetCode())
		acc.ForEachStorage(func(keyHash common.Hash, valueHash common.Hash) {
			db.SetState(addr, keyHash, valueHash)
		})
	})

}

// prepareBlockCtx creates a block context for evm call from an environment of a substate.
func prepareBlockCtx(inputEnv txcontext.BlockEnvironment) *vm.BlockContext {
	blockCtx := &vm.BlockContext{
		CanTransfer: core.CanTransfer,
		Transfer:    core.Transfer,
		Coinbase:    inputEnv.GetCoinbase(),
		BlockNumber: new(big.Int).SetUint64(inputEnv.GetNumber()),
		Time:        new(big.Int).SetUint64(inputEnv.GetTimestamp()),
		Difficulty:  inputEnv.GetDifficulty(),
		GasLimit:    inputEnv.GetGasLimit(),
		GetHash:     inputEnv.GetBlockHash,
	}
	// If currentBaseFee is defined, add it to the vmContext.
	baseFee := inputEnv.GetBaseFee()
	if baseFee != nil {
		blockCtx.BaseFee = new(big.Int).Set(baseFee)
	}
	return blockCtx
}

// compileVMResult creates a result of a txcontext as SubstateResult struct.
func compileVMResult(logs []*types.Log, recieptUsedGas uint64, recieptFailed bool, contract common.Address) txcontext.Receipt {
	vmResult := &substate.SubstateResult{
		ContractAddress: contract,
		GasUsed:         recieptUsedGas,
		Logs:            logs,
		Bloom:           types.BytesToBloom(types.LogsBloom(logs)),
	}
	if recieptFailed {
		vmResult.Status = types.ReceiptStatusFailed
	} else {
		vmResult.Status = types.ReceiptStatusSuccessful
	}
	return substatecontext.NewReceipt(vmResult)
}

// validateVMResult compares the result of a txcontext to an expected value.
func validateVMResult(vmResult, expectedResult txcontext.Receipt) error {
	if !expectedResult.Equal(vmResult) {
		return fmt.Errorf("inconsistent output\n"+
			"\ngot:\n"+
			"\tstatus: %v\n"+
			"\tbloom: %v\n"+
			"\tlogs: %v\n"+
			"\tcontract address: %v\n"+
			"\tgas used: %v\n"+
			"\nwant:\n"+
			"\tstatus: %v\n"+
			"\tbloom: %v\n"+
			"\tlogs: %v\n"+
			"\tcontract address: %v\n"+
			"\tgas used: %v\n",
			vmResult.GetStatus(),
			vmResult.GetBloom().Big().Uint64(),
			vmResult.GetLogs(),
			vmResult.GetContractAddress(),
			vmResult.GetGasUsed(),
			expectedResult.GetStatus(),
			expectedResult.GetBloom().Big().Uint64(),
			expectedResult.GetLogs(),
			expectedResult.GetContractAddress(),
			expectedResult.GetGasUsed(),
		)
	}
	return nil
}
