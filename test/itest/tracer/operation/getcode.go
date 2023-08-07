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

// GetCode data structure
type GetCode struct {
	Contract common.Address
}

// GetId returns the get-code operation identifier.
func (op *GetCode) GetId() byte {
	return GetCodeID
}

// NewGetCode creates a new get-code operation.
func NewGetCode(contract common.Address) *GetCode {
	return &GetCode{Contract: contract}
}

// ReadGetCode reads a get-code operation from a file.
func ReadGetCode(f io.Reader) (Operation, error) {
	data := new(GetCode)
	err := binary.Read(f, binary.LittleEndian, data)
	return data, err
}

// Write the get-code operation to a file.
func (op *GetCode) Write(f io.Writer) error {
	err := binary.Write(f, binary.LittleEndian, *op)
	return err
}

// Execute the get-code operation.
func (op *GetCode) Execute(db state.StateDB, ctx *context.Replay) time.Duration {
	contract := ctx.DecodeContract(op.Contract)
	start := time.Now()
	db.GetCode(contract)
	return time.Since(start)
}

// Debug prints a debug message for the get-code operation.
func (op *GetCode) Debug(ctx *context.Context) {
	fmt.Print(op.Contract)
}
