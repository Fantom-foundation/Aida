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

// Suicide data structure
type Suicide struct {
	Contract common.Address
}

// GetId returns the suicide operation identifier.
func (op *Suicide) GetId() byte {
	return SuicideID
}

// NewSuicide creates a new suicide operation.
func NewSuicide(contract common.Address) *Suicide {
	return &Suicide{Contract: contract}
}

// ReadSuicide reads a suicide operation from a file.
func ReadSuicide(f io.Reader) (Operation, error) {
	data := new(Suicide)
	err := binary.Read(f, binary.LittleEndian, data)
	return data, err
}

// Write the suicide operation to a file.
func (op *Suicide) Write(f io.Writer) error {
	err := binary.Write(f, binary.LittleEndian, *op)
	return err
}

// Execute the suicide operation.
func (op *Suicide) Execute(db state.StateDB, ctx *context.Replay) time.Duration {
	contract := ctx.DecodeContract(op.Contract)
	start := time.Now()
	db.Suicide(contract)
	return time.Since(start)
}

// Debug prints a debug message for the suicide operation.
func (op *Suicide) Debug(ctx *context.Context) {
	fmt.Print(ctx.DecodeContract(op.Contract))
}
