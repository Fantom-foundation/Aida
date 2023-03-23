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

// HasSuicided data structure
type HasSuicided struct {
	Contract common.Address
}

// GetId returns the HasSuicided operation identifier.
func (op *HasSuicided) GetId() byte {
	return HasSuicidedID
}

// NewHasSuicided creates a new HasSuicided operation.
func NewHasSuicided(contract common.Address) *HasSuicided {
	return &HasSuicided{Contract: contract}
}

// ReadHasSuicided reads a HasSuicided operation from a file.
func ReadHasSuicided(f io.Reader) (Operation, error) {
	data := new(HasSuicided)
	err := binary.Read(f, binary.LittleEndian, data)
	return data, err
}

// Write the HasSuicided operation to a file.
func (op *HasSuicided) Write(f io.Writer) error {
	err := binary.Write(f, binary.LittleEndian, *op)
	return err
}

// Execute the HasSuicided operation.
func (op *HasSuicided) Execute(db state.StateDB, ctx *context.Context) time.Duration {
	contract := ctx.DecodeContract(op.Contract)
	start := time.Now()
	db.HasSuicided(contract)
	return time.Since(start)
}

// Debug prints a debug message for the HasSuicided operation.
func (op *HasSuicided) Debug(ctx *context.Context) {
	fmt.Print(op.Contract)
}
