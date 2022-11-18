package operation

import (
	"encoding/binary"
	"io"
	"time"

	"github.com/Fantom-foundation/Aida/tracer/dict"
	"github.com/Fantom-foundation/Aida/tracer/state"
)

// Begin-epoch operation data structure
type BeginEpoch struct {
}

// GetId returns the begin-epoch operation identifier.
func (op *BeginEpoch) GetId() byte {
	return BeginEpochID
}

// NewBeginEpoch creates a new begin-epoch operation.
func NewBeginEpoch() *BeginEpoch {
	return &BeginEpoch{}
}

// ReadBeginEpoch reads a begin-epoch operation from file.
func ReadBeginEpoch(file io.Reader) (Operation, error) {
	return new(BeginEpoch), nil
}

// Write the begin-epoch operation to file.
func (op *BeginEpoch) Write(f io.Writer) error {
	err := binary.Write(f, binary.LittleEndian, *op)
	return err
}

// Execute the begin-epoch operation.
func (op *BeginEpoch) Execute(db state.StateDB, ctx *dict.DictionaryContext) time.Duration {
	start := time.Now()
	db.BeginEpoch()
	return time.Since(start)
}

// Debug prints a debug message for the begin-epoch operation.
func (op *BeginEpoch) Debug(ctx *dict.DictionaryContext) {
}
