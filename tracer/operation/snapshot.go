package operation

import (
	"encoding/binary"
	"fmt"
	"log"
	"math"
	"os"
	"time"

	"github.com/Fantom-foundation/Aida/tracer/dict"
	"github.com/Fantom-foundation/Aida/tracer/state"
)

// Snapshot data structure
type Snapshot struct {
	SnapshotID uint16 // returned ID (for later mapping)
}

// Return the snapshot operation identifier.
func (op *Snapshot) GetOpId() byte {
	return SnapshotID
}

// Create a new snapshot operation.
func NewSnapshot(snapshotID int) *Snapshot {
	if snapshotID > math.MaxUint16 {
		log.Fatalf("Snapshot ID exceeds 16 bit")
	}
	return &Snapshot{SnapshotID: uint16(snapshotID)}
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
func (op *Snapshot) Execute(db state.StateDB, ctx *dict.DictionaryContext) time.Duration {
	start := time.Now()
	ID := db.Snapshot()
	elapsed := time.Since(start)
	if ID > math.MaxUint16 {
		log.Fatalf("Snapshot ID exceeds 16 bit")
	}
	ctx.AddSnapshot(op.SnapshotID, uint16(ID))
	return elapsed
}

// Print the details for the snapshot operation.
func (op *Snapshot) Debug(*dict.DictionaryContext) {
	fmt.Printf("\trecorded snapshot id: %v\n", op.SnapshotID)
}
