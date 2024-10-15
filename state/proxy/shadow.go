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
	"bytes"
	"errors"
	"fmt"
	"strings"

	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/state"
	"github.com/Fantom-foundation/Aida/txcontext"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/stateless"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/trie/utils"
	"github.com/holiman/uint256"
)

// NewShadowProxy creates a StateDB instance bundling two other instances and running each
// operation on both of them, cross checking results. If the results are not equal, an error
// is logged and the result of the primary instance is returned.
func NewShadowProxy(prime, shadow state.StateDB, compareStateHash bool) state.StateDB {
	return &shadowStateDb{
		shadowVmStateDb: shadowVmStateDb{
			prime:            prime,
			shadow:           shadow,
			snapshots:        []snapshotPair{},
			err:              nil,
			compareStateHash: compareStateHash,
			log:              logger.NewLogger("shadow-db", "info"),
		},
		prime:  prime,
		shadow: shadow,
	}
}

type shadowVmStateDb struct {
	prime            state.VmStateDB
	shadow           state.VmStateDB
	snapshots        []snapshotPair
	err              error
	log              logger.Logger
	compareStateHash bool
}

type shadowNonCommittableStateDb struct {
	shadowVmStateDb
	prime  state.NonCommittableStateDB
	shadow state.NonCommittableStateDB
}

type shadowStateDb struct {
	shadowVmStateDb
	prime  state.StateDB
	shadow state.StateDB
}

type snapshotPair struct {
	prime, shadow int
}

func (s *shadowVmStateDb) CreateAccount(addr common.Address) {
	s.run("CreateAccount", func(s state.VmStateDB) error {
		s.CreateAccount(addr)
		return nil
	})
}

func (s *shadowVmStateDb) Exist(addr common.Address) bool {
	return s.getBool("Exist", func(s state.VmStateDB) bool { return s.Exist(addr) }, addr)
}

func (s *shadowVmStateDb) Empty(addr common.Address) bool {
	return s.getBool("Empty", func(s state.VmStateDB) bool { return s.Empty(addr) }, addr)
}

func (s *shadowVmStateDb) SelfDestruct(addr common.Address) {
	s.run("SelfDestruct", func(s state.VmStateDB) error {
		s.SelfDestruct(addr)
		return nil
	})
}

func (s *shadowVmStateDb) HasSelfDestructed(addr common.Address) bool {
	return s.getBool("HasSelfDestructed", func(s state.VmStateDB) bool { return s.HasSelfDestructed(addr) }, addr)
}

func (s *shadowVmStateDb) GetBalance(addr common.Address) *uint256.Int {
	return s.getUint256("GetBalance", func(s state.VmStateDB) *uint256.Int { return s.GetBalance(addr) }, addr)
}

func (s *shadowVmStateDb) AddBalance(addr common.Address, value *uint256.Int, reason tracing.BalanceChangeReason) {
	s.run("AddBalance", func(s state.VmStateDB) error {
		s.AddBalance(addr, value, reason)
		return nil
	})
}

func (s *shadowVmStateDb) SubBalance(addr common.Address, value *uint256.Int, reason tracing.BalanceChangeReason) {
	s.run("SubBalance", func(s state.VmStateDB) error {
		s.SubBalance(addr, value, reason)
		return nil
	})
}

func (s *shadowVmStateDb) GetNonce(addr common.Address) uint64 {
	return s.getUint64("GetNonce", func(s state.VmStateDB) uint64 { return s.GetNonce(addr) }, addr)
}

func (s *shadowVmStateDb) SetNonce(addr common.Address, value uint64) {
	s.run("SetNonce", func(s state.VmStateDB) error {
		s.SetNonce(addr, value)
		return nil
	})
}

func (s *shadowVmStateDb) GetCommittedState(addr common.Address, key common.Hash) common.Hash {
	// error here cannot happen
	return s.getHash("GetCommittedState", func(s state.VmStateDB) common.Hash { return s.GetCommittedState(addr, key) }, addr, key)
}

func (s *shadowVmStateDb) GetState(addr common.Address, key common.Hash) common.Hash {
	return s.getHash("GetState", func(s state.VmStateDB) common.Hash { return s.GetState(addr, key) }, addr, key)
}

