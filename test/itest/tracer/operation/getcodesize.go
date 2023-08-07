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

// GetCodeSize data structure
type GetCodeSize struct {
	Contract common.Address
}

// GetCodeSize returns the get-code-size operation identifier.
func (op *GetCodeSize) GetId() byte {
	return GetCodeSizeID
}

// NewGetCodeSize creates a new get-code-size operation.
func NewGetCodeSize(contract common.Address) *GetCodeSize {
	return &GetCodeSize{Contract: contract}
}

// ReadGetCodeSize reads a get-code-size operation from a file.
func ReadGetCodeSize(f io.Reader) (Operation, error) {
	data := new(GetCodeSize)
	err := binary.Read(f, binary.LittleEndian, data)
	return data, err
}

// Write the get-code-size operation to a file.
func (op *GetCodeSize) Write(f io.Writer) error {
	err := binary.Write(f, binary.LittleEndian, *op)
	return err
}

// Execute the get-code-size operation.
func (op *GetCodeSize) Execute(db state.StateDB, ctx *context.Replay) time.Duration {
	contract := ctx.DecodeContract(op.Contract)
	start := time.Now()
	db.GetCodeSize(contract)
	return time.Since(start)
}

// Debug prints a debug message for get-code-size.
func (op *GetCodeSize) Debug(ctx *context.Context) {
	fmt.Print(ctx.DecodeContract(op.Contract))
}
