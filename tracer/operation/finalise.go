package operation

import (
	"encoding/binary"
	"fmt"
	"io"
	"time"

	"github.com/Fantom-foundation/Aida/tracer/dict"
	"github.com/Fantom-foundation/Aida/tracer/state"
)

// Finalise data structure
type Finalise struct {
	DeleteEmptyObjects bool
}

// GetOpId returns the finalise operation identifier.
func (op *Finalise) GetOpId() byte {
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
func (op *Finalise) Execute(db state.StateDB, ctx *dict.DictionaryContext) time.Duration {
	start := time.Now()
	db.Finalise(op.DeleteEmptyObjects)
	return time.Since(start)
}

// Debug prints a debug message for the finalise operation.
func (op *Finalise) Debug(ctx *dict.DictionaryContext) {
	fmt.Printf("\tdelete empty objects: %v\n", op.DeleteEmptyObjects)
}
