package tracer

import (
	"github.com/ethereum/go-ethereum/common"
	"os"
	"testing"
)

// Positive Test: Check whether GetFilename for index zero and one returns the first two
// operation filenames.
func TestPositiveGetFilename(t *testing.T) {
	fn := GetFilename(0)
	if fn != "./sop-getstate.dat" {
		t.Fatalf("GetFilename(0) failed; returns %v", fn)
	}
	fn = GetFilename(1)
	if fn != "./sop-setstate.dat" {
		t.Fatalf("GetFilename(1) failed; returns %v", fn)
	}
	fn = GetFilename(2)
	if fn != "./sop-getcommittedstate.dat" {
		t.Fatalf("GetFilename(2) failed; returns %v", fn)
	}
	fn = GetFilename(3)
	if fn != "./sop-snapshot.dat" {
		t.Fatalf("GetFilename(3) failed; returns %v", fn)
	}
	fn = GetFilename(4)
	if fn != "./sop-reverttosnapshot.dat" {
		t.Fatalf("GetFilename(4) failed; returns %v", fn)
	}
	fn = GetFilename(5)
	if fn != "./sop-endoftransaction.dat" {
		t.Fatalf("GetFilename(5) failed; returns %v", fn)
	}
}

// Positive Test: Write/read test for GetStateOperation
func TestPositiveWriteReadGetState(t *testing.T) {
	filename := "./get_state_test.dat"

	//  write two test object to file
	f, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		t.Fatalf("Failed to open file for writing")
	}
	// write first object
	var sop = &GetStateOperation{ContractIndex: 1, StorageIndex: 2}
	sop.GetWritable().Set(1001)
	sop.Write(f)
	defer os.Remove(filename)
	// write second object
	sop.ContractIndex = 100
	sop.StorageIndex = 200
	sop.GetWritable().Set(1010)
	sop.Write(f)
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
	data, err := ReadGetStateOperation(f)
	if err != nil {
		t.Fatalf("Failed to read from file")
	}
	if data.GetWritable().Get() != 1001 || data.ContractIndex != 1 || data.StorageIndex != 2 {
		t.Fatalf("Failed comparison")
	}
	// read second object & compare
	data, err = ReadGetStateOperation(f)
	if err != nil {
		t.Fatalf("Failed to read from file")
	}
	if data.GetWritable().Get() != 1010 || data.ContractIndex != 100 || data.StorageIndex != 200 {
		t.Fatalf("Failed comparison")
	}
	err = f.Close()
	if err != nil {
		t.Fatalf("Failed to close file for reading")
	}
}

// Positive Test: Write/read test for SetStateOperation
func TestPositiveWriteReadSetState(t *testing.T) {
	filename := "./set_state_test.dat"

	//  write two test object to file
	f, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		t.Fatalf("Failed to open file for writing")
	}
	// write first object
	var sop = &SetStateOperation{ContractIndex: 1, StorageIndex: 2, Value: common.HexToHash("0x1000312211312312321312")}
	sop.GetWritable().Set(1001)
	sop.Write(f)
	defer os.Remove(filename)
	// write second object
	sop.ContractIndex = 100
	sop.StorageIndex = 200
	sop.Value = common.HexToHash("0x123111231231283012083")
	sop.GetWritable().Set(1010)
	sop.Write(f)
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
	data, err := ReadSetStateOperation(f)
	if err != nil {
		t.Fatalf("Failed to read from file")
	}
	if data.GetWritable().Get() != 1001 || data.ContractIndex != 1 || data.StorageIndex != 2 {
		t.Fatalf("Failed comparison")
	}
	// read second object & compare
	data, err = ReadSetStateOperation(f)
	if err != nil {
		t.Fatalf("Failed to read from file")
	}
	if data.GetWritable().Get() != 1010 || data.ContractIndex != 100 || data.StorageIndex != 200 {
		t.Fatalf("Failed comparison")
	}
	err = f.Close()
	if err != nil {
		t.Fatalf("Failed to close file for reading")
	}
}

