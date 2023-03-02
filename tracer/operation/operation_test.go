package operation

import (
	"bytes"
	"io"
	"math/big"
	"os"
	"reflect"
	"testing"

	"github.com/Fantom-foundation/Aida/state"
	"github.com/Fantom-foundation/Aida/tracer/dictionary"
	substate "github.com/Fantom-foundation/Substate"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

// MockStateDB data structure
type MockStateDB struct {
	recording []Record //signatures of called functions
}

func (s *MockStateDB) BeginBlockApply() error {
	return nil
}

// NewMockStateDB creates a new mock StateDB object for testing execute
func NewMockStateDB() *MockStateDB {
	return &MockStateDB{}
}

// GetRecording retrieves the call signature of the last call
func (s *MockStateDB) GetRecording() []Record {
	return s.recording
}

// Record structure
type Record struct {
	function  byte  //signatures of called function
	arguments []any //arguments
}

func (s *MockStateDB) CreateAccount(addr common.Address) {
	s.recording = append(s.recording, Record{CreateAccountID, []any{addr}})
}

func (s *MockStateDB) Exist(addr common.Address) bool {
	s.recording = append(s.recording, Record{ExistID, []any{addr}})
	return false
}

func (s *MockStateDB) Empty(addr common.Address) bool {
	s.recording = append(s.recording, Record{EmptyID, []any{addr}})
	return false
}

func (s *MockStateDB) Suicide(addr common.Address) bool {
	s.recording = append(s.recording, Record{SuicideID, []any{addr}})
	return false
}

func (s *MockStateDB) HasSuicided(addr common.Address) bool {
	s.recording = append(s.recording, Record{HasSuicidedID, []any{addr}})
	return false
}

func (s *MockStateDB) GetBalance(addr common.Address) *big.Int {
	s.recording = append(s.recording, Record{GetBalanceID, []any{addr}})
	return &big.Int{}
}

func (s *MockStateDB) AddBalance(addr common.Address, value *big.Int) {
	s.recording = append(s.recording, Record{AddBalanceID, []any{addr, value}})
}

func (s *MockStateDB) SubBalance(addr common.Address, value *big.Int) {
	s.recording = append(s.recording, Record{SubBalanceID, []any{addr, value}})
}

func (s *MockStateDB) GetNonce(addr common.Address) uint64 {
	s.recording = append(s.recording, Record{GetNonceID, []any{addr}})
	return uint64(0)
}

func (s *MockStateDB) SetNonce(addr common.Address, value uint64) {
	s.recording = append(s.recording, Record{SetNonceID, []any{addr, value}})
}

func (s *MockStateDB) GetCommittedState(addr common.Address, key common.Hash) common.Hash {
	s.recording = append(s.recording, Record{GetCommittedStateID, []any{addr, key}})
	return common.Hash{}
}

func (s *MockStateDB) GetState(addr common.Address, key common.Hash) common.Hash {
	s.recording = append(s.recording, Record{GetStateID, []any{addr, key}})
	return common.Hash{}
}

func (s *MockStateDB) SetState(addr common.Address, key common.Hash, value common.Hash) {
	s.recording = append(s.recording, Record{SetStateID, []any{addr, key, value}})
}

func (s *MockStateDB) GetCode(addr common.Address) []byte {
	s.recording = append(s.recording, Record{GetCodeID, []any{addr}})
	return []byte{}
}

func (s *MockStateDB) GetCodeHash(addr common.Address) common.Hash {
	s.recording = append(s.recording, Record{GetCodeHashID, []any{addr}})
	return common.Hash{}
}

func (s *MockStateDB) GetCodeSize(addr common.Address) int {
	s.recording = append(s.recording, Record{GetCodeSizeID, []any{addr}})
	return 0
}

func (s *MockStateDB) SetCode(addr common.Address, code []byte) {
	s.recording = append(s.recording, Record{SetCodeID, []any{addr, code}})
}

func (s *MockStateDB) Snapshot() int {
	s.recording = append(s.recording, Record{SnapshotID, []any{}})
	return 0
}

func (s *MockStateDB) RevertToSnapshot(id int) {
	s.recording = append(s.recording, Record{RevertToSnapshotID, []any{id}})
}

func (s *MockStateDB) BeginTransaction(tx uint32) {
	s.recording = append(s.recording, Record{BeginTransactionID, []any{tx}})
}

func (s *MockStateDB) EndTransaction() {
	s.recording = append(s.recording, Record{EndTransactionID, []any{}})
}

func (s *MockStateDB) BeginBlock(blk uint64) {
	s.recording = append(s.recording, Record{BeginBlockID, []any{blk}})
}

func (s *MockStateDB) EndBlock() {
	s.recording = append(s.recording, Record{EndBlockID, []any{}})
}

func (s *MockStateDB) BeginEpoch(id uint64) {
	s.recording = append(s.recording, Record{BeginEpochID, []any{id}})
}

func (s *MockStateDB) EndEpoch() {
	s.recording = append(s.recording, Record{EndEpochID, []any{}})
}

func (s *MockStateDB) StartBulkLoad() state.BulkLoad {
	panic("Bulk load not supported in mock")
	return nil
}

func (s *MockStateDB) GetArchiveState(block uint64) (state.StateDB, error) {
	panic("Archive state not supported in mock")
}

func (s *MockStateDB) GetMemoryUsage() *state.MemoryUsage {
	panic("GetMemoryUsage not supported in mock")
	return nil
}

func (s *MockStateDB) Finalise(deleteEmptyObjects bool) {
	s.recording = append(s.recording, Record{FinaliseID, []any{deleteEmptyObjects}})
}

func (s *MockStateDB) IntermediateRoot(deleteEmptyObjects bool) common.Hash {
	s.recording = append(s.recording, Record{IntermediateRootID, []any{deleteEmptyObjects}})
	return common.Hash{}
}

func (s *MockStateDB) Commit(deleteEmptyObjects bool) (common.Hash, error) {
	s.recording = append(s.recording, Record{CommitID, []any{deleteEmptyObjects}})
	return common.Hash{}, nil
}

func (s *MockStateDB) Prepare(thash common.Hash, ti int) {
	s.recording = append(s.recording, Record{PrepareID, []any{thash, ti}})
}

func (s *MockStateDB) AddRefund(gas uint64) {
	s.recording = append(s.recording, Record{AddRefundID, []any{gas}})
}

func (s *MockStateDB) SubRefund(gas uint64) {
	s.recording = append(s.recording, Record{SubRefundID, []any{gas}})
}

func (s *MockStateDB) GetRefund() uint64 {
	s.recording = append(s.recording, Record{GetRefundID, []any{}})
	return uint64(0)
}

func (s *MockStateDB) PrepareAccessList(sender common.Address, dest *common.Address, precompiles []common.Address, txAccesses types.AccessList) {
	s.recording = append(s.recording, Record{PrepareAccessListID, []any{sender, dest, precompiles, txAccesses}})
}

func (s *MockStateDB) AddressInAccessList(addr common.Address) bool {
	s.recording = append(s.recording, Record{AddressInAccessListID, []any{addr}})
	return false
}

func (s *MockStateDB) SlotInAccessList(addr common.Address, slot common.Hash) (addressOk bool, slotOk bool) {
	s.recording = append(s.recording, Record{SlotInAccessListID, []any{addr, slot}})
	return false, false
}

func (s *MockStateDB) AddAddressToAccessList(addr common.Address) {
	s.recording = append(s.recording, Record{AddAddressToAccessListID, []any{addr}})
}

func (s *MockStateDB) AddSlotToAccessList(addr common.Address, slot common.Hash) {
	s.recording = append(s.recording, Record{AddSlotToAccessListID, []any{addr, slot}})
}

func (s *MockStateDB) AddLog(log *types.Log) {
	s.recording = append(s.recording, Record{AddLogID, []any{log}})
}

func (s *MockStateDB) AddPreimage(hash common.Hash, preimage []byte) {
	s.recording = append(s.recording, Record{AddPreimageID, []any{hash, preimage}})
}

func (s *MockStateDB) ForEachStorage(addr common.Address, cb func(common.Hash, common.Hash) bool) error {
	s.recording = append(s.recording, Record{ForEachStorageID, []any{addr, cb}})
	return nil
}

func (s *MockStateDB) GetLogs(hash common.Hash, blockHash common.Hash) []*types.Log {
	s.recording = append(s.recording, Record{GetLogsID, []any{hash, blockHash}})
	return nil
}

func (s *MockStateDB) PrepareSubstate(substate *substate.SubstateAlloc, block uint64) {
	// ignored
}

func (s *MockStateDB) GetSubstatePostAlloc() substate.SubstateAlloc {
	// ignored
	return substate.SubstateAlloc{}
}

func (s *MockStateDB) Close() error {
	s.recording = append(s.recording, Record{CloseID, []any{}})
	return nil
}

func (s *MockStateDB) compareRecordings(expected []Record, t *testing.T) {
	recording := s.GetRecording()

	if len(recording) != len(expected) {
		t.Fatalf("number of recorded operations was %d; expected %d", len(recording), len(expected))
	}

	for i, record := range recording {
		if record.function != expected[i].function {
			t.Fatal("wrong function order executed:", record, ", expected:", expected[i])
		}
		if len(record.arguments) != len(expected[i].arguments) {
			t.Fatalf("number of arguments at %d did not match received %d, expected %d", record.function, len(record.arguments), len(expected[i].arguments))
		}

		for k, arg := range record.arguments {
			if !areEqual(arg, expected[i].arguments[k]) {
				t.Fatalf("wrong function %s argument: %s, expected %s", GetLabel(record.function), arg, expected[i].arguments[k])
			}
		}
	}
}

// areEqual compares two values whether they are identical
func areEqual(v1 any, v2 any) bool {
	if reflect.TypeOf(v1) != reflect.TypeOf(v2) {
		return false
	}

	switch c1 := v1.(type) {
	case []byte:
		c2 := v2.([]byte)
		return bytes.Compare(c1, c2) == 0
	case *big.Int:
		c2 := v2.(*big.Int)
		return c2.Cmp(c1) == 0
	default:
		return v1 == v2
	}
}

func getRandomAddress(t *testing.T) common.Address {
	// generate account public key
	pk, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("failed test data build; could not create random keys; %s", err.Error())
	}
	// generate account address
	return crypto.PubkeyToAddress(pk.PublicKey)
}

func testOperationReadWrite(t *testing.T, op1 Operation, opRead func(file io.Reader) (Operation, error)) {
	opBuffer := bytes.NewBufferString("")
	err := op1.Write(opBuffer)
	if err != nil {
		t.Fatalf("error operation write %v", err)
	}

	// read object from buffer
	op2, err := opRead(opBuffer)
	if err != nil {
		t.Fatalf("failed to read operation. Error: %v", err)
	}
	if op2 == nil {
		t.Fatalf("failed to create newly read operation from buffer")
	}
	// check equivalence
	if !reflect.DeepEqual(op1, op2) {
		t.Fatalf("operations are not the same")
	}
}

func testOperationDebug(t *testing.T, dict *dictionary.Context, op Operation, args string) {
	// divert stdout to a buffer
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// print debug message
	Debug(dict, op)

	// restore stdout
	w.Close()
	os.Stdout = old
	var buf bytes.Buffer
	io.Copy(&buf, r)

	// check debug message
	label := GetLabel(op.GetId())

	expected := "\t" + label + ": " + args + "\n"

	if buf.String() != expected {
		t.Fatalf("wrong debug message: %s vs %s; length of strings: %d vs %d", buf.String(), expected, len(buf.String()), len(expected))
	}
}
