package operation

import (
	"encoding/binary"
	"io"
	"time"

	"github.com/Fantom-foundation/Aida/state"

	"github.com/Fantom-foundation/Aida/tracer/context"
)

// End-epoch operation data structure
type EndEpoch struct {
}

// GetId returns the end-epoch operation identifier.
func (op *EndEpoch) GetId() byte {
	return EndEpochID
}

// NewEndEpoch creates a new end-epoch operation.
func NewEndEpoch() *EndEpoch {
	return &EndEpoch{}
}

// ReadEndEpoch reads an end-epoch operation from file.
func ReadEndEpoch(f io.Reader) (Operation, error) {
	return new(EndEpoch), nil
}

// Write the end-epoch operation to file.
func (op *EndEpoch) Write(f io.Writer) error {
	err := binary.Write(f, binary.LittleEndian, *op)
	return err
}

// Execute the end-epoch operation.
func (op *EndEpoch) Execute(db state.StateDB, ctx *context.Context) time.Duration {
	start := time.Now()
	db.EndEpoch()
	return time.Since(start)
}

// Debug prints a debug message for the end-epoch operation.
func (op *EndEpoch) Debug(ctx *context.Context) {
}
