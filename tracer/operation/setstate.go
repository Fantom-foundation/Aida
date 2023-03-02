package operation

import (
	"encoding/binary"
	"fmt"
	"io"
	"time"

	"github.com/Fantom-foundation/Aida/state"
	"github.com/ethereum/go-ethereum/common"

	"github.com/Fantom-foundation/Aida/tracer/dictionary"
)

// SetState data structure
type SetState struct {
	ContractIndex uint32      // encoded contract address
	StorageIndex  uint32      // encoded storage address
	Value         common.Hash // encoded storage value
}

// GetId returns the set-state identifier.
func (op *SetState) GetId() byte {
	return SetStateID
}

// NewSetState creates a new set-state operation.
func NewSetState(cIdx uint32, sIdx uint32, v *common.Hash) *SetState {
	return &SetState{ContractIndex: cIdx, StorageIndex: sIdx, Value: *v}
}

// ReadSetState reads a set-state operation from file.
func ReadSetState(file io.Reader) (Operation, error) {
	data := new(SetState)
	err := binary.Read(file, binary.LittleEndian, data)
	return data, err
}

// Write the set-state operation to file.
func (op *SetState) Write(f io.Writer) error {
	err := binary.Write(f, binary.LittleEndian, *op)
	return err
}

// Execute the set-state operation.
func (op *SetState) Execute(db state.StateDB, ctx *dictionary.Context) time.Duration {
	contract := ctx.DecodeContract(op.ContractIndex)
	storage := ctx.DecodeStorage(op.StorageIndex)
	value := op.Value
	start := time.Now()
	db.SetState(contract, storage, value)
	return time.Since(start)
}

// Debug prints a debug message for the set-state operation.
func (op *SetState) Debug(ctx *dictionary.Context) {
	fmt.Print(ctx.DecodeContract(op.ContractIndex), ctx.DecodeStorage(op.StorageIndex), op.Value)
}
