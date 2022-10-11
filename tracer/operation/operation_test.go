package operation

import (
	"github.com/ethereum/go-ethereum/common"
	"os"
	"testing"
)

// Positive Test: Write/read test for GetState
func TestPositiveWriteReadGetState(t *testing.T) {
	filename := "./get_state_test.dat"

	//  write two test object to file
	f, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		t.Fatalf("Failed to open file for writing")
	}
	// write first object
	var sop = &GetState{ContractIndex: 1, StorageIndex: 2}
	sop.writeOperation(f)
	defer os.Remove(filename)
	// write second object
	sop.ContractIndex = 100
	sop.StorageIndex = 200
	sop.writeOperation(f)
	err = f.Close()
	if err != nil {
		t.Fatalf("Failed to close file for writing")
	}

	// read test object from file
	f, err = os.OpenFile(filename, os.O_CREATE|os.O_RDONLY, 0644)
	if err != nil {
		t.Fatalf("Failed to open file for reading")
	}
	// read first object & compare
	data, err := ReadGetState(f)
	if err != nil {
		t.Fatalf("Failed to read from file")
	}
	if data.ContractIndex != 1 || data.StorageIndex != 2 {
		t.Fatalf("Failed comparison")
	}
	// read second object & compare
	data, err = ReadGetState(f)
	if err != nil {
		t.Fatalf("Failed to read from file")
	}
	if data.ContractIndex != 100 || data.StorageIndex != 200 {
		t.Fatalf("Failed comparison")
	}
	err = f.Close()
	if err != nil {
		t.Fatalf("Failed to close file for reading")
	}
}

// Positive Test: Write/read test for SetState
func TestPositiveWriteReadSetState(t *testing.T) {
	filename := "./set_state_test.dat"

	//  write two test object to file
	f, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		t.Fatalf("Failed to open file for writing")
	}
	// write first object
	var sop = &SetState{ContractIndex: 1, StorageIndex: 2, Value: common.HexToHash("0x1000312211312312321312")}
	sop.writeOperation(f)
	defer os.Remove(filename)
	// write second object
	sop.ContractIndex = 100
	sop.StorageIndex = 200
	sop.Value = common.HexToHash("0x123111231231283012083")
	sop.writeOperation(f)
	err = f.Close()
	if err != nil {
		t.Fatalf("Failed to close file for writing")
	}

	// read test object from file
	f, err = os.OpenFile(filename, os.O_CREATE|os.O_RDONLY, 0644)
	if err != nil {
		t.Fatalf("Failed to open file for reading")
	}
	// read first object & compare
	data, err := ReadSetState(f)
	if err != nil {
		t.Fatalf("Failed to read from file")
	}
	if data.ContractIndex != 1 || data.StorageIndex != 2 {
		t.Fatalf("Failed comparison")
	}
	// read second object & compare
	data, err = ReadSetState(f)
	if err != nil {
		t.Fatalf("Failed to read from file")
	}
	if data.ContractIndex != 100 || data.StorageIndex != 200 {
		t.Fatalf("Failed comparison")
	}
	err = f.Close()
	if err != nil {
		t.Fatalf("Failed to close file for reading")
	}
}

// Positive Test: Write/read test for GetCommittedState
func TestPositiveWriteReadGetCommittedState(t *testing.T) {
	filename := "./get_committed_state_test.dat"

	//  write two test object to file
	f, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		t.Fatalf("Failed to open file for writing")
	}
	// write first object
	var sop = &GetCommittedState{ContractIndex: 1, StorageIndex: 2}
	sop.writeOperation(f)
	defer os.Remove(filename)
	// write second object
	sop.ContractIndex = 100
	sop.StorageIndex = 200
	sop.writeOperation(f)
	err = f.Close()
	if err != nil {
		t.Fatalf("Failed to close file for writing")
	}

	// read test object from file
	f, err = os.OpenFile(filename, os.O_CREATE|os.O_RDONLY, 0644)
	if err != nil {
		t.Fatalf("Failed to open file for reading")
	}
	// read first object & compare
	data, err := ReadGetCommittedState(f)
	if err != nil {
		t.Fatalf("Failed to read from file")
	}
	if data.ContractIndex != 1 || data.StorageIndex != 2 {
		t.Fatalf("Failed comparison")
	}
	// read second object & compare
	data, err = ReadGetCommittedState(f)
	if err != nil {
		t.Fatalf("Failed to read from file")
	}
	if data.ContractIndex != 100 || data.StorageIndex != 200 {
		t.Fatalf("Failed comparison")
	}
	err = f.Close()
	if err != nil {
		t.Fatalf("Failed to close file for reading")
	}
}

