package operation

import (
	"encoding/binary"
	"fmt"
	"os"

	"github.com/Fantom-foundation/aida/tracer/dict"
	"github.com/Fantom-foundation/substate-cli/state"
)

////////////////////////////////////////////////////////////
// SetState Operation
////////////////////////////////////////////////////////////

// Set-state data structure
type SetState struct {
	ContractIndex uint32 // encoded contract address
	StorageIndex  uint32 // encoded storage address
	ValueIndex    uint64 // encoded storage value
}

// Return the set-state identifier
func (op *SetState) GetOpId() byte {
	return SetStateID
}

// Create a new set-state operation.
func NewSetState(cIdx uint32, sIdx uint32, vIdx uint64) *SetState {
	return &SetState{ContractIndex: cIdx, StorageIndex: sIdx, ValueIndex: vIdx}
}

// Read a set-state operation from file.
func ReadSetState(file *os.File) (Operation, error) {
	data := new(SetState)
	err := binary.Read(file, binary.LittleEndian, data)
	return data, err
}

// Write the set-state operation to file.
func (op *SetState) writeOperation(f *os.File) {
	//var data = []any{op.ContractIndex, op.StorageIndex, op.ValueIndex}
	writeStruct(f, op)
}

// Execute the set-state operation.
func (op *SetState) Execute(db state.StateDB, ctx *dict.DictionaryContext) {
	contract := ctx.DecodeContract(op.ContractIndex)
	storage := ctx.DecodeStorage(op.StorageIndex)
	value := ctx.DecodeValue(op.ValueIndex)
	db.SetState(contract, storage, value)
}

// Print a debug message for set-state.
func (op *SetState) Debug(ctx *dict.DictionaryContext) {
	fmt.Printf("\tcontract: %v\t storage: %v\t value: %v\n",
		ctx.DecodeContract(op.ContractIndex),
		ctx.DecodeStorage(op.StorageIndex),
		ctx.DecodeValue(op.ValueIndex))
}
