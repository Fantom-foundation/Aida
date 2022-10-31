package operation

import (
	"encoding/binary"
	"fmt"
	"io"
	"time"

	"github.com/Fantom-foundation/Aida/tracer/dict"
	"github.com/Fantom-foundation/Aida/tracer/state"
)

// RevertToSnapshot data structure
type RevertToSnapshot struct {
	SnapshotID int32 // snapshot id limited to 32 bits.
}

// RevertToSnapshot returns the revert-to-snapshot operation identifier.
func (op *RevertToSnapshot) GetOpId() byte {
	return RevertToSnapshotID
}

// NewRevertToSnapshot creates a new revert-to-snapshot operation.
func NewRevertToSnapshot(SnapshotID int) *RevertToSnapshot {
	return &RevertToSnapshot{SnapshotID: int32(SnapshotID)}
}

// ReadRevertToSnapshot reads revert-to-snapshot operation from file.
func ReadRevertToSnapshot(file io.Reader) (Operation, error) {
	data := new(RevertToSnapshot)
	err := binary.Read(file, binary.LittleEndian, data)
	return data, err
}

// Write the revert-to-snapshot operation to file.
func (op *RevertToSnapshot) Write(f io.Writer) error {
	err := binary.Write(f, binary.LittleEndian, *op)
	return err
}

// Execute the revert-to-snapshot operation.
func (op *RevertToSnapshot) Execute(db state.StateDB, ctx *dict.DictionaryContext) time.Duration {
	id := ctx.GetSnapshot(op.SnapshotID)
	start := time.Now()
	db.RevertToSnapshot(int(id))
	return time.Since(start)
}

// Debug prints a debug message for the revert-to-snapshot operation.
func (op *RevertToSnapshot) Debug(ctx *dict.DictionaryContext) {
	fmt.Printf("\tsnapshot id: %v\n", op.SnapshotID)
}
