package proxy

import (
	"bytes"
	"fmt"
	"log"
	"math/big"
	"strings"

	"github.com/Fantom-foundation/Aida/state"
	substate "github.com/Fantom-foundation/Substate"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/op/go-logging"
)

// NewShadowProxy creates a StateDB instance bundeling two other instances and running each
// operation on both of them, cross checking results. If the results are not equal, an error
// is logged and the result of the primary instance is returned.
func NewShadowProxy(prime, shadow state.StateDB) state.StateDB {
	return &ShadowProxy{
		prime:     prime,
		shadow:    shadow,
		snapshots: []snapshotPair{},
		err:       nil,
	}
}

type ShadowProxy struct {
	prime     state.StateDB
	shadow    state.StateDB
	snapshots []snapshotPair
	err       error
	log       *logging.Logger
}

type snapshotPair struct {
	prime, shadow int
}

func (s *ShadowProxy) CreateAccount(addr common.Address) {
	s.run("CreateAccount", func(s state.StateDB) { s.CreateAccount(addr) })
}

func (s *ShadowProxy) Exist(addr common.Address) bool {
	return s.getBool("Exist", func(s state.StateDB) bool { return s.Exist(addr) }, addr)
}

func (s *ShadowProxy) Empty(addr common.Address) bool {
	return s.getBool("Empty", func(s state.StateDB) bool { return s.Empty(addr) }, addr)
}

func (s *ShadowProxy) Suicide(addr common.Address) bool {
	return s.getBool("Suicide", func(s state.StateDB) bool { return s.Suicide(addr) }, addr)
}

func (s *ShadowProxy) HasSuicided(addr common.Address) bool {
	return s.getBool("HasSuicided", func(s state.StateDB) bool { return s.HasSuicided(addr) }, addr)
}

func (s *ShadowProxy) GetBalance(addr common.Address) *big.Int {
	return s.getBigInt("GetBalance", func(s state.StateDB) *big.Int { return s.GetBalance(addr) }, addr)
}

func (s *ShadowProxy) AddBalance(addr common.Address, value *big.Int) {
	s.run("AddBalance", func(s state.StateDB) { s.AddBalance(addr, value) })
}

func (s *ShadowProxy) SubBalance(addr common.Address, value *big.Int) {
	s.run("SubBalance", func(s state.StateDB) { s.SubBalance(addr, value) })
}

func (s *ShadowProxy) GetNonce(addr common.Address) uint64 {
	return s.getUint64("GetNonce", func(s state.StateDB) uint64 { return s.GetNonce(addr) }, addr)
}

func (s *ShadowProxy) SetNonce(addr common.Address, value uint64) {
	s.run("SetNonce", func(s state.StateDB) { s.SetNonce(addr, value) })
}

func (s *ShadowProxy) GetCommittedState(addr common.Address, key common.Hash) common.Hash {
	return s.getHash("GetCommittedState", func(s state.StateDB) common.Hash { return s.GetCommittedState(addr, key) }, addr, key)
}

func (s *ShadowProxy) GetState(addr common.Address, key common.Hash) common.Hash {
	return s.getHash("GetState", func(s state.StateDB) common.Hash { return s.GetState(addr, key) }, addr, key)
}

func (s *ShadowProxy) SetState(addr common.Address, key common.Hash, value common.Hash) {
	s.run("SetState", func(s state.StateDB) { s.SetState(addr, key, value) })
}

func (s *ShadowProxy) GetCode(addr common.Address) []byte {
	return s.getBytes("GetCode", func(s state.StateDB) []byte { return s.GetCode(addr) }, addr)
}

func (s *ShadowProxy) GetCodeSize(addr common.Address) int {
	return s.getInt("GetCodeSize", func(s state.StateDB) int { return s.GetCodeSize(addr) }, addr)
}

