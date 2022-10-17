package operation

import (
	"encoding/binary"
	"fmt"
	"os"

	"github.com/Fantom-foundation/Aida-Testing/tracer/dict"
	"github.com/Fantom-foundation/Aida-Testing/tracer/state"
)

// Suicide data structure
type Suicide struct {
	ContractIndex uint32 // encoded contract address
}

// Return the suicide operation identifier.
func (op *Suicide) GetOpId() byte {
	return SuicideID
}

// Create a new suicide operation.
func NewSuicide(cIdx uint32) *Suicide {
	return &Suicide{ContractIndex: cIdx}
}

// Read a suicide operation from a file.
func ReadSuicide(file *os.File) (Operation, error) {
	data := new(Suicide)
	err := binary.Read(file, binary.LittleEndian, data)
	return data, err
}

// Write the suicide operation to a file.
func (op *Suicide) Write(f *os.File) error {
	err := binary.Write(f, binary.LittleEndian, *op)
	return err
}

// Execute the suicide operation.
func (op *Suicide) Execute(db state.StateDB, ctx *dict.DictionaryContext) {
	contract := ctx.DecodeContract(op.ContractIndex)
	db.Suicide(contract)
}

// Print a debug message for suicide.
func (op *Suicide) Debug(ctx *dict.DictionaryContext) {
	fmt.Printf("\tcontract: %v\n", ctx.DecodeContract(op.ContractIndex))
}
