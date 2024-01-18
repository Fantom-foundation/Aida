package operation

import (
	"encoding/binary"
	"fmt"
	"io"
	"time"

	"github.com/Fantom-foundation/Aida/state"

	"github.com/Fantom-foundation/Aida/tracer/context"
)

// BeginTransaction data structure
type BeginTransaction struct {
	TransactionNumber uint32 // transaction number
}

// GetId returns the begin-transactionoperation identifier.
func (op *BeginTransaction) GetId() byte {
	return BeginTransactionID
}

// NewBeginTransaction creates a new begin-transaction operation.
func NewBeginTransaction(tx uint32) *BeginTransaction {
	return &BeginTransaction{tx}
}

// ReadBeginTransaction reads a new begin-transaction operation from file.
func ReadBeginTransaction(f io.Reader) (Operation, error) {
	data := new(BeginTransaction)
	err := binary.Read(f, binary.LittleEndian, data)
	return data, err
}

// Write the begin-transaction operation to file.
func (op *BeginTransaction) Write(f io.Writer) error {
	return binary.Write(f, binary.LittleEndian, *op)
}

// Execute the begin-transaction operation.
func (op *BeginTransaction) Execute(db state.StateDB, ctx *context.Replay) time.Duration {
	start := time.Now()
	db.BeginTransaction(op.TransactionNumber)
	return time.Since(start)
}

// Debug prints a debug message for the begin-transaction operation.
func (op *BeginTransaction) Debug(*context.Context) {
	fmt.Print(op.TransactionNumber)
}
