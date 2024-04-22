package executor

//go:generate mockgen -source transaction_processor.go -destination transaction_processor_mocks.go -package executor

import (
	"errors"
	"fmt"
	"math/big"
	"sync/atomic"

	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/state"
	"github.com/Fantom-foundation/Aida/txcontext"
	"github.com/Fantom-foundation/Aida/utils"
	"github.com/Fantom-foundation/go-opera/evmcore"
	"github.com/Fantom-foundation/go-opera/opera"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
)

// MakeLiveDbTxProcessor creates a executor.Processor which processes transaction into LIVE StateDb.
func MakeLiveDbTxProcessor(cfg *utils.Config) *LiveDbTxProcessor {
	return &LiveDbTxProcessor{makeTxProcessor(cfg)}
}

type LiveDbTxProcessor struct {
	*txProcessor
}

// Process transaction inside state into given LIVE StateDb
func (p *LiveDbTxProcessor) Process(state State[txcontext.TxContext], ctx *Context) error {
	var err error

	ctx.ExecutionResult, err = p.ProcessTransaction(ctx.State, state.Block, state.Transaction, state.Data)
	if err == nil {
		return nil
	}

	if !p.isErrFatal() {
		ctx.ErrorInput <- fmt.Errorf("live-db processor failed; %v", err)
		return nil
	}

	return err
}

// MakeArchiveDbTxProcessor creates a executor.Processor which processes transaction into ARCHIVE StateDb.
func MakeArchiveDbTxProcessor(cfg *utils.Config) *ArchiveDbTxProcessor {
	return &ArchiveDbTxProcessor{makeTxProcessor(cfg)}
}

type ArchiveDbTxProcessor struct {
	*txProcessor
}

// Process transaction inside state into given ARCHIVE StateDb
func (p *ArchiveDbTxProcessor) Process(state State[txcontext.TxContext], ctx *Context) error {
	var err error

	ctx.ExecutionResult, err = p.ProcessTransaction(ctx.Archive, state.Block, state.Transaction, state.Data)
	if err == nil {
		return nil
	}

	if !p.isErrFatal() {
		ctx.ErrorInput <- fmt.Errorf("archive-db processor failed; %v", err)
		return nil
	}

	return err
}

type TxProcessor interface {
	ProcessTransaction(db state.VmStateDB, block int, tx int, st txcontext.TxContext) (txcontext.Result, error)
}

type txProcessor struct {
	cfg       *utils.Config
	numErrors *atomic.Int32 // transactions can be processed in parallel, so this needs to be thread safe
	vmCfg     vm.Config
	chainCfg  *params.ChainConfig
	log       logger.Logger
}

func MakeTxProcessor(cfg *utils.Config) TxProcessor {
	return makeTxProcessor(cfg)
}

func makeTxProcessor(cfg *utils.Config) *txProcessor {
	var vmCfg vm.Config
	switch cfg.ChainID {
	case utils.EthereumChainID:
		break
	case utils.TestnetChainID:
		fallthrough
	case utils.MainnetChainID:
		vmCfg = opera.DefaultVMConfig
		vmCfg.NoBaseFee = true

	}

	vmCfg.InterpreterImpl = cfg.VmImpl
	vmCfg.Tracer = nil
	vmCfg.Debug = false

	return &txProcessor{
		cfg:       cfg,
		numErrors: new(atomic.Int32),
		vmCfg:     vmCfg,
		chainCfg:  utils.GetChainConfig(cfg.ChainID),
		log:       logger.NewLogger(cfg.LogLevel, "txProcessor"),
	}
}

func (s *txProcessor) isErrFatal() bool {
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

func (s *txProcessor) ProcessTransaction(db state.VmStateDB, block int, tx int, st txcontext.TxContext) (txcontext.Result, error) {
	if tx >= utils.PseudoTx {
		return s.processPseudoTx(st.GetOutputState(), db), nil
	}
	return s.processRegularTx(db, block, tx, st)
}

// processRegularTx executes VM on a chosen storage system.
func (s *txProcessor) processRegularTx(db state.VmStateDB, block int, tx int, st txcontext.TxContext) (res transactionResult, finalError error) {
	var (
		gasPool   = new(evmcore.GasPool)
		txHash    = common.HexToHash(fmt.Sprintf("0x%016d%016d", block, tx))
		inputEnv  = st.GetBlockEnvironment()
		msg       = st.GetMessage()
		hashError error
	)

	// prepare tx
	gasPool.AddGas(inputEnv.GetGasLimit())

	db.Prepare(txHash, tx)
	blockCtx := prepareBlockCtx(inputEnv, &hashError)
	txCtx := evmcore.NewEVMTxContext(msg)
	evm := vm.NewEVM(*blockCtx, txCtx, db, s.chainCfg, s.vmCfg)
	snapshot := db.Snapshot()

	// apply
	msgResult, err := evmcore.ApplyMessage(evm, msg, gasPool)
	if err != nil {
		// if transaction fails, revert to the first snapshot.
		db.RevertToSnapshot(snapshot)
		finalError = errors.Join(fmt.Errorf("block: %v transaction: %v", block, tx), err)
	}

	// inform about failing transaction
	if msgResult != nil && msgResult.Failed() {
		s.log.Debugf("Block: %v\nTransaction %v\n Status: Failed", block, tx)
	}

	// check whether getHash func produced an error
	if hashError != nil {
		finalError = errors.Join(finalError, hashError)
	}

	// if no prior error, create result and pass it to the data.
	blockHash := common.HexToHash(fmt.Sprintf("0x%016d", block))
	res = newTransactionResult(db.GetLogs(txHash, blockHash), msg, msgResult, err, evm.TxContext.Origin)
	return
}

// processPseudoTx processes pseudo transactions in Lachesis by applying the change in db state.
// The pseudo transactions includes Lachesis SFC, lachesis genesis and lachesis-opera transition.
func (s *txProcessor) processPseudoTx(ws txcontext.WorldState, db state.VmStateDB) txcontext.Result {
	ws.ForEachAccount(func(addr common.Address, acc txcontext.Account) {
		db.SubBalance(addr, db.GetBalance(addr))
		db.AddBalance(addr, acc.GetBalance())
		db.SetNonce(addr, acc.GetNonce())
		db.SetCode(addr, acc.GetCode())
		acc.ForEachStorage(func(keyHash common.Hash, valueHash common.Hash) {
			db.SetState(addr, keyHash, valueHash)
		})
	})
	return newPseudoExecutionResult()
}

// prepareBlockCtx creates a block context for evm call from given BlockEnvironment.
func prepareBlockCtx(inputEnv txcontext.BlockEnvironment, hashError *error) *vm.BlockContext {
	getHash := func(num uint64) common.Hash {
		var h common.Hash
		h, *hashError = inputEnv.GetBlockHash(num)
		return h
	}

	blockCtx := &vm.BlockContext{
		CanTransfer: core.CanTransfer,
		Transfer:    core.Transfer,
		Coinbase:    inputEnv.GetCoinbase(),
		BlockNumber: new(big.Int).SetUint64(inputEnv.GetNumber()),
		Time:        new(big.Int).SetUint64(inputEnv.GetTimestamp()),
		Difficulty:  inputEnv.GetDifficulty(),
		GasLimit:    inputEnv.GetGasLimit(),
		GetHash:     getHash,
	}
	// If currentBaseFee is defined, add it to the vmContext.
	baseFee := inputEnv.GetBaseFee()
	if baseFee != nil {
		blockCtx.BaseFee = new(big.Int).Set(baseFee)
	}
	return blockCtx
}
