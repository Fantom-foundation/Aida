package operation

import (
	"encoding/binary"
	"fmt"
	"os"

	"github.com/Fantom-foundation/aida/tracer/dict"
	"github.com/Fantom-foundation/aida/tracer/state"
)

// Get-code-hash data structure
type GetCodeHash struct {
	ContractIndex uint32 // encoded contract address
}

// Return the get-code-hash operation identifier.
func (op *GetCodeHash) GetOpId() byte {
	return GetCodeHashID
}

// Create a new get-code-hash operation.
func NewGetCodeHash(cIdx uint32) *GetCodeHash {
	return &GetCodeHash{ContractIndex: cIdx}
}

// Read a get-code-hash operation from a file.
func ReadGetCodeHash(file *os.File) (Operation, error) {
	data := new(GetCodeHash)
	err := binary.Read(file, binary.LittleEndian, data)
	return data, err
}

// Write the get-code-hash operation to a file.
func (op *GetCodeHash) Write(f *os.File) error {
	err := binary.Write(f, binary.LittleEndian, *op)
	return err
}

// Execute the get-code-hash operation.
func (op *GetCodeHash) Execute(db state.StateDB, ctx *dict.DictionaryContext) {
	contract := ctx.DecodeContract(op.ContractIndex)
	db.GetCodeHash(contract)
}

// Print a debug message for get-code-hash.
func (op *GetCodeHash) Debug(ctx *dict.DictionaryContext) {
	fmt.Printf("\tcontract: %v\n", ctx.DecodeContract(op.ContractIndex))
}
