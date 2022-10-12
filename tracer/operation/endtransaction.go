package operation

import (
	"encoding/binary"
	"os"

	"github.com/Fantom-foundation/aida/tracer/dict"
	"github.com/Fantom-foundation/substate-cli/state"
)

// End-transaction operation's data structure
type EndTransaction struct {
}

// Return the end-transaction operation identifier.
func (op *EndTransaction) GetOpId() byte {
	return EndTransactionID
}

// Create a new end-transaction operation.
func NewEndTransaction() *EndTransaction {
	return &EndTransaction{}
}

// Read a new end-transaction operation from file.
func ReadEndTransaction(*os.File) (Operation, error) {
	return new(EndTransaction), nil
}

// Write the end-transaction operation to file.
func (op *EndTransaction) Write(f *os.File) error {
	err := binary.Write(f, binary.LittleEndian, *op)
	return err
}

// Execute the end-transaction operation.
func (op *EndTransaction) Execute(db state.StateDB, ctx *dict.DictionaryContext) {
}

// Print a debug message for end-transaction.
func (op *EndTransaction) Debug(*dict.DictionaryContext) {
}