func (s *ShadowProxy) GetCodeHash(addr common.Address) common.Hash {
	return s.getHash("GetCodeHash", func(s state.StateDB) common.Hash { return s.GetCodeHash(addr) }, addr)
}

func (s *ShadowProxy) SetCode(addr common.Address, code []byte) {
	s.run("SetCode", func(s state.StateDB) { s.SetCode(addr, code) })
}

func (s *ShadowProxy) Snapshot() int {
	pair := snapshotPair{
		s.prime.Snapshot(),
		s.shadow.Snapshot(),
	}
	s.snapshots = append(s.snapshots, pair)
	return len(s.snapshots) - 1
}

func (s *ShadowProxy) RevertToSnapshot(id int) {
	if id < 0 || len(s.snapshots) <= id {
		panic(fmt.Sprintf("invalid snapshot id: %v, max: %v", id, len(s.snapshots)))
	}
	s.prime.RevertToSnapshot(s.snapshots[id].prime)
	s.shadow.RevertToSnapshot(s.snapshots[id].shadow)
}

func (s *ShadowProxy) BeginTransaction(tx uint32) {
	s.snapshots = s.snapshots[0:0]
	s.run("BeginTransaction", func(s state.StateDB) { s.BeginTransaction(tx) })
}

func (s *ShadowProxy) EndTransaction() {
	s.run("EndTransaction", func(s state.StateDB) { s.EndTransaction() })
}

func (s *ShadowProxy) BeginBlock(blk uint64) {
	s.run("BeginBlock", func(s state.StateDB) { s.BeginBlock(blk) })
}

func (s *ShadowProxy) EndBlock() {
	s.run("EndBlock", func(s state.StateDB) { s.EndBlock() })
}

func (s *ShadowProxy) BeginSyncPeriod(number uint64) {
	s.run("BeginSyncPeriod", func(s state.StateDB) { s.BeginSyncPeriod(number) })
}

func (s *ShadowProxy) EndSyncPeriod() {
	s.run("EndSyncPeriod", func(s state.StateDB) { s.EndSyncPeriod() })
}

func (s *ShadowProxy) GetHash() common.Hash {
	return s.prime.GetHash()
}

func (s *ShadowProxy) Close() error {
	return s.getError("Close", func(s state.StateDB) error { return s.Close() })
}

func (s *ShadowProxy) AddRefund(amount uint64) {
	s.run("AddRefund", func(s state.StateDB) { s.AddRefund(amount) })
	// check that the update value is the same
	s.getUint64("AddRefund", func(s state.StateDB) uint64 { return s.GetRefund() })
}

func (s *ShadowProxy) SubRefund(amount uint64) {
	s.run("SubRefund", func(s state.StateDB) { s.SubRefund(amount) })
	// check that the update value is the same
	s.getUint64("SubRefund", func(s state.StateDB) uint64 { return s.GetRefund() })
}

func (s *ShadowProxy) GetRefund() uint64 {
	return s.getUint64("GetRefund", func(s state.StateDB) uint64 { return s.GetRefund() })
}

func (s *ShadowProxy) PrepareAccessList(sender common.Address, dest *common.Address, precompiles []common.Address, txAccesses types.AccessList) {
	s.run("PrepareAccessList", func(s state.StateDB) { s.PrepareAccessList(sender, dest, precompiles, txAccesses) })
}

func (s *ShadowProxy) AddressInAccessList(addr common.Address) bool {
	return s.getBool("AddressInAccessList", func(s state.StateDB) bool { return s.AddressInAccessList(addr) }, addr)
}

func (s *ShadowProxy) SlotInAccessList(addr common.Address, slot common.Hash) (addressOk bool, slotOk bool) {
	return s.getBoolBool("SlotInAccessList", func(s state.StateDB) (bool, bool) { return s.SlotInAccessList(addr, slot) }, addr, slot)
}

