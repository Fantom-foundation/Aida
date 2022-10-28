package operation

import (
	"os"
	"time"

	"github.com/Fantom-foundation/Aida/tracer/dict"
	"github.com/Fantom-foundation/Aida/tracer/state"
)

// Endblock data structure
type EndBlock struct {
}

// GetOpId returns the end-block operation identifier.
func (op *EndBlock) GetOpId() byte {
	return EndBlockID
}

// NewEndBlock creates a new end-block operation.
func NewEndBlock() *EndBlock {
	return &EndBlock{}
}

// ReadEndBlock reads an end-block operation from file.
func ReadEndBlock(file *os.File) (Operation, error) {
	return NewEndBlock(), nil
}

// Write the end-block operation to file.
func (op *EndBlock) Write(f *os.File) error {
	return nil
}

// Execute the end-block operation.
func (op *EndBlock) Execute(db state.StateDB, ctx *dict.DictionaryContext) time.Duration {
	return time.Duration(0)
}

// Debug prints a debug message for the end-block operation.
func (op *EndBlock) Debug(ctx *dict.DictionaryContext) {
}
