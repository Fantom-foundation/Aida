package operation

import (
	"encoding/binary"
	"fmt"
	"io"
	"time"

	"github.com/Fantom-foundation/Aida/state"

	"github.com/Fantom-foundation/Aida/tracer/dictionary"
)

// BeginTransaction data structure
type BeginTransaction struct {
	TransactionNumber uint32 // transaction number
}

// GetId returns the begin-transaction operation identifier.
func (op *BeginTransaction) GetId() byte {
	return BeginTransactionID
}

// NewBeginTransaction creates a new begin-transaction operation.
func NewBeginTransaction(tx uint32) *BeginTransaction {
	return &BeginTransaction{tx}
}

// ReadBeginTransaction reads a new begin-transaction operation from file.
func ReadBeginTransaction(file io.Reader) (Operation, error) {
	data := new(BeginTransaction)
	err := binary.Read(file, binary.LittleEndian, data)
	return data, err
}

// Write the begin-transaction operation to file.
func (op *BeginTransaction) Write(f io.Writer) error {
	return binary.Write(f, binary.LittleEndian, *op)
}

// Execute the begin-transaction operation.
func (op *BeginTransaction) Execute(db state.StateDB, ctx *dictionary.DictionaryContext) time.Duration {
	start := time.Now()
	db.BeginTransaction(op.TransactionNumber)
	return time.Since(start)
}

// Debug prints a debug message for the begin-transaction operation.
func (op *BeginTransaction) Debug(*dictionary.DictionaryContext) {
	fmt.Print(op.TransactionNumber)
}