// Positive Test: Write/read test for Snapshot
func TestPositiveWriteReadSnapshotState(t *testing.T) {
	filename := "./snapshot_test.dat"

	//  write two test object to file
	f, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		t.Fatalf("Failed to open file for writing")
	}
	// write first object
	var sop = &Snapshot{}
	sop.writeOperation(f)
	defer os.Remove(filename)
	// write second object
	sop.writeOperation(f)
	err = f.Close()
	if err != nil {
		t.Fatalf("Failed to close file for writing")
	}
	// read test object from file
	f, err = os.OpenFile(filename, os.O_CREATE|os.O_RDONLY, 0644)
	if err != nil {
		t.Fatalf("Failed to open file for reading")
	}
	// read first object & compare
	_, err := ReadSnapshot(f)
	if err != nil {
		t.Fatalf("Failed to read from file")
	}
	// read second object & compare
	_, err = ReadSnapshot(f)
	if err != nil {
		t.Fatalf("Failed to read from file")
	}
	err = f.Close()
	if err != nil {
		t.Fatalf("Failed to close file for reading")
	}
}

// Positive Test: Write/read test for RevertToSnapshot
func TestPositiveWriteReadRevertToSnapshot(t *testing.T) {
	filename := "./revert_to_snapshot_test.dat"

	//  write two test object to file
	f, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		t.Fatalf("Failed to open file for writing")
	}
	// write first object
	sop1 := NewRevertToSnapshot(31)
	sop1.Write(f)
	defer os.Remove(filename)
	// write second object
	sop2 := NewRevertToSnapshot(200)
	sop2.Write(f)
	err = f.Close()
	if err != nil {
		t.Fatalf("Failed to close file for writing")
	}

	// read test object from file
	f, err = os.OpenFile(filename, os.O_CREATE|os.O_RDONLY, 0644)
	if err != nil {
		t.Fatalf("Failed to open file for reading")
	}
	// read first object & compare
	data, err := ReadRevertToSnapshot(f)
	if err != nil {
		t.Fatalf("Failed to read from file. %v", err)
	}
	if data.SnapshotID != sop1.SnapshotID {
		t.Fatalf("Failed comparison")
	}
	// read second object & compare
	data, err = ReadRevertToSnapshot(f)
	if err != nil {
		t.Fatalf("Failed to read from file")
	}
	if data.SnapshotID != sop2.SnapshotID {
		t.Fatalf("Failed comparison")
	}
	err = f.Close()
	if err != nil {
		t.Fatalf("Failed to close file for reading")
	}
}

// Positive Test: Write/read test for EndTransaction
func TestPositiveWriteReadEndTransactionState(t *testing.T) {
	filename := "./end_of_transaction_test.dat"

	//  write two test object to file
	f, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		t.Fatalf("Failed to open file for writing")
	}
	// write first object
	var sop = &EndTransaction{}
	sop.writeOperation(f)
	defer os.Remove(filename)
	// write second object
	sop.writeOperation(f)
	err = f.Close()
	if err != nil {
		t.Fatalf("Failed to close file for writing")
	}
	// read test object from file
	f, err = os.OpenFile(filename, os.O_CREATE|os.O_RDONLY, 0644)
	if err != nil {
		t.Fatalf("Failed to open file for reading")
	}
	// read first object & compare
	_, err := ReadEndTransaction(f)
	if err != nil {
		t.Fatalf("Failed to read from file")
	}
	// read second object & compare
	_, err = ReadEndTransaction(f)
	if err != nil {
		t.Fatalf("Failed to read from file")
	}
	err = f.Close()
	if err != nil {
		t.Fatalf("Failed to close file for reading")
	}
}