func (s *shadowVmStateDb) SetState(addr common.Address, key common.Hash, value common.Hash) {
	s.err = errors.Join(s.err, s.run("SetState", func(s state.VmStateDB) error {
		s.SetState(addr, key, value)
		return nil
	}))
}

func (s *shadowVmStateDb) SetTransientState(addr common.Address, key common.Hash, value common.Hash) {
	s.err = errors.Join(s.err, s.run("SetTransientState", func(s state.VmStateDB) error {
		s.SetTransientState(addr, key, value)
		return nil
	}))
}

func (s *shadowVmStateDb) GetTransientState(addr common.Address, key common.Hash) common.Hash {
	return s.getHash("GetTransientState", func(s state.VmStateDB) common.Hash { return s.GetTransientState(addr, key) }, addr, key)
}

func (s *shadowVmStateDb) GetCode(addr common.Address) []byte {
	return s.getBytes("GetCode", func(s state.VmStateDB) []byte { return s.GetCode(addr) }, addr)
}

func (s *shadowVmStateDb) GetCodeSize(addr common.Address) int {
	return s.getInt("GetCodeSize", func(s state.VmStateDB) int { return s.GetCodeSize(addr) }, addr)
}

func (s *shadowVmStateDb) GetCodeHash(addr common.Address) common.Hash {
	return s.getHash("GetCodeHash", func(s state.VmStateDB) common.Hash { return s.GetCodeHash(addr) }, addr)
}

func (s *shadowVmStateDb) SetCode(addr common.Address, code []byte) {
	s.run("SetCode", func(s state.VmStateDB) error {
		s.SetCode(addr, code)
		return nil
	})
}

func (s *shadowVmStateDb) Snapshot() int {
	pair := snapshotPair{
		s.prime.Snapshot(),
		s.shadow.Snapshot(),
	}
	s.snapshots = append(s.snapshots, pair)
	return len(s.snapshots) - 1
}

func (s *shadowVmStateDb) RevertToSnapshot(id int) {
	if id < 0 || len(s.snapshots) <= id {
		panic(fmt.Sprintf("invalid snapshot id: %v, max: %v", id, len(s.snapshots)))
	}
	s.prime.RevertToSnapshot(s.snapshots[id].prime)
	s.shadow.RevertToSnapshot(s.snapshots[id].shadow)
}

func (s *shadowVmStateDb) BeginTransaction(tx uint32) error {
	s.snapshots = s.snapshots[0:0]
	return s.run("BeginTransaction", func(s state.VmStateDB) error { return s.BeginTransaction(tx) })
}

func (s *shadowVmStateDb) EndTransaction() error {
	return s.run("EndTransaction", func(s state.VmStateDB) error { return s.EndTransaction() })
}

func (s *shadowStateDb) BeginBlock(blk uint64) error {
	return s.run("BeginBlock", func(s state.StateDB) error { return s.BeginBlock(blk) })
}

func (s *shadowStateDb) EndBlock() error {
	return s.run("EndBlock", func(s state.StateDB) error { return s.EndBlock() })
}

func (s *shadowStateDb) BeginSyncPeriod(number uint64) {
	s.run("BeginSyncPeriod", func(s state.StateDB) error {
		s.BeginSyncPeriod(number)
		return nil
	})
}

func (s *shadowStateDb) EndSyncPeriod() {
	s.run("EndSyncPeriod", func(s state.StateDB) error {
		s.EndSyncPeriod()
		return nil
	})
}

func (s *shadowStateDb) GetHash() (common.Hash, error) {
	if s.compareStateHash {
		return s.getHash("GetHash", func(s state.StateDB) (common.Hash, error) {
			return s.GetHash()
		})
	}
	return s.prime.GetHash()
}

func (s *shadowNonCommittableStateDb) GetHash() (common.Hash, error) {
	if s.compareStateHash {
		return s.getHash("GetHash", func(s state.NonCommittableStateDB) (common.Hash, error) {
			return s.GetHash()
		})
	}
	return s.prime.GetHash()
}

func (s *shadowStateDb) Close() error {
	return s.getError("Close", func(s state.StateDB) error { return s.Close() })
}

func (s *shadowNonCommittableStateDb) Release() error {
	s.run("Release", func(s state.NonCommittableStateDB) { s.Release() })
	return nil
}

