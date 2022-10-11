package operation

import (
	"encoding/binary"
	"fmt"
	"os"

	"github.com/Fantom-foundation/aida/tracer/dict"
	"github.com/Fantom-foundation/substate-cli/state"
)

////////////////////////////////////////////////////////////
// GetBalance Operation
////////////////////////////////////////////////////////////

// GetBalance data structure
type GetBalance struct {
	ContractIndex uint32
}

// Return the get-balance operation identifier.
func (op *GetBalance) GetOpId() byte {
	return GetBalanceID
}

// Create a new get-balance operation.
func NewGetBalance(cIdx uint32) *GetBalance {
	return &GetBalance{ContractIndex: cIdx}
}

// Read a get-balance operation from a file.
func ReadGetBalance(file *os.File) (Operation, error) {
	data := new(GetBalance)
	err := binary.Read(file, binary.LittleEndian, data)
	return data, err
}

// Write the get-balance operation.
func (op *GetBalance) writeOperation(f *os.File) {
	//var data = []any{op.ContractIndex}
	writeStruct(f, op)
}

// Execute the get-balance operation.
func (op *GetBalance) Execute(db state.StateDB, ctx *dict.DictionaryContext) {
	contract := ctx.DecodeContract(op.ContractIndex)
	db.GetBalance(contract)
}

// Print a debug message for get-balance.
func (op *GetBalance) Debug(ctx *dict.DictionaryContext) {
	fmt.Printf("\tcontract: %v\n", ctx.DecodeContract(op.ContractIndex))
}
