package operation

import (
	"encoding/binary"
	"fmt"
	"os"

	"github.com/Fantom-foundation/Aida-Testing/tracer/dict"
	"github.com/Fantom-foundation/Aida-Testing/tracer/state"
)

// Get-committed-state data structure
type GetCommittedState struct {
	ContractIndex uint32 // encoded contract address
	StorageIndex  uint32 // encoded storage address
}

// Return the get-commited-state-operation identifier.
func (op *GetCommittedState) GetOpId() byte {
	return GetCommittedStateID
}

// Create a new get-commited-state operation.
func NewGetCommittedState(cIdx uint32, sIdx uint32) *GetCommittedState {
	return &GetCommittedState{ContractIndex: cIdx, StorageIndex: sIdx}
}

// Read a get-commited-state operation from file.
func ReadGetCommittedState(file *os.File) (Operation, error) {
	data := new(GetCommittedState)
	err := binary.Read(file, binary.LittleEndian, data)
	return data, err
}

// Write the get-commited-state operation to file.
func (op *GetCommittedState) Write(f *os.File) error {
	err := binary.Write(f, binary.LittleEndian, *op)
	return err
}

// Execute the get-committed-state operation.
func (op *GetCommittedState) Execute(db state.StateDB, ctx *dict.DictionaryContext) {
	contract := ctx.DecodeContract(op.ContractIndex)
	storage := ctx.DecodeStorage(op.StorageIndex)
	db.GetCommittedState(contract, storage)
}

// Print debug message for get-committed-state.
func (op *GetCommittedState) Debug(ctx *dict.DictionaryContext) {
	fmt.Printf("\tcontract: %v\t storage: %v\n",
		ctx.DecodeContract(op.ContractIndex),
		ctx.DecodeStorage(op.StorageIndex))
}
