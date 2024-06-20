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

package proxy

import (
	"fmt"
	"time"

	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/state"
	"github.com/Fantom-foundation/Aida/tracer/operation"
	"github.com/Fantom-foundation/Aida/txcontext"
	"github.com/Fantom-foundation/Aida/utils/analytics"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
)

// ProfilerProxy data structure for capturing and recording
// invoked StateDB operations.
type ProfilerProxy struct {
	db   state.StateDB // state db
	anlt *analytics.IncrementalAnalytics
	log  logger.Logger
}

// NewProfilerProxy creates a new StateDB profiler.
func NewProfilerProxy(db state.StateDB, anlt *analytics.IncrementalAnalytics, logLevel string) *ProfilerProxy {
	p := new(ProfilerProxy)
	p.db = db
	p.anlt = anlt
	p.log = logger.NewLogger(logLevel, "Proxy Profiler")
	return p
}

// CreateAccount creates a new account.
func (p *ProfilerProxy) CreateAccount(addr common.Address) {
	p.do(operation.CreateAccountID, func() {
		p.db.CreateAccount(addr)
	})
}

// SubBalance subtracts amount from a contract address.
func (p *ProfilerProxy) SubBalance(addr common.Address, amount *uint256.Int, reason tracing.BalanceChangeReason) {
	p.do(operation.SubBalanceID, func() {
		p.db.SubBalance(addr, amount, reason)
	})
}

// AddBalance adds amount to a contract address.
func (p *ProfilerProxy) AddBalance(addr common.Address, amount *uint256.Int, reason tracing.BalanceChangeReason) {
	p.do(operation.AddBalanceID, func() {
		p.db.AddBalance(addr, amount, reason)
	})
}

// GetBalance retrieves the amount of a contract address.
func (p *ProfilerProxy) GetBalance(addr common.Address) *uint256.Int {
	var res *uint256.Int
	p.do(operation.GetBalanceID, func() {
		res = p.db.GetBalance(addr)
	})
	return res
}

// GetNonce retrieves the nonce of a contract address.
func (p *ProfilerProxy) GetNonce(addr common.Address) uint64 {
	var res uint64
	p.do(operation.GetNonceID, func() {
		res = p.db.GetNonce(addr)
	})
	return res
}

// SetNonce sets the nonce of a contract address.
func (p *ProfilerProxy) SetNonce(addr common.Address, nonce uint64) {
	p.do(operation.SetNonceID, func() {
		p.db.SetNonce(addr, nonce)
	})
}

// GetCodeHash returns the hash of the EVM bytecode.
func (p *ProfilerProxy) GetCodeHash(addr common.Address) common.Hash {
	var res common.Hash
	p.do(operation.GetCodeHashID, func() {
		res = p.db.GetCodeHash(addr)
	})
	return res
}

// GetCode returns the EVM bytecode of a contract.
func (p *ProfilerProxy) GetCode(addr common.Address) []byte {
	var res []byte
	p.do(operation.GetCodeID, func() {
		res = p.db.GetCode(addr)
	})
	return res
}

// SetCode sets the EVM bytecode of a contract.
func (p *ProfilerProxy) SetCode(addr common.Address, code []byte) {
	p.do(operation.SetCodeID, func() {
		p.db.SetCode(addr, code)
	})
}

// GetCodeSize returns the EVM bytecode's size.
func (p *ProfilerProxy) GetCodeSize(addr common.Address) int {
	var res int
	p.do(operation.GetCodeSizeID, func() {
		res = p.db.GetCodeSize(addr)
	})
	return res
}

// AddRefund adds gas to the refund counter.
func (p *ProfilerProxy) AddRefund(gas uint64) {
	p.do(operation.AddRefundID, func() {
		p.db.AddRefund(gas)
	})
}

// SubRefund subtracts gas to the refund counter.
func (p *ProfilerProxy) SubRefund(gas uint64) {
	p.do(operation.SubRefundID, func() {
		p.db.SubRefund(gas)
	})
}

// GetRefund returns the current value of the refund counter.
func (p *ProfilerProxy) GetRefund() uint64 {
	var res uint64
	p.do(operation.GetRefundID, func() {
		res = p.db.GetRefund()
	})
	return res
}

// GetCommittedState retrieves a value that is already committed.
func (p *ProfilerProxy) GetCommittedState(addr common.Address, key common.Hash) common.Hash {
	var res common.Hash
	p.do(operation.GetCommittedStateID, func() {
		res = p.db.GetCommittedState(addr, key)
	})
	return res
}

// GetState retrieves a value from the StateDB.
func (p *ProfilerProxy) GetState(addr common.Address, key common.Hash) common.Hash {
	var res common.Hash
	p.do(operation.GetStateID, func() {
		res = p.db.GetState(addr, key)
	})
	return res
}

