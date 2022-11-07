package operation

import (
	"encoding/binary"
	"fmt"
	"io"
	"time"

	"github.com/Fantom-foundation/Aida/tracer/dict"
	"github.com/Fantom-foundation/Aida/tracer/state"
)

// EndTransaction data structure
type EndTransaction struct {
}

// GetId returns the end-transaction operation identifier.
func (op *EndTransaction) GetId() byte {
	return EndTransactionID
}

// NewEndTransaction creates a new end-transaction operation.
func NewEndTransaction() *EndTransaction {
	return &EndTransaction{}
}

// ReadEndTransaction reads a new end-transaction operation from file.
func ReadEndTransaction(io.Reader) (Operation, error) {
	return new(EndTransaction), nil
}

// Write the end-transaction operation to file.
func (op *EndTransaction) Write(f io.Writer) error {
	err := binary.Write(f, binary.LittleEndian, *op)
	return err
}

// Execute the end-transaction operation.
func (op *EndTransaction) Execute(db state.StateDB, ctx *dict.DictionaryContext) time.Duration {
	ctx.InitSnapshot()
	return time.Duration(0)
}

// Debug prints a debug message for the end-transaction operation.
func (op *EndTransaction) Debug(*dict.DictionaryContext) {
	fmt.Printf("\t%s\n", operationLabels[EndTransactionID])
}
