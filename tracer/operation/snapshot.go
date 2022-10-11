package operation

import (
	"os"

	"github.com/Fantom-foundation/aida/tracer/dict"
	"github.com/Fantom-foundation/substate-cli/state"
)

////////////////////////////////////////////////////////////
// Snapshot Operation
////////////////////////////////////////////////////////////

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
func (op *Snapshot) writeOperation(f *os.File) {
}

// Execute the snapshot operation.
func (op *Snapshot) Execute(db state.StateDB, ctx *dict.DictionaryContext) {
	db.Snapshot()
}

// Print the details for the snapshot operation.
func (op *Snapshot) Debug(*dict.DictionaryContext) {
}