// SetState sets a value in the StateDB.
func (p *ProfilerProxy) SetState(addr common.Address, key common.Hash, value common.Hash) {
	p.do(operation.SetStateID, func() {
		p.db.SetState(addr, key, value)
	})
}

// SelfDestruct marks the given account as suicided. This clears the account balance.
// The account is still available until the state is committed;
// return a non-nil account after SelfDestruct.
func (p *ProfilerProxy) SelfDestruct(addr common.Address) {
	p.do(operation.SuicideID, func() {
		p.db.SelfDestruct(addr)
	})
}

// HasSelfDestructed checks whether a contract has been suicided.
func (p *ProfilerProxy) HasSelfDestructed(addr common.Address) bool {
	var res bool
	p.do(operation.HasSuicidedID, func() {
		res = p.db.HasSelfDestructed(addr)
	})
	return res
}

// Exist checks whether the contract exists in the StateDB.
// Notably this also returns true for suicided accounts.
func (p *ProfilerProxy) Exist(addr common.Address) bool {
	var res bool
	p.do(operation.ExistID, func() {
		res = p.db.Exist(addr)
	})
	return res
}

// Empty checks whether the contract is either non-existent
// or empty according to the EIP161 specification (balance = nonce = code = 0).
func (p *ProfilerProxy) Empty(addr common.Address) bool {
	var empty bool
	p.do(operation.EmptyID, func() {
		empty = p.db.Empty(addr)
	})
	return empty
}

// Prepare handles the preparatory steps for executing a state transition with
// regards to both EIP-2929 and EIP-2930:
//
// - Add sender to access list (2929)
// - Add destination to access list (2929)
// - Add precompiles to access list (2929)
// - Add the contents of the optional tx access list (2930)
//
// This method should only be called if Berlin/2929+2930 is applicable at the current number.
func (p *ProfilerProxy) Prepare(rules params.Rules, sender, coinbase common.Address, dest *common.Address, precompiles []common.Address, txAccesses types.AccessList) {
	p.do(operation.PrepareAccessListID, func() {
		p.db.Prepare(rules, sender, coinbase, dest, precompiles, txAccesses)
	})
}

// AddAddressToAccessList adds an address to the access list.
func (p *ProfilerProxy) AddAddressToAccessList(addr common.Address) {
	p.do(operation.AddAddressToAccessListID, func() {
		p.db.AddAddressToAccessList(addr)
	})
}

// AddressInAccessList checks whether an address is in the access list.
func (p *ProfilerProxy) AddressInAccessList(addr common.Address) bool {
	var res bool
	p.do(operation.AddressInAccessListID, func() {
		res = p.db.AddressInAccessList(addr)
	})
	return res
}

// SlotInAccessList checks whether the (address, slot)-tuple is in the access list.
func (p *ProfilerProxy) SlotInAccessList(addr common.Address, slot common.Hash) (bool, bool) {
	var addressOk, slotOk bool
	p.do(operation.SlotInAccessListID, func() {
		addressOk, slotOk = p.db.SlotInAccessList(addr, slot)
	})
	return addressOk, slotOk
}

// AddSlotToAccessList adds the given (address, slot)-tuple to the access list
func (p *ProfilerProxy) AddSlotToAccessList(addr common.Address, slot common.Hash) {
	p.do(operation.AddSlotToAccessListID, func() {
		p.db.AddSlotToAccessList(addr, slot)
	})
}

// Snapshot returns an identifier for the current revision of the state.
func (p *ProfilerProxy) Snapshot() int {
	var res int
	p.do(operation.SnapshotID, func() {
		res = p.db.Snapshot()
	})
	return res
}

// RevertToSnapshot reverts all state changes from a given revision.
func (p *ProfilerProxy) RevertToSnapshot(snapshot int) {
	p.do(operation.RevertToSnapshotID, func() {
		p.db.RevertToSnapshot(snapshot)
	})
}

func (p *ProfilerProxy) Error() error {
	return p.db.Error()
}

func (p *ProfilerProxy) do(opId byte, op func()) {
	start := time.Now()
	op()
	elapsed := time.Since(start)
	p.anlt.Update(opId, float64(elapsed))
}

func (p *ProfilerProxy) BeginTransaction(number uint32) error {
	var err error
	p.do(operation.BeginTransactionID, func() {
		err = p.db.BeginTransaction(number)
	})
	return err
}

func (p *ProfilerProxy) EndTransaction() error {
	var err error
	p.do(operation.EndTransactionID, func() {
		err = p.db.EndTransaction()
	})
	return err
}

func (p *ProfilerProxy) BeginBlock(number uint64) error {
	var err error
	p.do(operation.BeginBlockID, func() {
		err = p.db.BeginBlock(number)
	})
	return err
}

