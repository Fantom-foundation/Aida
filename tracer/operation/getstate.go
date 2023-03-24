package operation

import (
	"encoding/binary"
	"fmt"
	"io"
	"time"

	"github.com/Fantom-foundation/Aida/state"
	"github.com/ethereum/go-ethereum/common"

	"github.com/Fantom-foundation/Aida/tracer/context"
)

// GetState data structure
type GetState struct {
	Contract common.Address
	Key      common.Hash
}

// GetId returns the get-state operation identifier.
func (op *GetState) GetId() byte {
	return GetStateID
}

// NewGetState creates a new get-state operation.
func NewGetState(contract common.Address, key common.Hash) *GetState {
	return &GetState{Contract: contract, Key: key}
}

// ReadGetState reads a get-state operation from a file.
func ReadGetState(f io.Reader) (Operation, error) {
	data := new(GetState)
	err := binary.Read(f, binary.LittleEndian, data)
	return data, err
}

// Write the get-state operation to file.
func (op *GetState) Write(f io.Writer) error {
	err := binary.Write(f, binary.LittleEndian, *op)
	return err
}

// Execute the get-state operation.
func (op *GetState) Execute(db state.StateDB, ctx *context.Context) time.Duration {
	contract := ctx.DecodeContract(op.Contract)
	storage := ctx.DecodeKey(op.Key)
	start := time.Now()
	db.GetState(contract, storage)
	return time.Since(start)
}

// Debug prints a debug message for the get-state operation.
func (op *GetState) Debug(ctx *context.Context) {
	fmt.Print(op.Contract, op.Key)
}
