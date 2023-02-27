package state

import (
	"bytes"
	"fmt"
	"log"
	"math/big"
	"strings"

	substate "github.com/Fantom-foundation/Substate"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// MakeShadowStateDB creates a StateDB instance bundeling two other instances and running each
// operation on both of them, cross checking results. If the results are not equal, an error
// is logged and the result of the primary instance is returned.
func MakeShadowStateDB(prime, shadow StateDB) StateDB {
	return &shadowStateDB{
		prime:     prime,
		shadow:    shadow,
		snapshots: []snapshotPair{},
	}
}

type shadowStateDB struct {
	prime     StateDB
	shadow    StateDB
	snapshots []snapshotPair
}

type snapshotPair struct {
	prime, shadow int
}

func (s *shadowStateDB) BeginBlockApply() error {
	return s.getError("BeginBlockApply", func(s StateDB) error { return s.BeginBlockApply() })
}

func (s *shadowStateDB) CreateAccount(addr common.Address) {
	s.run("CreateAccount", func(s StateDB) { s.CreateAccount(addr) })
}

func (s *shadowStateDB) Exist(addr common.Address) bool {
	return s.getBool("Exist", func(s StateDB) bool { return s.Exist(addr) }, addr)
}

func (s *shadowStateDB) Empty(addr common.Address) bool {
	return s.getBool("Empty", func(s StateDB) bool { return s.Empty(addr) }, addr)
}

func (s *shadowStateDB) Suicide(addr common.Address) bool {
	return s.getBool("Suicide", func(s StateDB) bool { return s.Suicide(addr) }, addr)
}

func (s *shadowStateDB) HasSuicided(addr common.Address) bool {
	return s.getBool("HasSuicided", func(s StateDB) bool { return s.HasSuicided(addr) }, addr)
}

func (s *shadowStateDB) GetBalance(addr common.Address) *big.Int {
	return s.getBigInt("GetBalance", func(s StateDB) *big.Int { return s.GetBalance(addr) }, addr)
}

func (s *shadowStateDB) AddBalance(addr common.Address, value *big.Int) {
	s.run("AddBalance", func(s StateDB) { s.AddBalance(addr, value) })
}

func (s *shadowStateDB) SubBalance(addr common.Address, value *big.Int) {
	s.run("SubBalance", func(s StateDB) { s.SubBalance(addr, value) })
}

func (s *shadowStateDB) GetNonce(addr common.Address) uint64 {
	return s.getUint64("GetNonce", func(s StateDB) uint64 { return s.GetNonce(addr) }, addr)
}

func (s *shadowStateDB) SetNonce(addr common.Address, value uint64) {
	s.run("SetNonce", func(s StateDB) { s.SetNonce(addr, value) })
}

func (s *shadowStateDB) GetCommittedState(addr common.Address, key common.Hash) common.Hash {
	return s.getHash("GetCommittedState", func(s StateDB) common.Hash { return s.GetCommittedState(addr, key) }, addr, key)
}

func (s *shadowStateDB) GetState(addr common.Address, key common.Hash) common.Hash {
	return s.getHash("GetState", func(s StateDB) common.Hash { return s.GetState(addr, key) }, addr, key)
}

func (s *shadowStateDB) SetState(addr common.Address, key common.Hash, value common.Hash) {
	s.run("SetState", func(s StateDB) { s.SetState(addr, key, value) })
}

func (s *shadowStateDB) GetCode(addr common.Address) []byte {
	return s.getBytes("GetCode", func(s StateDB) []byte { return s.GetCode(addr) }, addr)
}

func (s *shadowStateDB) GetCodeSize(addr common.Address) int {
	return s.getInt("GetCodeSize", func(s StateDB) int { return s.GetCodeSize(addr) }, addr)
}

func (s *shadowStateDB) GetCodeHash(addr common.Address) common.Hash {
	return s.getHash("GetCodeHash", func(s StateDB) common.Hash { return s.GetCodeHash(addr) }, addr)
}

func (s *shadowStateDB) SetCode(addr common.Address, code []byte) {
	s.run("SetCode", func(s StateDB) { s.SetCode(addr, code) })
}

func (s *shadowStateDB) Snapshot() int {
	pair := snapshotPair{
		s.prime.Snapshot(),
		s.shadow.Snapshot(),
	}
	s.snapshots = append(s.snapshots, pair)
	return len(s.snapshots) - 1
}

func (s *shadowStateDB) RevertToSnapshot(id int) {
	if id < 0 || len(s.snapshots) <= id {
		panic(fmt.Sprintf("invalid snapshot id: %v, max: %v", id, len(s.snapshots)))
	}
	s.prime.RevertToSnapshot(s.snapshots[id].prime)
	s.shadow.RevertToSnapshot(s.snapshots[id].shadow)
}

func (s *shadowStateDB) BeginTransaction(tx uint32) {
	s.snapshots = s.snapshots[0:0]
	s.run("BeginTransaction", func(s StateDB) { s.BeginTransaction(tx) })
}

func (s *shadowStateDB) EndTransaction() {
	s.run("EndTransaction", func(s StateDB) { s.EndTransaction() })
}

func (s *shadowStateDB) BeginBlock(blk uint64) {
	s.run("BeginBlock", func(s StateDB) { s.BeginBlock(blk) })
}

func (s *shadowStateDB) EndBlock() {
	s.run("EndBlock", func(s StateDB) { s.EndBlock() })
}

func (s *shadowStateDB) BeginEpoch(number uint64) {
	s.run("BeginEpoch", func(s StateDB) { s.BeginEpoch(number) })
}

func (s *shadowStateDB) EndEpoch() {
	s.run("EndEpoch", func(s StateDB) { s.EndEpoch() })
}

func (s *shadowStateDB) Close() error {
	return s.getError("Close", func(s StateDB) error { return s.Close() })
}

func (s *shadowStateDB) AddRefund(amount uint64) {
	s.run("AddRefund", func(s StateDB) { s.AddRefund(amount) })
	// check that the update value is the same
	s.getUint64("AddRefund", func(s StateDB) uint64 { return s.GetRefund() })
}

func (s *shadowStateDB) SubRefund(amount uint64) {
	s.run("SubRefund", func(s StateDB) { s.SubRefund(amount) })
	// check that the update value is the same
	s.getUint64("SubRefund", func(s StateDB) uint64 { return s.GetRefund() })
}

func (s *shadowStateDB) GetRefund() uint64 {
	return s.getUint64("GetRefund", func(s StateDB) uint64 { return s.GetRefund() })
}

func (s *shadowStateDB) PrepareAccessList(sender common.Address, dest *common.Address, precompiles []common.Address, txAccesses types.AccessList) {
	s.run("PrepareAccessList", func(s StateDB) { s.PrepareAccessList(sender, dest, precompiles, txAccesses) })
}

func (s *shadowStateDB) AddressInAccessList(addr common.Address) bool {
	return s.getBool("AddressInAccessList", func(s StateDB) bool { return s.AddressInAccessList(addr) }, addr)
}

func (s *shadowStateDB) SlotInAccessList(addr common.Address, slot common.Hash) (addressOk bool, slotOk bool) {
	return s.getBoolBool("SlotInAccessList", func(s StateDB) (bool, bool) { return s.SlotInAccessList(addr, slot) }, addr, slot)
}

func (s *shadowStateDB) AddAddressToAccessList(addr common.Address) {
	s.run("AddAddressToAccessList", func(s StateDB) { s.AddAddressToAccessList(addr) })
}

func (s *shadowStateDB) AddSlotToAccessList(addr common.Address, slot common.Hash) {
	s.run("AddSlotToAccessList", func(s StateDB) { s.AddSlotToAccessList(addr, slot) })
}

func (s *shadowStateDB) AddLog(log *types.Log) {
	s.run("AddLog", func(s StateDB) { s.AddLog(log) })
}

func (s *shadowStateDB) GetLogs(hash common.Hash, blockHash common.Hash) []*types.Log {
	// ignored
	return nil
}

func (s *shadowStateDB) Finalise(deleteEmptyObjects bool) {
	s.run("Finalise", func(s StateDB) { s.Finalise(deleteEmptyObjects) })
}

func (s *shadowStateDB) IntermediateRoot(deleteEmptyObjects bool) common.Hash {
	// Do not check hashes for equivalents.
	s.shadow.IntermediateRoot(deleteEmptyObjects)
	return s.prime.IntermediateRoot(deleteEmptyObjects)
}

func (s *shadowStateDB) Commit(deleteEmptyObjects bool) (common.Hash, error) {
	// Do not check hashes for equivalents.
	s.shadow.Commit(deleteEmptyObjects)
	return s.prime.Commit(deleteEmptyObjects)
}

func (s *shadowStateDB) Prepare(thash common.Hash, ti int) {
	s.run("Prepare", func(s StateDB) { s.Prepare(thash, ti) })
}

func (s *shadowStateDB) PrepareSubstate(substate *substate.SubstateAlloc, block uint64) {
	s.run("PrepareSubstate", func(s StateDB) { s.PrepareSubstate(substate, block) })
}

func (s *shadowStateDB) GetSubstatePostAlloc() substate.SubstateAlloc {
	// Skip comparing those results.
	s.shadow.GetSubstatePostAlloc()
	return s.prime.GetSubstatePostAlloc()
}

func (s *shadowStateDB) AddPreimage(hash common.Hash, plain []byte) {
	s.run("AddPreimage", func(s StateDB) { s.AddPreimage(hash, plain) })
}

func (s *shadowStateDB) ForEachStorage(common.Address, func(common.Hash, common.Hash) bool) error {
	// ignored
	panic("ForEachStorage not implemented")
}

func (s *shadowStateDB) StartBulkLoad() BulkLoad {
	return &shadowBulkLoad{s.prime.StartBulkLoad(), s.shadow.StartBulkLoad()}
}

func (s *shadowStateDB) GetArchiveState(block uint64) (StateDB, error) {
	var prime, shadow StateDB
	var err error
	if prime, err = s.prime.GetArchiveState(block); err != nil {
		return nil, err
	}
	if shadow, err = s.shadow.GetArchiveState(block); err != nil {
		return nil, err
	}
	return MakeShadowStateDB(prime, shadow), err
}

type stringStringer struct {
	str string
}

func (s stringStringer) String() string {
	return s.str
}

func (s *shadowStateDB) GetMemoryUsage() *MemoryUsage {
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
	return &MemoryUsage{
		UsedBytes: usedBytes,
		Breakdown: stringStringer{breakdown.String()},
	}
}

type shadowBulkLoad struct {
	prime  BulkLoad
	shadow BulkLoad
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

func (s *shadowStateDB) run(opName string, op func(s StateDB)) {
	op(s.prime)
	op(s.shadow)
}

func (s *shadowStateDB) getBool(opName string, op func(s StateDB) bool, args ...any) bool {
	resP := op(s.prime)
	resS := op(s.shadow)
	if resP != resS {
		logIssue(opName, resP, resS, args)
	}
	return resP
}

func (s *shadowStateDB) getBoolBool(opName string, op func(s StateDB) (bool, bool), args ...any) (bool, bool) {
	resP1, resP2 := op(s.prime)
	resS1, resS2 := op(s.shadow)
	if resP1 != resS1 || resP2 != resS2 {
		logIssue(opName, fmt.Sprintf("(%v,%v)", resP1, resP2), fmt.Sprintf("(%v,%v)", resS1, resS2), args)
	}
	return resP1, resP2
}

func (s *shadowStateDB) getInt(opName string, op func(s StateDB) int, args ...any) int {
	resP := op(s.prime)
	resS := op(s.shadow)
	if resP != resS {
		logIssue(opName, resP, resS, args)
	}
	return resP
}

func (s *shadowStateDB) getUint64(opName string, op func(s StateDB) uint64, args ...any) uint64 {
	resP := op(s.prime)
	resS := op(s.shadow)
	if resP != resS {
		logIssue(opName, resP, resS, args)
	}
	return resP
}

func (s *shadowStateDB) getHash(opName string, op func(s StateDB) common.Hash, args ...any) common.Hash {
	resP := op(s.prime)
	resS := op(s.shadow)
	if resP != resS {
		logIssue(opName, resP, resS, args)
	}
	return resP
}

func (s *shadowStateDB) getBigInt(opName string, op func(s StateDB) *big.Int, args ...any) *big.Int {
	resP := op(s.prime)
	resS := op(s.shadow)
	if resP.Cmp(resS) != 0 {
		logIssue(opName, resP, resS, args)
	}
	return resP
}

func (s *shadowStateDB) getBytes(opName string, op func(s StateDB) []byte, args ...any) []byte {
	resP := op(s.prime)
	resS := op(s.shadow)
	if bytes.Compare(resP, resS) != 0 {
		logIssue(opName, resP, resS, args)
	}
	return resP
}

func (s *shadowStateDB) getError(opName string, op func(s StateDB) error, args ...any) error {
	resP := op(s.prime)
	resS := op(s.shadow)
	if resP != resS {
		logIssue(opName, resP, resS, args)
	}
	return resP
}

func logIssue(opName string, prime, shadow any, args ...any) {
	log.Printf("Diff for %v(", opName)
	for _, arg := range args {
		log.Printf("\t%v", arg)
	}
	log.Printf(")\n")
	log.Printf("\tPrimary: %v\n", prime)
	log.Printf("\tShadow:  %v\n", shadow)
}
