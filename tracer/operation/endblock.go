package operation

import (
	"os"

	"github.com/Fantom-foundation/aida/tracer/dict"
	"github.com/Fantom-foundation/substate-cli/state"
)

// End-block operation data structure
type EndBlock struct {
}

// Return the end-block operation identifier.
func (op *EndBlock) GetOpId() byte {
	return EndBlockID
}

// Create a new end-block operation.
func NewEndBlock() *EndBlock {
	return &EndBlock{}
}

// Read an end-block operation from file.
func ReadEndBlock(file *os.File) (Operation, error) {
	return NewEndBlock(), nil
}

// Write the end-block operation to file.
func (op *EndBlock) Write(f *os.File) error {
	return nil
}

// Execute the end-block operation.
func (op *EndBlock) Execute(db state.StateDB, ctx *dict.DictionaryContext) {
}

// Print a debug message for end-block.
func (op *EndBlock) Debug(ctx *dict.DictionaryContext) {
}
