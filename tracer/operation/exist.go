package operation

import (
	"encoding/binary"
	"fmt"
	"os"

	"github.com/Fantom-foundation/Aida/tracer/dict"
	"github.com/Fantom-foundation/Aida/tracer/state"
)

// Exist data structure
type Exist struct {
	ContractIndex uint32 // encoded contract address
}

// Return the exist operation identifier.
func (op *Exist) GetOpId() byte {
	return ExistID
}

// Create a new exist operation.
func NewExist(cIdx uint32) *Exist {
	return &Exist{ContractIndex: cIdx}
}

// Read a exist operation from a file.
func ReadExist(file *os.File) (Operation, error) {
	data := new(Exist)
	err := binary.Read(file, binary.LittleEndian, data)
	return data, err
}

// Write the exist operation to a file.
func (op *Exist) Write(f *os.File) error {
	err := binary.Write(f, binary.LittleEndian, *op)
	return err
}

// Execute the exist operation.
func (op *Exist) Execute(db state.StateDB, ctx *dict.DictionaryContext) {
	contract := ctx.DecodeContract(op.ContractIndex)
	db.Exist(contract)
}

// Print a debug message for exist.
func (op *Exist) Debug(ctx *dict.DictionaryContext) {
	fmt.Printf("\tcontract: %v\n", ctx.DecodeContract(op.ContractIndex))
}
