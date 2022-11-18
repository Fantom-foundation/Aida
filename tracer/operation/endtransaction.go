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
	TransactionNumber uint32 // transaction number
}

// GetId returns the end-transaction operation identifier.
func (op *EndTransaction) GetId() byte {
	return EndTransactionID
}

// NewEndTransaction creates a new end-transaction operation.
func NewEndTransaction(tx uint32) *EndTransaction {
	return &EndTransaction{tx}
}

// ReadEndTransaction reads a new end-transaction operation from file.
func ReadEndTransaction(file io.Reader) (Operation, error) {
	data := new(EndTransaction)
	err := binary.Read(file, binary.LittleEndian, data)
	return data, err
}

// Write the end-transaction operation to file.
func (op *EndTransaction) Write(f io.Writer) error {
	return binary.Write(f, binary.LittleEndian, *op)
}

// Execute the end-transaction operation.
func (op *EndTransaction) Execute(db state.StateDB, ctx *dict.DictionaryContext) time.Duration {
	start := time.Now()
	db.EndTransaction(op.TransactionNumber)
	return time.Since(start)
}

// Debug prints a debug message for the end-transaction operation.
func (op *EndTransaction) Debug(*dict.DictionaryContext) {
	fmt.Print(op.TransactionNumber)
}
