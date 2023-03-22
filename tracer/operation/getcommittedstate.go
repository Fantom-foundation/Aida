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

// GetCommittedState data structure
type GetCommittedState struct {
	Contract common.Address
	Key      common.Hash
}

// GetId returns the get-commited-state-operation identifier.
func (op *GetCommittedState) GetId() byte {
	return GetCommittedStateID
}

// NewGetCommittedState creates a new get-commited-state operation.
func NewGetCommittedState(contract common.Address, key common.Hash) *GetCommittedState {
	return &GetCommittedState{Contract: contract, Key: key}
}

// ReadGetCommittedState reads a get-commited-state operation from file.
func ReadGetCommittedState(file io.Reader) (Operation, error) {
	data := new(GetCommittedState)
	err := binary.Read(file, binary.LittleEndian, data)
	return data, err
}

// Write the get-commited-state operation to file.
func (op *GetCommittedState) Write(f io.Writer) error {
	err := binary.Write(f, binary.LittleEndian, *op)
	return err
}

// Execute the get-committed-state operation.
func (op *GetCommittedState) Execute(db state.StateDB, ctx *dictionary.Context) time.Duration {
	contract := ctx.DecodeContract(op.Contract)
	storage := ctx.DecodeStorage(op.Key)
	start := time.Now()
	db.GetCommittedState(contract, storage)
	return time.Since(start)
}

// Debug prints debug message for the get-committed-state operation.
func (op *GetCommittedState) Debug(ctx *dictionary.Context) {
	fmt.Print(op.Contract, op.Key)
}