// Positive Test: Write/read test for GetCommittedStateOperation
func TestPositiveWriteReadGetCommittedState(t *testing.T) {
	filename := "./get_committed_state_test.dat"

	//  write two test object to file
	f, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		t.Fatalf("Failed to open file for writing")
	}
	// write first object
	var sop = &GetCommittedStateOperation{ContractIndex: 1, StorageIndex: 2}
	sop.GetWritable().Set(1001)
	sop.Write(f)
	defer os.Remove(filename)
	// write second object
	sop.ContractIndex = 100
	sop.StorageIndex = 200
	sop.GetWritable().Set(1010)
	sop.Write(f)
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
	data, err := ReadGetCommittedStateOperation(f)
	if err != nil {
		t.Fatalf("Failed to read from file")
	}
	if data.GetWritable().Get() != 1001 || data.ContractIndex != 1 || data.StorageIndex != 2 {
		t.Fatalf("Failed comparison")
	}
	// read second object & compare
	data, err = ReadGetCommittedStateOperation(f)
	if err != nil {
		t.Fatalf("Failed to read from file")
	}
	if data.GetWritable().Get() != 1010 || data.ContractIndex != 100 || data.StorageIndex != 200 {
		t.Fatalf("Failed comparison")
	}
	err = f.Close()
	if err != nil {
		t.Fatalf("Failed to close file for reading")
	}
}

// Positive Test: Write/read test for SnapshotOperation
func TestPositiveWriteReadSnapshotState(t *testing.T) {
	filename := "./snapshot_test.dat"

	//  write two test object to file
	f, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		t.Fatalf("Failed to open file for writing")
	}
	// write first object
	var sop = &SnapshotOperation{}
	sop.GetWritable().Set(1001)
	sop.Write(f)
	defer os.Remove(filename)
	// write second object
	sop.GetWritable().Set(1010)
	sop.Write(f)
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
	data, err := ReadSnapshotOperation(f)
	if err != nil {
		t.Fatalf("Failed to read from file")
	}
	if data.GetWritable().Get() != 1001 {
		t.Fatalf("Failed comparison")
	}
	// read second object & compare
	data, err = ReadSnapshotOperation(f)
	if err != nil {
		t.Fatalf("Failed to read from file")
	}
	if data.GetWritable().Get() != 1010 {
		t.Fatalf("Failed comparison")
	}
	err = f.Close()
	if err != nil {
		t.Fatalf("Failed to close file for reading")
	}
}

// Positive Test: Write/read test for RevertToSnapshotOperation
func TestPositiveWriteReadRevertToSnapshot(t *testing.T) {
	filename := "./revert_to_snapshot_test.dat"

	//  write two test object to file
	f, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		t.Fatalf("Failed to open file for writing")
	}
	// write first object
	sop1 := NewRevertToSnapshotOperation(31)
	sop1.GetWritable().Set(1001)
	sop1.Write(f)
	defer os.Remove(filename)
	// write second object
	sop2 := NewRevertToSnapshotOperation(200)
	sop2.GetWritable().Set(1010)
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
	data, err := ReadRevertToSnapshotOperation(f)
	if err != nil {
		t.Fatalf("Failed to read from file. %v", err)
	}
	if data.GetWritable().Get() != 1001 || data.SnapshotID != sop1.SnapshotID {
		t.Fatalf("Failed comparison")
	}
	// read second object & compare
	data, err = ReadRevertToSnapshotOperation(f)
	if err != nil {
		t.Fatalf("Failed to read from file")
	}
	if data.GetWritable().Get() != 1010 || data.SnapshotID != sop2.SnapshotID {
		t.Fatalf("Failed comparison")
	}
	err = f.Close()
	if err != nil {
		t.Fatalf("Failed to close file for reading")
	}
}

// Positive Test: Write/read test for EndTransactionOperation
func TestPositiveWriteReadEndTransactionState(t *testing.T) {
	filename := "./end_of_transaction_test.dat"

	//  write two test object to file
	f, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		t.Fatalf("Failed to open file for writing")
	}
	// write first object
	var sop = &EndTransactionOperation{}
	sop.GetWritable().Set(1001)
	sop.Write(f)
	defer os.Remove(filename)
	// write second object
	sop.GetWritable().Set(1010)
	sop.Write(f)
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
	data, err := ReadEndTransactionOperation(f)
	if err != nil {
		t.Fatalf("Failed to read from file")
	}
	if data.GetWritable().Get() != 1001 {
		t.Fatalf("Failed comparison")
	}
	// read second object & compare
	data, err = ReadEndTransactionOperation(f)
	if err != nil {
		t.Fatalf("Failed to read from file")
	}
	if data.GetWritable().Get() != 1010 {
		t.Fatalf("Failed comparison")
	}
	err = f.Close()
	if err != nil {
		t.Fatalf("Failed to close file for reading")
	}
}
