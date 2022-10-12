package operation

import (
	"encoding/binary"
	"fmt"
	"os"

	"github.com/Fantom-foundation/aida/tracer/dict"
	"github.com/Fantom-foundation/aida/tracer/state"
)

// Revert-to-snapshot operation's data structure with returned snapshot id
type RevertToSnapshot struct {
	SnapshotID int32
}

// Return the revert-to-snapshot operation identifier.
func (op *RevertToSnapshot) GetOpId() byte {
	return RevertToSnapshotID
}

// Create a new revert-to-snapshot operation.
func NewRevertToSnapshot(SnapshotID int) *RevertToSnapshot {
	return &RevertToSnapshot{SnapshotID: int32(SnapshotID)}
}

// Read a revert-to-snapshot operation from file.
func ReadRevertToSnapshot(file *os.File) (Operation, error) {
	data := new(RevertToSnapshot)
	err := binary.Read(file, binary.LittleEndian, data)
	return data, err
}

// Write the revert-to-snapshot operation to file.
func (op *RevertToSnapshot) Write(f *os.File) error {
	err := binary.Write(f, binary.LittleEndian, *op)
	return err
}

// Execute the revert-to-snapshot operation.
func (op *RevertToSnapshot) Execute(db state.StateDB, ctx *dict.DictionaryContext) {
	db.RevertToSnapshot(int(op.SnapshotID))
}

// Print a debug message for revert-to-snapshot operation.
func (op *RevertToSnapshot) Debug(ctx *dict.DictionaryContext) {
	fmt.Printf("\tsnapshot id: %v\n", op.SnapshotID)
}