func (s *shadowVmStateDb) AddRefund(amount uint64) {
	s.run("AddRefund", func(s state.VmStateDB) error {
		s.AddRefund(amount)
		return nil
	})
	// check that the update value is the same
	s.getUint64("AddRefund", func(s state.VmStateDB) uint64 { return s.GetRefund() })
}

func (s *shadowVmStateDb) SubRefund(amount uint64) {
	s.run("SubRefund", func(s state.VmStateDB) error {
		s.SubRefund(amount)
		return nil
	})
	// check that the update value is the same
	s.getUint64("SubRefund", func(s state.VmStateDB) uint64 { return s.GetRefund() })
}

func (s *shadowVmStateDb) GetRefund() uint64 {
	return s.getUint64("GetRefund", func(s state.VmStateDB) uint64 { return s.GetRefund() })
}

func (s *shadowVmStateDb) Prepare(rules params.Rules, sender, coinbase common.Address, dest *common.Address, precompiles []common.Address, txAccesses types.AccessList) {
	s.run("Prepare", func(s state.VmStateDB) error {
		s.Prepare(rules, sender, coinbase, dest, precompiles, txAccesses)
		return nil
	})
}

func (s *shadowVmStateDb) AddressInAccessList(addr common.Address) bool {
	return s.getBool("AddressInAccessList", func(s state.VmStateDB) bool { return s.AddressInAccessList(addr) }, addr)
}

func (s *shadowVmStateDb) SlotInAccessList(addr common.Address, slot common.Hash) (addressOk bool, slotOk bool) {
	return s.getBoolBool("SlotInAccessList", func(s state.VmStateDB) (bool, bool) { return s.SlotInAccessList(addr, slot) }, addr, slot)
}

func (s *shadowVmStateDb) AddAddressToAccessList(addr common.Address) {
	s.run("AddAddressToAccessList", func(s state.VmStateDB) error {
		s.AddAddressToAccessList(addr)
		return nil
	})
}

func (s *shadowVmStateDb) AddSlotToAccessList(addr common.Address, slot common.Hash) {
	s.run("AddSlotToAccessList", func(s state.VmStateDB) error {
		s.AddSlotToAccessList(addr, slot)
		return nil
	})
}

func (s *shadowVmStateDb) AddLog(log *types.Log) {
	s.run("AddPreimage", func(s state.VmStateDB) error {
		s.AddLog(log)
		return nil
	})
}

func (s *shadowVmStateDb) GetLogs(hash common.Hash, block uint64, blockHash common.Hash) []*types.Log {
	logsP := s.prime.GetLogs(hash, block, blockHash)
	logsS := s.shadow.GetLogs(hash, block, blockHash)

	equal := len(logsP) == len(logsS)
	if equal {
		// check bloom
		bloomP := types.BytesToBloom(types.LogsBloom(logsP))
		bloomS := types.BytesToBloom(types.LogsBloom(logsS))
		if bloomP != bloomS {
			equal = false
		}
	}
	if !equal {
		s.logIssue("GetLogs", logsP, logsS, hash, blockHash)
		s.err = fmt.Errorf("%v diverged from shadow DB", getOpcodeString("GetLogs", hash, blockHash))
	}
	return logsP
}

func (s *shadowVmStateDb) GetStorageRoot(addr common.Address) common.Hash {
	// call must be done onto both databases but result must not be compared
	_ = s.shadow.GetStorageRoot(addr)
	// prime must be returned
	return s.prime.GetStorageRoot(addr)
}

func (s *shadowVmStateDb) CreateContract(addr common.Address) {
	s.run("CreateContract", func(s state.VmStateDB) error {
		s.CreateContract(addr)
		return nil
	})
}

func (s *shadowVmStateDb) Selfdestruct6780(addr common.Address) {
	s.run("Selfdestruct6780", func(s state.VmStateDB) error {
		s.Selfdestruct6780(addr)
		return nil
	})
}

func (s *shadowVmStateDb) PointCache() *utils.PointCache {
	return s.prime.PointCache()
}

func (s *shadowVmStateDb) Witness() *stateless.Witness {
	return s.prime.Witness()
}

func (s *shadowStateDb) Finalise(deleteEmptyObjects bool) {
	s.run("Finalise", func(s state.StateDB) error {
		s.Finalise(deleteEmptyObjects)
		return nil
	})
}

