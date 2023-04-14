package operation

import (
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"math"
	"time"

	"github.com/Fantom-foundation/Aida/state"

	"github.com/Fantom-foundation/Aida/tracer/context"
)

// Snapshot data structure
type Snapshot struct {
	SnapshotID int32 // returned ID (for later mapping)
}

// GetId returns the snapshot operation identifier.
func (op *Snapshot) GetId() byte {
	return SnapshotID
}

// NewSnapshot creates a new snapshot operation.
func NewSnapshot(SnapshotID int32) *Snapshot {
	return &Snapshot{SnapshotID: SnapshotID}
}

// ReadSnapshot reads a snapshot operation from a file.
func ReadSnapshot(f io.Reader) (Operation, error) {
	data := new(Snapshot)
	err := binary.Read(f, binary.LittleEndian, data)
	return data, err
}

// Write the snapshot operation to file.
func (op *Snapshot) Write(f io.Writer) error {
	err := binary.Write(f, binary.LittleEndian, *op)
	return err
}

// Execute the snapshot operation.
func (op *Snapshot) Execute(db state.StateDB, ctx *context.Replay) time.Duration {
	start := time.Now()
	ID := db.Snapshot()
	elapsed := time.Since(start)
	if ID > math.MaxInt32 {
		log.Fatalf("Snapshot ID exceeds 32 bit")
	}
	ctx.AddSnapshot(op.SnapshotID, int32(ID))
	return elapsed
}

// Debug prints the details for the snapshot operation.
func (op *Snapshot) Debug(*context.Context) {
	fmt.Print(op.SnapshotID)
}
