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

// Empty data structure
type Empty struct {
	Contract common.Address
}

// GetId returns the Empty operation identifier.
func (op *Empty) GetId() byte {
	return EmptyID
}

// NewEmpty creates a new Empty operation.
func NewEmpty(contract common.Address) *Empty {
	return &Empty{Contract: contract}
}

// ReadEmpty reads an Empty operation from a file.
func ReadEmpty(f io.Reader) (Operation, error) {
	data := new(Empty)
	err := binary.Read(f, binary.LittleEndian, data)
	return data, err
}

// Write the Empty operation to a file.
func (op *Empty) Write(f io.Writer) error {
	err := binary.Write(f, binary.LittleEndian, *op)
	return err
}

// Execute the Empty operation.
func (op *Empty) Execute(db state.StateDB, ctx *context.Context) time.Duration {
	contract := ctx.DecodeContract(op.Contract)
	start := time.Now()
	db.Empty(contract)
	return time.Since(start)
}

// Debug prints a debug message for the Empty operation.
func (op *Empty) Debug(ctx *context.Context) {
	fmt.Print(op.Contract)
}
