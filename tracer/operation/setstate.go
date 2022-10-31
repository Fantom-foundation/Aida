package operation

import (
	"encoding/binary"
	"fmt"
	"os"
	"time"

	"github.com/Fantom-foundation/Aida/tracer/dict"
	"github.com/Fantom-foundation/Aida/tracer/state"
)

// SetState data structure
type SetState struct {
	ContractIndex uint32 // encoded contract address
	StorageIndex  uint32 // encoded storage address
	ValueIndex    uint64 // encoded storage value
}

// GetId returns the set-state identifier.
func (op *SetState) GetId() byte {
	return SetStateID
}

// NewSetState creates a new set-state operation.
func NewSetState(cIdx uint32, sIdx uint32, vIdx uint64) *SetState {
	return &SetState{ContractIndex: cIdx, StorageIndex: sIdx, ValueIndex: vIdx}
}

// ReadSetState reads a set-state operation from file.
func ReadSetState(file *os.File) (Operation, error) {
	data := new(SetState)
	err := binary.Read(file, binary.LittleEndian, data)
	return data, err
}

// Write the set-state operation to file.
func (op *SetState) Write(f *os.File) error {
	err := binary.Write(f, binary.LittleEndian, *op)
	return err
}

// Execute the set-state operation.
func (op *SetState) Execute(db state.StateDB, ctx *dict.DictionaryContext) time.Duration {
	contract := ctx.DecodeContract(op.ContractIndex)
	storage := ctx.DecodeStorage(op.StorageIndex)
	value := ctx.DecodeValue(op.ValueIndex)
	start := time.Now()
	db.SetState(contract, storage, value)
	return time.Since(start)
}

// Debug prints a debug message for the set-state operation.
func (op *SetState) Debug(ctx *dict.DictionaryContext) {
	fmt.Printf("\tcontract: %v\t storage: %v\t value: %v\n",
		ctx.DecodeContract(op.ContractIndex),
		ctx.DecodeStorage(op.StorageIndex),
		ctx.DecodeValue(op.ValueIndex))
}