func (p *ProfilerProxy) EndBlock() error {
	var err error
	p.do(operation.EndBlockID, func() {
		err = p.db.EndBlock()
	})
	return err
}

func (p *ProfilerProxy) BeginSyncPeriod(number uint64) {
	p.do(operation.BeginSyncPeriodID, func() {
		p.db.BeginSyncPeriod(number)
	})
}

func (p *ProfilerProxy) EndSyncPeriod() {
	p.do(operation.EndSyncPeriodID, func() {
		p.db.EndSyncPeriod()
	})
}

func (p *ProfilerProxy) GetHash() (common.Hash, error) {
	// TODO: add profiling for this operation
	return p.db.GetHash()
}

// AddLog adds a log entry.
func (p *ProfilerProxy) AddLog(log *types.Log) {
	p.do(operation.AddLogID, func() {
		p.db.AddLog(log)
	})
}

// GetLogs retrieves log entries.
func (p *ProfilerProxy) GetLogs(hash common.Hash, block uint64, blockHash common.Hash) []*types.Log {
	var logs []*types.Log
	p.do(operation.GetLogsID, func() {
		logs = p.db.GetLogs(hash, block, blockHash)
	})
	return logs
}

// AddPreimage adds a SHA3 preimage.
func (p *ProfilerProxy) AddPreimage(addr common.Hash, image []byte) {
	p.do(operation.AddPreimageID, func() {
		p.db.AddPreimage(addr, image)
	})
}

// Prepare sets the current transaction hash and index.
func (p *ProfilerProxy) SetTxContext(thash common.Hash, ti int) {
	p.do(operation.PrepareID, func() {
		p.db.SetTxContext(thash, ti)
	})
}

// Finalise the state in StateDB.
func (p *ProfilerProxy) Finalise(deleteEmptyObjects bool) {
	p.do(operation.FinaliseID, func() {
		p.db.Finalise(deleteEmptyObjects)
	})
}

// IntermediateRoot computes the current hash of the StateDB.
// It is called in between transactions to get the root hash that
// goes into transaction receipts.
func (p *ProfilerProxy) IntermediateRoot(deleteEmptyObjects bool) common.Hash {
	var hash common.Hash
	p.do(operation.IntermediateRootID, func() {
		hash = p.db.IntermediateRoot(deleteEmptyObjects)
	})
	return hash
}

func (p *ProfilerProxy) Commit(block uint64, deleteEmptyObjects bool) (common.Hash, error) {
	var hash common.Hash
	var err error
	p.do(operation.CommitID, func() {
		hash, err = p.db.Commit(block, deleteEmptyObjects)
	})
	return hash, err
}

// GetSubstatePostAlloc gets substate post allocation.
func (p *ProfilerProxy) GetSubstatePostAlloc() txcontext.WorldState {
	return p.db.GetSubstatePostAlloc()
}

func (p *ProfilerProxy) PrepareSubstate(substate txcontext.WorldState, block uint64) {
	p.db.PrepareSubstate(substate, block)
}

func (p *ProfilerProxy) Close() error {
	var err error
	p.do(operation.CloseID, func() {
		err = p.db.Close()
	})
	return err
}

func (p *ProfilerProxy) StartBulkLoad(block uint64) (state.BulkLoad, error) {
	p.log.Fatal("StartBulkLoad not supported by ProfilerProxy")
	return nil, nil
}

func (p *ProfilerProxy) GetArchiveState(block uint64) (state.NonCommittableStateDB, error) {
	return nil, fmt.Errorf("archive states are not (yet) supported by the profiling profiler")
}

func (p *ProfilerProxy) GetArchiveBlockHeight() (uint64, bool, error) {
	return 0, false, fmt.Errorf("archive states are not (yet) supported by the profiling profiler")
}

func (p *ProfilerProxy) GetMemoryUsage() *state.MemoryUsage {
	return p.db.GetMemoryUsage()
}

func (p *ProfilerProxy) GetShadowDB() state.StateDB {
	return p.db.GetShadowDB()
}

// TODO profile new operations
func (p *ProfilerProxy) CreateContract(addr common.Address) {
	p.db.CreateContract(addr)
}

func (p *ProfilerProxy) Selfdestruct6780(addr common.Address) {
	p.db.Selfdestruct6780(addr)
}

func (p *ProfilerProxy) GetStorageRoot(addr common.Address) common.Hash {
	return p.db.GetStorageRoot(addr)
}

func (p *ProfilerProxy) SetTransientState(addr common.Address, key common.Hash, value common.Hash) {
	p.db.SetTransientState(addr, key, value)
}

func (p *ProfilerProxy) GetTransientState(addr common.Address, key common.Hash) common.Hash {
	return p.db.GetTransientState(addr, key)
}
