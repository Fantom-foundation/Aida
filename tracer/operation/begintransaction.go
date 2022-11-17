package operation

import (
	"encoding/binary"
	"io"
	"time"

	"github.com/Fantom-foundation/Aida/tracer/dict"
	"github.com/Fantom-foundation/Aida/tracer/state"
)

// BeginTransaction data structure
type BeginTransaction struct {
}

// GetId returns the begin-transaction operation identifier.
func (op *BeginTransaction) GetId() byte {
	return BeginTransactionID
}

// NewBeginTransaction creates a new begin-transaction operation.
func NewBeginTransaction() *BeginTransaction {
	return &BeginTransaction{}
}

// ReadBeginTransaction reads a new begin-transaction operation from file.
func ReadBeginTransaction(io.Reader) (Operation, error) {
	return new(BeginTransaction), nil
}

// Write the begin-transaction operation to file.
func (op *BeginTransaction) Write(f io.Writer) error {
	err := binary.Write(f, binary.LittleEndian, *op)
	return err
}

// Execute the begin-transaction operation.
func (op *BeginTransaction) Execute(db state.StateDB, ctx *dict.DictionaryContext) time.Duration {
	ctx.InitSnapshot()
	start := time.Now()
	db.BeginTransaction()
	return time.Since(start)
}

// Debug prints a debug message for the begin-transaction operation.
func (op *BeginTransaction) Debug(*dict.DictionaryContext) {
}
