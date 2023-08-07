package operation

import (
	"encoding/binary"
	"io"
	"time"

	"github.com/Fantom-foundation/rc-testing/test/itest/state"

	"github.com/Fantom-foundation/rc-testing/test/itest/tracer/context"
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
func (op *EndTransaction) Execute(db state.StateDB, ctx *context.Replay) time.Duration {
	ctx.InitSnapshot()
	start := time.Now()
	db.EndTransaction()
	return time.Since(start)
}

// Debug prints a debug message for the end-transaction operation.
func (op *EndTransaction) Debug(*context.Context) {
}