func (s *ShadowProxy) AddAddressToAccessList(addr common.Address) {
	s.run("AddAddressToAccessList", func(s state.StateDB) { s.AddAddressToAccessList(addr) })
}

func (s *ShadowProxy) AddSlotToAccessList(addr common.Address, slot common.Hash) {
	s.run("AddSlotToAccessList", func(s state.StateDB) { s.AddSlotToAccessList(addr, slot) })
}

func (s *ShadowProxy) AddLog(log *types.Log) {
	s.run("AddLog", func(s state.StateDB) { s.AddLog(log) })
}

func (s *ShadowProxy) GetLogs(hash common.Hash, blockHash common.Hash) []*types.Log {
	logsP := s.prime.GetLogs(hash, blockHash)
	logsS := s.shadow.GetLogs(hash, blockHash)

	equal := len(logsP) == len(logsS)
	if equal {
		for i, logP := range logsP {
			logS := logsS[i]
			if logP != logS {
				equal = false
				break
			}
		}
	}
	if !equal {
		s.logIssue("GetLogs", logsP, logsS, hash, blockHash)
		s.err = fmt.Errorf("%v diverged from shadow DB", getOpcodeString("GetLogs", hash, blockHash))
	}
	return logsP
}

func (s *ShadowProxy) Finalise(deleteEmptyObjects bool) {
	s.run("Finalise", func(s state.StateDB) { s.Finalise(deleteEmptyObjects) })
}

func (s *ShadowProxy) IntermediateRoot(deleteEmptyObjects bool) common.Hash {
	// Do not check hashes for equivalents.
	s.shadow.IntermediateRoot(deleteEmptyObjects)
	return s.prime.IntermediateRoot(deleteEmptyObjects)
}

func (s *ShadowProxy) Commit(deleteEmptyObjects bool) (common.Hash, error) {
	// Do not check hashes for equivalents.
	s.shadow.Commit(deleteEmptyObjects)
	return s.prime.Commit(deleteEmptyObjects)
}

// GetError returns an error then reset it.
func (s *ShadowProxy) Error() error {
	err := s.err
	// reset error message
	s.err = nil
	return err
}

func (s *ShadowProxy) Prepare(thash common.Hash, ti int) {
	s.run("Prepare", func(s state.StateDB) { s.Prepare(thash, ti) })
}

func (s *ShadowProxy) PrepareSubstate(substate *substate.SubstateAlloc, block uint64) {
	s.run("PrepareSubstate", func(s state.StateDB) { s.PrepareSubstate(substate, block) })
}

func (s *ShadowProxy) GetSubstatePostAlloc() substate.SubstateAlloc {
	// Skip comparing those results.
	s.shadow.GetSubstatePostAlloc()
	return s.prime.GetSubstatePostAlloc()
}

func (s *ShadowProxy) AddPreimage(hash common.Hash, plain []byte) {
	s.run("AddPreimage", func(s state.StateDB) { s.AddPreimage(hash, plain) })
}

func (s *ShadowProxy) ForEachStorage(common.Address, func(common.Hash, common.Hash) bool) error {
	// ignored
	panic("ForEachStorage not implemented")
}

func (s *ShadowProxy) StartBulkLoad(block uint64) state.BulkLoad {
	return &shadowBulkLoad{s.prime.StartBulkLoad(block), s.shadow.StartBulkLoad(block)}
}

func (s *ShadowProxy) GetArchiveState(block uint64) (state.StateDB, error) {
	var prime, shadow state.StateDB
	var err error
	if prime, err = s.prime.GetArchiveState(block); err != nil {
		return nil, err
	}
	if shadow, err = s.shadow.GetArchiveState(block); err != nil {
		return nil, err
	}
	return &ShadowProxy{
		prime:     prime,
		shadow:    shadow,
		snapshots: []snapshotPair{},
		err:       nil,
		log:       s.log,
	}, nil
}

type stringStringer struct {
	str string
}

func (s stringStringer) String() string {
	return s.str
}

