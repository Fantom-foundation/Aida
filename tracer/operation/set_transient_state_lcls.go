package operation

import (
	"encoding/binary"
	"fmt"
	"io"
	"time"

	"github.com/Fantom-foundation/Aida/state"
	"github.com/Fantom-foundation/Aida/tracer/context"
	"github.com/ethereum/go-ethereum/common"
)

// SetTransientStateLcls data structure
type SetTransientStateLcls struct {
	Value common.Hash // encoded storage value
}

// GetId returns the set-state-lcls identifier.
func (op *SetTransientStateLcls) GetId() byte {
	return SetTransientStateLclsID
}

// SetTransientStateLcls creates a new set-state-lcls operation.
func NewSetTransientStateLcls(value common.Hash) *SetTransientStateLcls {
	return &SetTransientStateLcls{Value: value}
}

// ReadSetTransientStateLcls reads a set-state-lcls operation from file.
func ReadSetTransientStateLcls(f io.Reader) (Operation, error) {
	data := new(SetTransientStateLcls)
	err := binary.Read(f, binary.LittleEndian, data)
	return data, err
}

// Write the set-state-lcls operation to file.
func (op *SetTransientStateLcls) Write(f io.Writer) error {
	err := binary.Write(f, binary.LittleEndian, *op)
	return err
}

// Execute the set-state-lcls operation.
func (op *SetTransientStateLcls) Execute(db state.StateDB, ctx *context.Replay) time.Duration {
	contract := ctx.PrevContract()
	storage := ctx.DecodeKeyCache(0)
	start := time.Now()
	db.SetTransientState(contract, storage, op.Value)
	return time.Since(start)
}

// Debug prints a debug message for the set-state-lcls operation.
func (op *SetTransientStateLcls) Debug(ctx *context.Context) {
	contract := ctx.PrevContract()
	storage := ctx.ReadKeyCache(0)
	fmt.Print(contract, storage, op.Value)
}
