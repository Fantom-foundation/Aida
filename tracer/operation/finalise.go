package operation

import (
	"encoding/binary"
	"fmt"
	"io"
	"time"

	"github.com/Fantom-foundation/Aida/state"

	"github.com/Fantom-foundation/Aida/tracer/dictionary"
)

// Finalise data structure
type Finalise struct {
	DeleteEmptyObjects bool
}

// GetId returns the finalise operation identifier.
func (op *Finalise) GetId() byte {
	return FinaliseID
}

// NewFinalise creates a new finalise operation.
func NewFinalise(deleteEmptyObjects bool) *Finalise {
	return &Finalise{DeleteEmptyObjects: deleteEmptyObjects}
}

// ReadFinalise reads a finalise operation from a file.
func ReadFinalise(file io.Reader) (Operation, error) {
	data := new(Finalise)
	err := binary.Read(file, binary.LittleEndian, data)
	return data, err
}

// Write the finalise operation to a file.
func (op *Finalise) Write(f io.Writer) error {
	err := binary.Write(f, binary.LittleEndian, *op)
	return err
}

// Execute the finalise operation.
func (op *Finalise) Execute(db state.StateDB, ctx *dictionary.DictionaryContext) time.Duration {
	start := time.Now()
	db.Finalise(op.DeleteEmptyObjects)
	return time.Since(start)
}

// Debug prints a debug message for the finalise operation.
func (op *Finalise) Debug(ctx *dictionary.DictionaryContext) {
	fmt.Print(op.DeleteEmptyObjects)
}
