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

// GetNonce data structure
type GetNonce struct {
	Contract common.Address
}

// GetId returns the get-nonce operation identifier.
func (op *GetNonce) GetId() byte {
	return GetNonceID
}

// NewGetNonce creates a new get-nonce operation.
func NewGetNonce(contract common.Address) *GetNonce {
	return &GetNonce{Contract: contract}
}

// ReadGetNonce reads a get-nonce operation from a file.
func ReadGetNonce(f io.Reader) (Operation, error) {
	data := new(GetNonce)
	err := binary.Read(f, binary.LittleEndian, data)
	return data, err
}

// Write the get-nonce operation to a file.
func (op *GetNonce) Write(f io.Writer) error {
	err := binary.Write(f, binary.LittleEndian, *op)
	return err
}

// Execute the get-nonce operation.
func (op *GetNonce) Execute(db state.StateDB, ctx *context.Context) time.Duration {
	contract := ctx.DecodeContract(op.Contract)
	start := time.Now()
	db.GetNonce(contract)
	return time.Since(start)
}

// Debug prints a debug message for the get-nonce operation.
func (op *GetNonce) Debug(ctx *context.Context) {
	fmt.Print(op.Contract)
}
