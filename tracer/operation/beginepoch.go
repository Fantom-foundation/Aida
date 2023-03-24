package operation

import (
	"encoding/binary"
	"fmt"
	"io"
	"time"

	"github.com/Fantom-foundation/Aida/state"

	"github.com/Fantom-foundation/Aida/tracer/context"
)

// BeginEpoch data structure
type BeginEpoch struct {
	EpochNumber uint64
}

// GetId returns the begin-epoch operation identifier.
func (op *BeginEpoch) GetId() byte {
	return BeginEpochID
}

// NewBeginEpoch creates a new begin-epoch operation.
func NewBeginEpoch(number uint64) *BeginEpoch {
	return &BeginEpoch{number}
}

// ReadBeginEpoch reads a begin-epoch operation from file.
func ReadBeginEpoch(f io.Reader) (Operation, error) {
	data := new(BeginEpoch)
	err := binary.Read(f, binary.LittleEndian, data)
	return data, err
}

// Write the begin-epoch operation to file.
func (op *BeginEpoch) Write(f io.Writer) error {
	return binary.Write(f, binary.LittleEndian, *op)
}

// Execute the begin-epoch operation.
func (op *BeginEpoch) Execute(db state.StateDB, ctx *context.Context) time.Duration {
	start := time.Now()
	db.BeginEpoch(op.EpochNumber)
	return time.Since(start)
}

// Debug prints a debug message for the begin-epoch operation.
func (op *BeginEpoch) Debug(ctx *context.Context) {
	fmt.Print(op.EpochNumber)
}
