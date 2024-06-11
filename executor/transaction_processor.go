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

package executor

import (
	"errors"
	"fmt"
	"math/big"
	"strings"
	"sync/atomic"

	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/state"
	"github.com/Fantom-foundation/Aida/txcontext"
	"github.com/Fantom-foundation/Aida/utils"
	tosca "github.com/Fantom-foundation/Tosca/go/vm"
	tosca_geth "github.com/Fantom-foundation/Tosca/go/vm/geth"
	"github.com/Fantom-foundation/go-opera/evmcore"
	"github.com/Fantom-foundation/go-opera/opera"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
)

// MakeLiveDbTxProcessor creates a executor.Processor which processes transaction into LIVE StateDb.
func MakeLiveDbTxProcessor(cfg *utils.Config) (*LiveDbTxProcessor, error) {
	processor, err := MakeTxProcessor(cfg)
	if err != nil {
		return nil, err
	}
	return &LiveDbTxProcessor{processor}, nil
}

type LiveDbTxProcessor struct {
	*TxProcessor
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
func MakeArchiveDbTxProcessor(cfg *utils.Config) (*ArchiveDbTxProcessor, error) {
	processor, err := MakeTxProcessor(cfg)
	if err != nil {
		return nil, err
	}
	return &ArchiveDbTxProcessor{processor}, nil
}

type ArchiveDbTxProcessor struct {
	*TxProcessor
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

type TxProcessor struct {
	cfg       *utils.Config
	numErrors *atomic.Int32 // transactions can be processed in parallel, so this needs to be thread safe
	log       logger.Logger
	processor processor
}

func MakeTxProcessor(cfg *utils.Config) (*TxProcessor, error) {
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

	var processor processor
	switch strings.ToLower(cfg.EvmImpl) {
	case "opera":
		processor = &operaProcessor{
			vmCfg:    vmCfg,
			chainCfg: utils.GetChainConfig(cfg.ChainID),
			log:      logger.NewLogger(cfg.LogLevel, "OperaProcessor"),
		}
	case "tosca":
		processor = &toscaProcessor{
			vmImpl:   cfg.VmImpl,
			chainCfg: utils.GetChainConfig(cfg.ChainID),
			log:      logger.NewLogger(cfg.LogLevel, "ToscaProcessor"),
		}
	default:
		return nil, fmt.Errorf("unknown EVM implementation: %s", cfg.EvmImpl)
	}

	return &TxProcessor{
		cfg:       cfg,
		numErrors: new(atomic.Int32),
		log:       logger.NewLogger(cfg.LogLevel, "TxProcessor"),
		processor: processor,
	}, nil
}

func (s *TxProcessor) isErrFatal() bool {
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

func (s *TxProcessor) ProcessTransaction(db state.VmStateDB, block int, tx int, st txcontext.TxContext) (txcontext.Result, error) {
	if tx >= utils.PseudoTx {
		return s.processPseudoTx(st.GetOutputState(), db), nil
	}
	return s.processor.processRegularTx(db, block, tx, st)
}

type processor interface {
	processRegularTx(db state.VmStateDB, block int, tx int, st txcontext.TxContext) (transactionResult, error)
}

type operaProcessor struct {
	vmCfg    vm.Config
	chainCfg *params.ChainConfig
	log      logger.Logger
}

// processRegularTx executes VM on a chosen storage system.
func (s *operaProcessor) processRegularTx(db state.VmStateDB, block int, tx int, st txcontext.TxContext) (res transactionResult, finalError error) {
	var (
		gasPool   = new(evmcore.GasPool)
		txHash    = common.HexToHash(fmt.Sprintf("0x%016d%016d", block, tx))
		inputEnv  = st.GetBlockEnvironment()
		msg       = st.GetMessage()
		hashError error
	)

	// prepare tx
	gasPool.AddGas(inputEnv.GetGasLimit())

	db.SetTxContext(txHash, tx)
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
	res = newTransactionResult(db.GetLogs(txHash, uint64(block), blockHash), msg, msgResult, err, evm.TxContext.Origin)
	return
}

// processPseudoTx processes pseudo transactions in Lachesis by applying the change in db state.
// The pseudo transactions includes Lachesis SFC, lachesis genesis and lachesis-opera transition.
func (s *TxProcessor) processPseudoTx(ws txcontext.WorldState, db state.VmStateDB) txcontext.Result {
	ws.ForEachAccount(func(addr common.Address, acc txcontext.Account) {
		db.SubBalance(addr, db.GetBalance(addr), tracing.BalanceChangeUnspecified)
		db.AddBalance(addr, acc.GetBalance(), tracing.BalanceChangeUnspecified)
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
		Time:        inputEnv.GetTimestamp(),
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

type toscaProcessor struct {
	vmImpl   string
	chainCfg *params.ChainConfig
	log      logger.Logger
}

func bigToValue(value *big.Int) tosca.Value {
	var res tosca.Value
	value.FillBytes(res[:])
	return res
}

func (t *toscaProcessor) processRegularTx(db state.VmStateDB, block int, tx int, st txcontext.TxContext) (res transactionResult, finalError error) {

	// TODO: use the registry to pick the desired EVM implementation
	processor := tosca_geth.NewProcessorWithVm(t.vmImpl)

	blockInfo := tosca.BlockInfo{
		BlockNumber: int64(block),
		Timestamp:   int64(st.GetBlockEnvironment().GetTimestamp()),
		/*
			GasPrice: 0,
			Coinbase:    Address,
			GasLimit:    Gas,
			PrevRandao:  Hash,
			ChainID:     Word,
			BaseFee:     Value,
			BlobBaseFee: Value,
		*/
		Revision: tosca.R07_Istanbul,
	}

	transaction := tosca.Transaction{
		Sender: tosca.Address(st.GetMessage().From()),
		Recipient: func() *tosca.Address {
			addr := st.GetMessage().To()
			if addr == nil {
				return nil
			}
			toscaAddr := tosca.Address(*addr)
			return &toscaAddr
		}(),
		Nonce:    st.GetMessage().Nonce(),
		Input:    st.GetMessage().Data(),
		Value:    bigToValue(st.GetMessage().Value()),
		GasLimit: tosca.Gas(st.GetMessage().Gas()),
		/*
			AccessList []AccessTuple
		*/
	}

	transactionContext := tosca.TransactionContext{
		BlockInfo: blockInfo,
		Origin:    tosca.Address(st.GetMessage().From()),
	}

	state := &stateAdapter{
		transactionContext: transactionContext,
		db:                 db,
	}

	fmt.Printf("running block %d / tx %d\n", block, tx)
	receipt, err := processor.Run(blockInfo, transaction, state)
	if err != nil {
		panic(err)
		return transactionResult{}, err
	}

	log := []*types.Log{}
	for _, l := range receipt.Logs {
		topics := make([]common.Hash, len(l.Topics))
		for i, t := range l.Topics {
			topics[i] = common.Hash(t)
		}
		log = append(log, &types.Log{
			Address: common.Address(l.Address),
			Topics:  topics,
			Data:    l.Data,
		})
	}
	msg := st.GetMessage()

	if !receipt.Success {
		// The actual error is not relevant. Anything
		// that is not equal to nil will be considered
		// as a failed execution that got rolled back.
		err = fmt.Errorf("transaction failed")
	}

	result := &evmcore.ExecutionResult{
		UsedGas:    uint64(receipt.GasUsed),
		Err:        err,
		ReturnData: receipt.Output,
	}

	return newTransactionResult(log, msg, result, nil, msg.From()), nil
}

type stateAdapter struct {
	transactionContext tosca.TransactionContext
	db                 state.VmStateDB
}

func (a *stateAdapter) AccountExists(addr tosca.Address) bool {
	return a.db.Exist(common.Address(addr))
}

func (a *stateAdapter) GetBalance(addr tosca.Address) tosca.Value {
	return bigToValue(a.db.GetBalance(common.Address(addr)))
}

func (a *stateAdapter) SetBalance(addr tosca.Address, balance tosca.Value) {
	want := balance.ToBig()
	account := common.Address(addr)
	cur := a.db.GetBalance(account)
	diff := new(big.Int).Sub(want, cur)
	a.db.AddBalance(account, diff)
}

func (a *stateAdapter) GetNonce(addr tosca.Address) uint64 {
	return a.db.GetNonce(common.Address(addr))
}

func (a *stateAdapter) SetNonce(addr tosca.Address, nonce uint64) {
	a.db.SetNonce(common.Address(addr), nonce)
}

func (a *stateAdapter) GetCodeSize(addr tosca.Address) int {
	return a.db.GetCodeSize(common.Address(addr))
}

func (a *stateAdapter) GetCodeHash(addr tosca.Address) tosca.Hash {
	return tosca.Hash(a.db.GetCodeHash(common.Address(addr)))
}

func (a *stateAdapter) GetCode(addr tosca.Address) []byte {
	return a.db.GetCode(common.Address(addr))
}

func (a *stateAdapter) SetCode(addr tosca.Address, code []byte) {
	a.db.SetCode(common.Address(addr), code)
}

func (a *stateAdapter) GetStorage(addr tosca.Address, key tosca.Key) tosca.Word {
	return tosca.Word(a.db.GetState(common.Address(addr), common.Hash(key)))
}

func (a *stateAdapter) GetCommittedStorage(addr tosca.Address, key tosca.Key) tosca.Word {
	return tosca.Word(a.db.GetCommittedState(common.Address(addr), common.Hash(key)))
}

func (a *stateAdapter) SetStorage(addr tosca.Address, key tosca.Key, value tosca.Word) tosca.StorageStatus {
	original := a.GetCommittedStorage(addr, key)
	current := a.GetStorage(addr, key)
	a.db.SetState(common.Address(addr), common.Hash(key), common.Hash(value))
	return tosca.GetStorageStatus(original, current, value)
}

func (a *stateAdapter) GetTransactionContext() tosca.TransactionContext {
	return a.transactionContext
}

func (a *stateAdapter) GetBlockHash(number int64) tosca.Hash {
	panic("implement me - block hash")
}

func (a *stateAdapter) EmitLog(addr tosca.Address, topics []tosca.Hash, data []byte) {
	tpcs := make([]common.Hash, len(topics))
	for i, t := range topics {
		tpcs[i] = common.Hash(t)
	}

	a.db.AddLog(&types.Log{
		Address: common.Address(addr),
		Topics:  tpcs,
		Data:    data,
	})
}

func (a *stateAdapter) GetLogs() []tosca.Log {
	res := []tosca.Log{}
	for _, l := range a.db.GetLogs(common.Hash{}, common.Hash{}) {
		topics := make([]tosca.Hash, len(l.Topics))
		for i, t := range l.Topics {
			topics[i] = tosca.Hash(t)
		}
		res = append(res, tosca.Log{
			Address: tosca.Address(l.Address),
			Topics:  topics,
			Data:    l.Data,
		})
	}
	return res
}

func (a *stateAdapter) Call(kind tosca.CallKind, parameter tosca.CallParameter) (tosca.CallResult, error) {
	panic("implement me - call")
}

func (a *stateAdapter) SelfDestruct(addr tosca.Address, beneficiary tosca.Address) bool {
	panic("implement me - self destruct")
}

func (a *stateAdapter) AccessAccount(addr tosca.Address) tosca.AccessStatus {
	res := a.IsAddressInAccessList(addr)
	a.db.AddAddressToAccessList(common.Address(addr))
	if res {
		return tosca.WarmAccess
	}
	return tosca.ColdAccess
}

func (a *stateAdapter) AccessStorage(addr tosca.Address, key tosca.Key) tosca.AccessStatus {
	_, res := a.IsSlotInAccessList(addr, key)
	a.db.AddSlotToAccessList(common.Address(addr), common.Hash(key))
	if res {
		return tosca.WarmAccess
	}
	return tosca.ColdAccess
}

func (a *stateAdapter) HasSelfDestructed(addr tosca.Address) bool {
	panic("implement me - has self destructed")
}

func (a *stateAdapter) CreateSnapshot() int {
	return a.db.Snapshot()
}

func (a *stateAdapter) RestoreSnapshot(snapshot int) {
	a.db.RevertToSnapshot(snapshot)
}

func (a *stateAdapter) IsAddressInAccessList(addr tosca.Address) bool {
	return a.db.AddressInAccessList(common.Address(addr))
}

func (a *stateAdapter) IsSlotInAccessList(addr tosca.Address, key tosca.Key) (addressPresent, slotPresent bool) {
	return a.db.SlotInAccessList(common.Address(addr), common.Hash(key))
}
