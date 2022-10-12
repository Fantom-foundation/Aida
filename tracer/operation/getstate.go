package operation

import (
	"encoding/binary"
	"fmt"
	"os"

	"github.com/Fantom-foundation/aida/tracer/dict"
	"github.com/Fantom-foundation/substate-cli/state"
)

// Get-state data structure
type GetState struct {
	ContractIndex uint32 // encoded contract address
	StorageIndex  uint32 // encoded storage address
}

// Return the get-state operation identifier.
func (op *GetState) GetOpId() byte {
	return GetStateID
}

// Create a new get-state operation.
func NewGetState(cIdx uint32, sIdx uint32) *GetState {
	return &GetState{ContractIndex: cIdx, StorageIndex: sIdx}
}

// Read a get-state operation from a file.
func ReadGetState(file *os.File) (Operation, error) {
	data := new(GetState)
	err := binary.Read(file, binary.LittleEndian, data)
	return data, err
}

// Write the get-state operation to file.
func (op *GetState) Write(f *os.File) error {
	err := binary.Write(f, binary.LittleEndian, *op)
	return err
}

// Execute the get-state operation.
func (op *GetState) Execute(db state.StateDB, ctx *dict.DictionaryContext) {
	contract := ctx.DecodeContract(op.ContractIndex)
	storage := ctx.DecodeStorage(op.StorageIndex)
	db.GetState(contract, storage)
}

// Print a debug message.
func (op *GetState) Debug(ctx *dict.DictionaryContext) {
	fmt.Printf("\tcontract: %v\t storage: %v\n",
		ctx.DecodeContract(op.ContractIndex),
		ctx.DecodeStorage(op.StorageIndex))
}
