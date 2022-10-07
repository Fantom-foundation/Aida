package tracer

import (
	"os"
	"testing"
)

// Add()
// Positive Test: Add a new set of blocks and compare the size of index map
func TestPositiveOperationIndexAdd(t *testing.T) {
	var blk1 uint64 = 1
	var blk2 uint64 = 2
	var pos1 uint64 = 0
	var pos2 uint64 = 1
	opIdx := NewOperationIndex()

	err1 := opIdx.Add(blk1, pos1)
	if err1 != nil {
		t.Fatalf("Failed to add new block: %v", err1)
	}
	err2 := opIdx.Add(blk2, pos2)
	if err2 != nil {
		t.Fatalf("Failed to add new block: %v", err2)
	}
	want := 2
	have := len(opIdx.blockToOperation)
	if have != want {
		t.Fatalf("Unexpected map size")
	}
}

// Negative Test: Add a duplicate block and compare whether the values are added twice
func TestNegativeOperationIndexAdd(t *testing.T) {
	var blk uint64 = 1
	var pos uint64 = 0
	opIdx := NewOperationIndex()

	err1 := opIdx.Add(blk, pos)
	if err1 != nil {
		t.Fatalf("Failed to add new block: %v", err1)
	}
	err2 := opIdx.Add(blk, pos)
	if err2 == nil {
		t.Fatalf("Expected an error when to add an existing block")
	}

	want := 1
	have := len(opIdx.blockToOperation)
	if have != want {
		t.Fatalf("Unexpectd map size")
	}
}

// Get()
// Positive Test: Get file positions from OperationIndex and compare index postions
func TestPositiveOperationIndexGet(t *testing.T) {
	var blk uint64 = 1
	var pos uint64 = 8
	opIdx := NewOperationIndex()

	opIdx.Add(blk, pos)
	opnum, err := opIdx.Get(blk)
	if err != nil {
		t.Fatalf("Failed to get block %va", blk)
	}
	if pos != opnum {
		t.Fatalf("Operation number mismatched")
	}
}

// Negative Test: Get file positions of a block whcih is not in OperationIndex
func TestNegativeOperationIndexGet(t *testing.T) {
	var blk uint64 = 1
	var pos uint64 = 8
	opIdx := NewOperationIndex()

	opIdx.Add(blk, pos)
	_, err := opIdx.Get(blk + 1)
	if err == nil {
		t.Fatalf("Failed to report error. Block %v doesn't exist", blk+1)
	}
}

// Read and Write()
// Positive Tetst: Write a set of postion index to a binary file and read from it.
// Compare whether indices are the same.
func TestPositiveOperationIndexReadWrite(t *testing.T) {
	var blk uint64 = 1
	var pos uint64 = 3
	filename := "./operation_index_test.dat"

	wOpIdx := NewOperationIndex()
	wOpIdx.Add(blk, pos)

	err1 := wOpIdx.Write(filename)
	defer os.Remove(filename)
	if err1 != nil {
		t.Fatalf("Failed to write file. %v", err1)
	}
	rOpIdx := NewOperationIndex()
	err2 := rOpIdx.Read(filename)
	if err2 != nil {
		t.Fatalf("Failed to read file. %v", err2)
	}
	opnum, err3 := rOpIdx.Get(blk)
	if err3 != nil {
		t.Fatalf("Failed to get block %v. %v", blk, err3)
	}
	if pos != opnum {
		t.Fatalf("Operation number mismatched")
	}
}

// Positive Tetst: Create
// Negative Tetst: Write a corrupted file and read from it.
func TestNegativeOperationIndexWrite(t *testing.T) {
	filename := "./operation_index_test.dat"
	f, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		t.Fatalf("Failed opening file")
	}
	defer os.Remove(filename)
	// write corrupted entry
	data := []byte("hello")
	if _, err := f.Write(data); err != nil {
		t.Fatalf("Failed to write data")
	}
	err = f.Close()
	if err != nil {
		t.Fatalf("Failed to close file")
	}
	opIdx := NewOperationIndex()
	err = opIdx.Read(filename)
	if err == nil {
		t.Fatalf("Failed to report error when reading a corrupted file")
	}
}
