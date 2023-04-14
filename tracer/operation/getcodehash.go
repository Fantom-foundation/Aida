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

// GetCodeHash data structure
type GetCodeHash struct {
	Contract common.Address
}

// GetId returns the get-code-hash operation identifier.
func (op *GetCodeHash) GetId() byte {
	return GetCodeHashID
}

// NewGetCodeHash creates a new get-code-hash operation.
func NewGetCodeHash(contract common.Address) *GetCodeHash {
	return &GetCodeHash{Contract: contract}
}

// ReadGetHash reads a get-code-hash operation from a file.
func ReadGetCodeHash(f io.Reader) (Operation, error) {
	data := new(GetCodeHash)
	err := binary.Read(f, binary.LittleEndian, data)
	return data, err
}

// Write the get-code-hash operation to a file.
func (op *GetCodeHash) Write(f io.Writer) error {
	err := binary.Write(f, binary.LittleEndian, *op)
	return err
}

// Execute the get-code-hash operation.
func (op *GetCodeHash) Execute(db state.StateDB, ctx *context.Replay) time.Duration {
	contract := ctx.DecodeContract(op.Contract)
	start := time.Now()
	db.GetCodeHash(contract)
	return time.Since(start)
}

// Debug prints a debug message for the get-code-hash operation.
func (op *GetCodeHash) Debug(ctx *context.Context) {
	fmt.Print(op.Contract)
}
