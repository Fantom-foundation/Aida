package operation

import (
	"encoding/binary"
	"fmt"
	"os"

	"github.com/Fantom-foundation/aida/tracer/dict"
	"github.com/Fantom-foundation/substate-cli/state"
)

////////////////////////////////////////////////////////////
// Finalise Operation
////////////////////////////////////////////////////////////

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
func (op *Finalise) writeOperation(f *os.File) {
	//var data = []any{op.DeleteEmptyObjects}
	writeStruct(f, op)
}

// Execute the finalise operation.
func (op *Finalise) Execute(db state.StateDB, ctx *dict.DictionaryContext) {
	db.Finalise(op.DeleteEmptyObjects)
}

// Print a debug message for finalise.
func (op *Finalise) Debug(ctx *dict.DictionaryContext) {
	fmt.Printf("\tdelete empty objects: %v\n", op.DeleteEmptyObjects)
}
