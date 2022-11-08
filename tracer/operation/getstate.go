package operation

import (
	"encoding/binary"
	"fmt"
	"io"
	"time"

	"github.com/Fantom-foundation/Aida/tracer/dict"
	"github.com/Fantom-foundation/Aida/tracer/state"
)

// GetState data structure
type GetState struct {
	ContractIndex uint32 // encoded contract address
	StorageIndex  uint32 // encoded storage address
}

// GetId returns the get-state operation identifier.
func (op *GetState) GetId() byte {
	return GetStateID
}

// NewGetState creates a new get-state operation.
func NewGetState(cIdx uint32, sIdx uint32) *GetState {
	return &GetState{ContractIndex: cIdx, StorageIndex: sIdx}
}

// ReadGetState reads a get-state operation from a file.
func ReadGetState(file io.Reader) (Operation, error) {
	data := new(GetState)
	err := binary.Read(file, binary.LittleEndian, data)
	return data, err
}

// Write the get-state operation to file.
func (op *GetState) Write(f io.Writer) error {
	err := binary.Write(f, binary.LittleEndian, *op)
	return err
}

// Execute the get-state operation.
func (op *GetState) Execute(db state.StateDB, ctx *dict.DictionaryContext) time.Duration {
	contract := ctx.DecodeContract(op.ContractIndex)
	storage := ctx.DecodeStorage(op.StorageIndex)
	start := time.Now()
	db.GetState(contract, storage)
	return time.Since(start)
}

// Debug prints a debug message for the get-state operation.
func (op *GetState) Debug(ctx *dict.DictionaryContext) {
	fmt.Printf("\t%s: %s, %s\n", GetLabel(GetStateID), ctx.DecodeContract(op.ContractIndex), ctx.DecodeStorage(op.StorageIndex))
}
