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

// SetNonce data structure
type SetNonce struct {
	Contract common.Address
	Nonce    uint64 // nonce
}

// GetId returns the set-nonce operation identifier.
func (op *SetNonce) GetId() byte {
	return SetNonceID
}

// NewSetNonce creates a new set-nonce operation.
func NewSetNonce(contract common.Address, nonce uint64) *SetNonce {
	return &SetNonce{Contract: contract, Nonce: nonce}
}

// ReadSetNonce reads a set-nonce operation from a file.
func ReadSetNonce(f io.Reader) (Operation, error) {
	data := new(SetNonce)
	err := binary.Read(f, binary.LittleEndian, data)
	return data, err
}

// Write the set-nonce operation to a file.
func (op *SetNonce) Write(f io.Writer) error {
	err := binary.Write(f, binary.LittleEndian, *op)
	return err
}

// Execute the set-nonce operation.
func (op *SetNonce) Execute(db state.StateDB, ctx *context.Context) time.Duration {
	contract := ctx.DecodeContract(op.Contract)
	start := time.Now()
	db.SetNonce(contract, op.Nonce)
	return time.Since(start)
}

// Debug prints a debug message for the set-nonce operation.
func (op *SetNonce) Debug(ctx *context.Context) {
	fmt.Print(op.Contract, op.Nonce)
}
