package operation

import (
	"encoding/binary"
	"fmt"
	"os"

	"github.com/Fantom-foundation/Aida/tracer/dict"
	"github.com/Fantom-foundation/Aida/tracer/state"
)

// Finalise data structure
type Finalise struct {
	DeleteEmptyObjects bool
}

// Return the finalise operation identifier.
func (op *Finalise) GetOpId() byte {
	return FinaliseID
}

// Create a new finalise operation.
func NewFinalise(deleteEmptyObjects bool) *Finalise {
	return &Finalise{DeleteEmptyObjects: deleteEmptyObjects}
}

// Read a finalise operation from a file.
func ReadFinalise(file *os.File) (Operation, error) {
	data := new(Finalise)
	err := binary.Read(file, binary.LittleEndian, data)
	return data, err
}

// Write the finalise operation to a file.
func (op *Finalise) Write(f *os.File) error {
	err := binary.Write(f, binary.LittleEndian, *op)
	return err
}

// Execute the finalise operation.
func (op *Finalise) Execute(db state.StateDB, ctx *dict.DictionaryContext) {
	db.Finalise(op.DeleteEmptyObjects)
}

// Print a debug message for finalise.
func (op *Finalise) Debug(ctx *dict.DictionaryContext) {
	fmt.Printf("\tdelete empty objects: %v\n", op.DeleteEmptyObjects)
}