func (s *ShadowProxy) GetMemoryUsage() *state.MemoryUsage {
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

func (s *ShadowProxy) GetShadowDB() state.StateDB {
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

func (l *shadowBulkLoad) SetBalance(addr common.Address, value *big.Int) {
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
	if err := l.prime.Close(); err != nil {
		return err
	}
	if err := l.shadow.Close(); err != nil {
		return err
	}
	return nil
}

func (s *ShadowProxy) run(opName string, op func(s state.StateDB)) {
	op(s.prime)
	op(s.shadow)
}

func (s *ShadowProxy) getBool(opName string, op func(s state.StateDB) bool, args ...any) bool {
	resP := op(s.prime)
	resS := op(s.shadow)
	if resP != resS {
		s.logIssue(opName, resP, resS, args)
		s.err = fmt.Errorf("%v diverged from shadow DB.", getOpcodeString(opName, args))
	}
	return resP
}

func (s *ShadowProxy) getBoolBool(opName string, op func(s state.StateDB) (bool, bool), args ...any) (bool, bool) {
	resP1, resP2 := op(s.prime)
	resS1, resS2 := op(s.shadow)
	if resP1 != resS1 || resP2 != resS2 {
		s.logIssue(opName, fmt.Sprintf("(%v,%v)", resP1, resP2), fmt.Sprintf("(%v,%v)", resS1, resS2), args)
		s.err = fmt.Errorf("%v diverged from shadow DB.", getOpcodeString(opName, args))
	}
	return resP1, resP2
}

func (s *ShadowProxy) getInt(opName string, op func(s state.StateDB) int, args ...any) int {
	resP := op(s.prime)
	resS := op(s.shadow)
	if resP != resS {
		s.logIssue(opName, resP, resS, args)
		s.err = fmt.Errorf("%v diverged from shadow DB.", getOpcodeString(opName, args))
	}
	return resP
}

func (s *ShadowProxy) getUint64(opName string, op func(s state.StateDB) uint64, args ...any) uint64 {
	resP := op(s.prime)
	resS := op(s.shadow)
	if resP != resS {
		s.logIssue(opName, resP, resS, args)
		s.err = fmt.Errorf("%v diverged from shadow DB.", getOpcodeString(opName, args))
	}
	return resP
}

func (s *ShadowProxy) getHash(opName string, op func(s state.StateDB) common.Hash, args ...any) common.Hash {
	resP := op(s.prime)
	resS := op(s.shadow)
	if resP != resS {
		s.logIssue(opName, resP, resS, args)
		s.err = fmt.Errorf("%v diverged from shadow DB.", getOpcodeString(opName, args))
	}
	return resP
}

func (s *ShadowProxy) getBigInt(opName string, op func(s state.StateDB) *big.Int, args ...any) *big.Int {
	resP := op(s.prime)
	resS := op(s.shadow)
	if resP.Cmp(resS) != 0 {
		s.logIssue(opName, resP, resS, args)
		s.err = fmt.Errorf("%v diverged from shadow DB.", getOpcodeString(opName, args))
	}
	return resP
}

func (s *ShadowProxy) getBytes(opName string, op func(s state.StateDB) []byte, args ...any) []byte {
	resP := op(s.prime)
	resS := op(s.shadow)
	if bytes.Compare(resP, resS) != 0 {
		s.logIssue(opName, resP, resS, args)
		s.err = fmt.Errorf("%v diverged from shadow DB.", getOpcodeString(opName, args))
	}
	return resP
}

func (s *ShadowProxy) getError(opName string, op func(s state.StateDB) error, args ...any) error {
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

func (s *ShadowProxy) logIssue(opName string, prime, shadow any, args ...any) {
	log.Printf("Diff for %v\n"+
		"\tPrimary: %v \n"+
		"\tShadow: %v", getOpcodeString(opName, args), prime, shadow)

}
