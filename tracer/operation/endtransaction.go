package operation

import (
	"encoding/binary"
	"io"
	"time"

	"github.com/Fantom-foundation/Aida/state"

	"github.com/Fantom-foundation/Aida/tracer/context"
)

// EndTransaction data structure
type EndTransaction struct {
}

// GetId returns the end-transactionoperation identifier.
func (op *EndTransaction) GetId() byte {
	return EndTransactionID
}

// NewEndTransaction creates a new end-transactionoperation.
func NewEndTransaction() *EndTransaction {
	return &EndTransaction{}
}

// ReadEndTransaction reads a new end-transactionoperation from file.
func ReadEndTransaction(io.Reader) (Operation, error) {
	return new(EndTransaction), nil
}

// Write the end-transactionoperation to file.
func (op *EndTransaction) Write(f io.Writer) error {
	err := binary.Write(f, binary.LittleEndian, *op)
	return err
}

// Execute the end-transactionoperation.
func (op *EndTransaction) Execute(db state.StateDB, ctx *context.Replay) time.Duration {
	ctx.InitSnapshot()
	start := time.Now()
	db.EndTransaction()
	return time.Since(start)
}

// Debug prints a debug message for the end-transactionoperation.
func (op *EndTransaction) Debug(*context.Context) {
}
