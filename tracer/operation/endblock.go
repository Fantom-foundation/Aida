package operation

import (
	"encoding/binary"
	"io"
	"time"

	"github.com/Fantom-foundation/Aida/tracer/dict"
	"github.com/Fantom-foundation/Aida/tracer/state"
)

// Endblock data structure
type EndBlock struct {
}

// GetId returns the end-block operation identifier.
func (op *EndBlock) GetId() byte {
	return EndBlockID
}

// NewEndBlock creates a new end-block operation.
func NewEndBlock() *EndBlock {
	return &EndBlock{}
}

// ReadEndBlock reads an end-block operation from file.
func ReadEndBlock(file io.Reader) (Operation, error) {
	return new(EndBlock), nil
}

// Write the end-block operation to file.
func (op *EndBlock) Write(f io.Writer) error {
	return binary.Write(f, binary.LittleEndian, *op)
}

// Execute the end-block operation.
func (op *EndBlock) Execute(db state.StateDB, ctx *dict.DictionaryContext) time.Duration {
	start := time.Now()
	db.EndBlock()
	return time.Since(start)
}

// Debug prints a debug message for the end-block operation.
func (op *EndBlock) Debug(ctx *dict.DictionaryContext) {
}
