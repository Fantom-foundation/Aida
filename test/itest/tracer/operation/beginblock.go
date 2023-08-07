package operation

import (
	"encoding/binary"
	"fmt"
	"io"
	"time"

	"github.com/Fantom-foundation/rc-testing/test/itest/state"

	"github.com/Fantom-foundation/rc-testing/test/itest/tracer/context"
)

// Begin-block operation data structure
type BeginBlock struct {
	BlockNumber uint64 // block number
}

// GetId returns the begin-block operation identifier.
func (op *BeginBlock) GetId() byte {
	return BeginBlockID
}

// NewBeginBlock creates a new begin-block operation.
func NewBeginBlock(bbNum uint64) *BeginBlock {
	return &BeginBlock{BlockNumber: bbNum}
}

// ReadBeginBlock reads a begin-block operation from file.
func ReadBeginBlock(f io.Reader) (Operation, error) {
	data := new(BeginBlock)
	err := binary.Read(f, binary.LittleEndian, data)
	return data, err
}

// Write the begin-block operation to file.
func (op *BeginBlock) Write(f io.Writer) error {
	err := binary.Write(f, binary.LittleEndian, *op)
	return err
}

// Execute the begin-block operation.
func (op *BeginBlock) Execute(db state.StateDB, ctx *context.Replay) time.Duration {
	start := time.Now()
	db.BeginBlock(op.BlockNumber)
	return time.Since(start)
}

// Debug prints a debug message for the begin-block operation.
func (op *BeginBlock) Debug(ctx *context.Context) {
	fmt.Print(op.BlockNumber)
}
