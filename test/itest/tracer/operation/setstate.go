package operation

import (
	"encoding/binary"
	"fmt"
	"io"
	"time"

	"github.com/Fantom-foundation/rc-testing/test/itest/state"
	"github.com/ethereum/go-ethereum/common"

	"github.com/Fantom-foundation/rc-testing/test/itest/tracer/context"
)

// SetState data structure
type SetState struct {
	Contract common.Address // encoded contract address
	Key      common.Hash    // encoded storage address
	Value    common.Hash    // encoded storage value
}

// GetId returns the set-state identifier.
func (op *SetState) GetId() byte {
	return SetStateID
}

// NewSetState creates a new set-state operation.
func NewSetState(contract common.Address, key common.Hash, value common.Hash) *SetState {
	return &SetState{Contract: contract, Key: key, Value: value}
}

// ReadSetState reads a set-state operation from file.
func ReadSetState(f io.Reader) (Operation, error) {
	data := new(SetState)
	err := binary.Read(f, binary.LittleEndian, data)
	return data, err
}

// Write the set-state operation to file.
func (op *SetState) Write(f io.Writer) error {
	err := binary.Write(f, binary.LittleEndian, *op)
	return err
}

// Execute the set-state operation.
func (op *SetState) Execute(db state.StateDB, ctx *context.Replay) time.Duration {
	contract := ctx.DecodeContract(op.Contract)
	storage := ctx.DecodeKey(op.Key)
	value := op.Value
	start := time.Now()
	db.SetState(contract, storage, value)
	return time.Since(start)
}

// Debug prints a debug message for the set-state operation.
func (op *SetState) Debug(ctx *context.Context) {
	fmt.Print(op.Contract, op.Key, op.Value)
}
