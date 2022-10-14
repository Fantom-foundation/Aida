package operation

import (
	"encoding/binary"
	"fmt"
	"os"

	"github.com/Fantom-foundation/aida/tracer/dict"
	"github.com/Fantom-foundation/aida/tracer/state"
)

// Snapshot data structure
type Snapshot struct {
	SnapshotID int32 // returned ID (for later mapping)
}

// Return the snapshot operation identifier.
func (op *Snapshot) GetOpId() byte {
	return SnapshotID
}

// Create a new snapshot operation.
func NewSnapshot(SnapshotID int32) *Snapshot {
	return &Snapshot{SnapshotID: SnapshotID}
}

// Read a snapshot operation from a file.
func ReadSnapshot(file *os.File) (Operation, error) {
	data := new(Snapshot)
	err := binary.Read(file, binary.LittleEndian, data)
	return data, err
}

// Write the snapshot operation to file.
func (op *Snapshot) Write(f *os.File) error {
	err := binary.Write(f, binary.LittleEndian, *op)
	return err
}

// Execute the snapshot operation.
func (op *Snapshot) Execute(db state.StateDB, ctx *dict.DictionaryContext) {
	ID := db.Snapshot()
	// TODO: check that ID does not exceed 32bit
	ctx.AddSnapshot(op.SnapshotID, int32(ID))
}

// Print the details for the snapshot operation.
func (op *Snapshot) Debug(*dict.DictionaryContext) {
	fmt.Printf("\trecorded snapshot id: %v\n", op.SnapshotID)
}
