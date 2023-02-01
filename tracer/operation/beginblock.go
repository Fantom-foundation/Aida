package operation

import (
	"encoding/binary"
	"fmt"
	"github.com/Fantom-foundation/Aida/state"
	"io"
	"time"

	"github.com/Fantom-foundation/Aida/tracer/dict"
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
func ReadBeginBlock(file io.Reader) (Operation, error) {
	data := new(BeginBlock)
	err := binary.Read(file, binary.LittleEndian, data)
	return data, err
}

// Write the begin-block operation to file.
func (op *BeginBlock) Write(f io.Writer) error {
	err := binary.Write(f, binary.LittleEndian, *op)
	return err
}

// Execute the begin-block operation.
func (op *BeginBlock) Execute(db state.StateDB, ctx *dict.DictionaryContext) time.Duration {
	start := time.Now()
	db.BeginBlock(op.BlockNumber)
	return time.Since(start)
}

// Debug prints a debug message for the begin-block operation.
func (op *BeginBlock) Debug(ctx *dict.DictionaryContext) {
	fmt.Print(op.BlockNumber)
}