func (s *shadowStateDb) IntermediateRoot(deleteEmptyObjects bool) common.Hash {
	// Do not check hashes for equivalents.
	s.shadow.IntermediateRoot(deleteEmptyObjects)
	return s.prime.IntermediateRoot(deleteEmptyObjects)
}

func (s *shadowStateDb) Commit(block uint64, deleteEmptyObjects bool) (common.Hash, error) {
	// Do not check hashes for equivalents.
	s.shadow.Commit(block, deleteEmptyObjects)
	return s.prime.Commit(block, deleteEmptyObjects)
}

// GetError returns an error then reset it.
func (s *shadowVmStateDb) Error() error {
	err := s.err
	// reset error message
	s.err = nil
	return err
}

func (s *shadowVmStateDb) SetTxContext(thash common.Hash, ti int) {
	s.run("SetTxContext", func(s state.VmStateDB) error {
		s.SetTxContext(thash, ti)
		return nil
	})
}

func (s *shadowStateDb) PrepareSubstate(substate txcontext.WorldState, block uint64) {
	s.run("PrepareSubstate", func(s state.StateDB) error {
		s.PrepareSubstate(substate, block)
		return nil
	})
}

func (s *shadowVmStateDb) GetSubstatePostAlloc() txcontext.WorldState {
	// Skip comparing those results.
	s.shadow.GetSubstatePostAlloc()
	return s.prime.GetSubstatePostAlloc()
}

func (s *shadowVmStateDb) AddPreimage(hash common.Hash, plain []byte) {
	s.run("AddPreimage", func(s state.VmStateDB) error {
		s.AddPreimage(hash, plain)
		return nil
	})
}

func (s *shadowStateDb) StartBulkLoad(block uint64) (state.BulkLoad, error) {
	pbl, err := s.prime.StartBulkLoad(block)
	if err != nil {
		return nil, fmt.Errorf("cannot start prime bulkload; %w", err)
	}
	sbl, err := s.shadow.StartBulkLoad(block)
	if err != nil {
		return nil, fmt.Errorf("cannot start shadow bulkload; %w", err)
	}
	return &shadowBulkLoad{pbl, sbl}, nil
}

func (s *shadowStateDb) GetArchiveState(block uint64) (state.NonCommittableStateDB, error) {
	var prime, shadow state.NonCommittableStateDB
	var err error
	if prime, err = s.prime.GetArchiveState(block); err != nil {
		return nil, err
	}
	if shadow, err = s.shadow.GetArchiveState(block); err != nil {
		return nil, err
	}
	return &shadowNonCommittableStateDb{
		shadowVmStateDb: shadowVmStateDb{
			prime:     prime,
			shadow:    shadow,
			snapshots: []snapshotPair{},
			err:       nil,
			log:       s.log,
		},
		prime:  prime,
		shadow: shadow,
	}, nil
}

func (s *shadowStateDb) GetArchiveBlockHeight() (uint64, bool, error) {
	// There is no strict need for both archives to be on the same level.
	// Thus, we report the minimum of the two available block heights.
	pBlock, pEmpty, pErr := s.prime.GetArchiveBlockHeight()
	sBlock, sEmpty, sErr := s.shadow.GetArchiveBlockHeight()
	if pErr != nil {
		return 0, false, pErr
	}
	if sErr != nil {
		return 0, false, sErr
	}
	if pEmpty || sEmpty {
		return 0, true, nil
	}
	min := pBlock
	if sBlock < min {
		min = sBlock
	}
	return min, false, nil
}

type stringStringer struct {
	str string
}

func (s stringStringer) String() string {
	return s.str
}

func (s *shadowStateDb) GetMemoryUsage() *state.MemoryUsage {
	var (
		breakdown strings.Builder
		usedBytes uint64 = 0
	)

	breakdown.WriteString("Primary:\n")
	resP := s.prime.GetMemoryUsage()
	if resP != nil {
		fmt.Fprintf(&breakdown, "%v\n", resP.Breakdown)
		usedBytes += resP.UsedBytes
	} else {
		breakdown.WriteString("\tMemory breakdown not supported.\n")
	}
	breakdown.WriteString("Shadow:\n")
	resS := s.shadow.GetMemoryUsage()
	if resS != nil {
		fmt.Fprintf(&breakdown, "%v\n", resS.Breakdown)
		usedBytes += resS.UsedBytes
	} else {
		breakdown.WriteString("\tMemory breakdown not supported.\n")
	}
	return &state.MemoryUsage{
		UsedBytes: usedBytes,
		Breakdown: stringStringer{breakdown.String()},
	}
}

