package operation

import (
	"encoding/binary"
	"fmt"
	"io"
	"time"

	"github.com/Fantom-foundation/Aida/tracer/dict"
	"github.com/Fantom-foundation/Aida/tracer/state"
)

// EndEpoch data structure
type EndEpoch struct {
	EpochNumber uint64
}

// GetId returns the end-epoch operation identifier.
func (op *EndEpoch) GetId() byte {
	return EndEpochID
}

// NewEndEpoch creates a new end-epoch operation.
func NewEndEpoch(number uint64) *EndEpoch {
	return &EndEpoch{number}
}

// ReadEndEpoch reads an end-epoch operation from file.
func ReadEndEpoch(file io.Reader) (Operation, error) {
	data := new(EndEpoch)
	err := binary.Read(file, binary.LittleEndian, data)
	return data, err
}

// Write the end-epoch operation to file.
func (op *EndEpoch) Write(f io.Writer) error {
	return binary.Write(f, binary.LittleEndian, *op)
}

// Execute the end-epoch operation.
func (op *EndEpoch) Execute(db state.StateDB, ctx *dict.DictionaryContext) time.Duration {
	start := time.Now()
	db.EndEpoch(op.EpochNumber)
	return time.Since(start)
}

// Debug prints a debug message for the end-epoch operation.
func (op *EndEpoch) Debug(ctx *dict.DictionaryContext) {
	fmt.Print(op.EpochNumber)
}
