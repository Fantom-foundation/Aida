package operation

import (
	"encoding/binary"
	"os"

	"github.com/Fantom-foundation/aida/tracer/dict"
	"github.com/Fantom-foundation/substate-cli/state"
)

// Snapshot data structure
type Snapshot struct {
}

// Return the snapshot operation identifier.
func (op *Snapshot) GetOpId() byte {
	return SnapshotID
}

// Create a new snapshot operation.
func NewSnapshot() *Snapshot {
	return &Snapshot{}
}

// Read a snapshot operation from a file.
func ReadSnapshot(file *os.File) (Operation, error) {
	return NewSnapshot(), nil
}

// Write the snapshot operation to file.
func (op *Snapshot) Write(f *os.File) error {
	err := binary.Write(f, binary.LittleEndian, *op)
	return err
}

// Execute the snapshot operation.
func (op *Snapshot) Execute(db state.StateDB, ctx *dict.DictionaryContext) {
	db.Snapshot()
}

// Print the details for the snapshot operation.
func (op *Snapshot) Debug(*dict.DictionaryContext) {
}