func (s *shadowStateDb) GetShadowDB() state.StateDB {
	return s.shadow
}

type shadowBulkLoad struct {
	prime  state.BulkLoad
	shadow state.BulkLoad
}

func (l *shadowBulkLoad) CreateAccount(addr common.Address) {
	l.prime.CreateAccount(addr)
	l.shadow.CreateAccount(addr)
}

func (l *shadowBulkLoad) SetBalance(addr common.Address, value *uint256.Int) {
	l.prime.SetBalance(addr, value)
	l.shadow.SetBalance(addr, value)
}

func (l *shadowBulkLoad) SetNonce(addr common.Address, value uint64) {
	l.prime.SetNonce(addr, value)
	l.shadow.SetNonce(addr, value)
}

func (l *shadowBulkLoad) SetState(addr common.Address, key common.Hash, value common.Hash) {
	l.prime.SetState(addr, key, value)
	l.shadow.SetState(addr, key, value)
}

func (l *shadowBulkLoad) SetCode(addr common.Address, code []byte) {
	l.prime.SetCode(addr, code)
	l.shadow.SetCode(addr, code)
}

func (l *shadowBulkLoad) Close() error {
	return errors.Join(
		l.prime.Close(),
		l.shadow.Close(),
	)
}

func (s *shadowVmStateDb) run(opName string, op func(s state.VmStateDB) error) error {
	if err := op(s.prime); err != nil {
		return fmt.Errorf("prime: %w", err)
	}
	if err := op(s.shadow); err != nil {
		return fmt.Errorf("shadow: %w", err)
	}

	return nil
}

func (s *shadowNonCommittableStateDb) run(opName string, op func(s state.NonCommittableStateDB)) {
	op(s.prime)
	op(s.shadow)
}

func (s *shadowStateDb) run(opName string, op func(s state.StateDB) error) error {
	if err := op(s.prime); err != nil {
		return fmt.Errorf("prime: %w", err)
	}
	if err := op(s.shadow); err != nil {
		return fmt.Errorf("shadow: %w", err)
	}

	return nil
}

func (s *shadowVmStateDb) getBool(opName string, op func(s state.VmStateDB) bool, args ...any) bool {
	resP := op(s.prime)
	resS := op(s.shadow)
	if resP != resS {
		s.logIssue(opName, resP, resS, args)
		s.err = fmt.Errorf("%v diverged from shadow DB.", getOpcodeString(opName, args))
	}
	return resP
}

func (s *shadowVmStateDb) getBoolBool(opName string, op func(s state.VmStateDB) (bool, bool), args ...any) (bool, bool) {
	resP1, resP2 := op(s.prime)
	resS1, resS2 := op(s.shadow)
	if resP1 != resS1 || resP2 != resS2 {
		s.logIssue(opName, fmt.Sprintf("(%v,%v)", resP1, resP2), fmt.Sprintf("(%v,%v)", resS1, resS2), args)
		s.err = fmt.Errorf("%v diverged from shadow DB.", getOpcodeString(opName, args))
	}
	return resP1, resP2
}

func (s *shadowVmStateDb) getInt(opName string, op func(s state.VmStateDB) int, args ...any) int {
	resP := op(s.prime)
	resS := op(s.shadow)
	if resP != resS {
		s.logIssue(opName, resP, resS, args)
		s.err = fmt.Errorf("%v diverged from shadow DB.", getOpcodeString(opName, args))
	}
	return resP
}

func (s *shadowVmStateDb) getUint64(opName string, op func(s state.VmStateDB) uint64, args ...any) uint64 {
	resP := op(s.prime)
	resS := op(s.shadow)
	if resP != resS {
		s.logIssue(opName, resP, resS, args)
		s.err = fmt.Errorf("%v diverged from shadow DB.", getOpcodeString(opName, args))
	}
	return resP
}

func (s *shadowStateDb) getHash(opName string, op func(s state.StateDB) (common.Hash, error), args ...any) (common.Hash, error) {
	resP, err := op(s.prime)
	if err != nil {
		return common.Hash{}, err
	}
	resS, err := op(s.shadow)
	if err != nil {
		return common.Hash{}, err
	}
	if resP != resS {
		s.logIssue(opName, fmt.Sprintf("%x", resP), fmt.Sprintf("%x", resS), args)
		s.err = fmt.Errorf("%v diverged from shadow DB.", getOpcodeString(opName, args))
		return common.Hash{}, s.err
	}
	return resP, nil
}

func (s *shadowNonCommittableStateDb) getHash(opName string, op func(s state.NonCommittableStateDB) (common.Hash, error), args ...any) (common.Hash, error) {
	resP, err := op(s.prime)
	if err != nil {
		return common.Hash{}, err
	}
	resS, err := op(s.shadow)
	if err != nil {
		return common.Hash{}, err
	}
	if resP != resS {
		s.logIssue(opName, resP, resS, args)
		s.err = fmt.Errorf("%v diverged from shadow DB.", getOpcodeString(opName, args))
		return common.Hash{}, s.err
	}
	return resP, fmt.Errorf("%v diverged from shadow DB.", getOpcodeString(opName, args))
}

func (s *shadowVmStateDb) getStateHash(opName string, op func(s state.VmStateDB) (common.Hash, error), args ...any) (common.Hash, error) {
	resP, err := op(s.prime)
	if err != nil {
		return common.Hash{}, err
	}
	resS, err := op(s.shadow)
	if err != nil {
		return common.Hash{}, err
	}
	if resP != resS {
		s.logIssue(opName, resP, resS, args)
		s.err = fmt.Errorf("%v diverged from shadow DB.", getOpcodeString(opName, args))
	}
	return resP, nil
}

func (s *shadowVmStateDb) getHash(opName string, op func(s state.VmStateDB) common.Hash, args ...any) common.Hash {
	resP := op(s.prime)
	resS := op(s.shadow)
	if resP != resS {
		s.logIssue(opName, resP, resS, args)
		s.err = fmt.Errorf("%v diverged from shadow DB.", getOpcodeString(opName, args))
	}
	return resP
}

func (s *shadowVmStateDb) getUint256(opName string, op func(s state.VmStateDB) *uint256.Int, args ...any) *uint256.Int {
	resP := op(s.prime)
	resS := op(s.shadow)
	if resP.Cmp(resS) != 0 {
		s.logIssue(opName, resP, resS, args)
		s.err = fmt.Errorf("%v diverged from shadow DB.", getOpcodeString(opName, args))
	}
	return resP
}

func (s *shadowVmStateDb) getBytes(opName string, op func(s state.VmStateDB) []byte, args ...any) []byte {
	resP := op(s.prime)
	resS := op(s.shadow)
	if bytes.Compare(resP, resS) != 0 {
		s.logIssue(opName, resP, resS, args)
		s.err = fmt.Errorf("%v diverged from shadow DB.", getOpcodeString(opName, args))
	}
	return resP
}

func (s *shadowVmStateDb) getError(opName string, op func(s state.VmStateDB) error, args ...any) error {
	resP := op(s.prime)
	resS := op(s.shadow)
	if resP != resS {
		s.logIssue(opName, resP, resS, args)
		s.err = fmt.Errorf("%v diverged from shadow DB.", getOpcodeString(opName, args))
	}
	return resP
}

func (s *shadowStateDb) getError(opName string, op func(s state.StateDB) error, args ...any) error {
	resP := op(s.prime)
	resS := op(s.shadow)
	if resP != resS {
		s.logIssue(opName, resP, resS, args)
		s.err = fmt.Errorf("%v diverged from shadow DB.", getOpcodeString(opName, args))
	}
	return resP
}

func getOpcodeString(opName string, args ...any) string {
	var opcode strings.Builder
	opcode.WriteString(fmt.Sprintf("%v(", opName))
	for _, arg := range args {
		opcode.WriteString(fmt.Sprintf("%v ", arg))
	}
	opcode.WriteString(")")
	return opcode.String()
}

func (s *shadowVmStateDb) logIssue(opName string, prime, shadow any, args ...any) {
	s.log.Errorf("Diff for %v\n"+
		"\tPrimary: %v \n"+
		"\tShadow: %v", getOpcodeString(opName, args), prime, shadow)

}
