package operation

import (
	"encoding/binary"
	"fmt"
	"io"
	"time"

	"github.com/Fantom-foundation/Aida/tracer/dict"
	"github.com/Fantom-foundation/Aida/tracer/state"
)

// Endblock data structure
type EndBlock struct {
	BlockNumber uint64
}

// GetId returns the end-block operation identifier.
func (op *EndBlock) GetId() byte {
	return EndBlockID
}

// NewEndBlock creates a new end-block operation.
func NewEndBlock(number uint64) *EndBlock {
	return &EndBlock{number}
}

// ReadEndBlock reads an end-block operation from file.
func ReadEndBlock(file io.Reader) (Operation, error) {
	data := new(EndBlock)
	err := binary.Read(file, binary.LittleEndian, data)
	return data, err
}

// Write the end-block operation to file.
func (op *EndBlock) Write(f io.Writer) error {
	return binary.Write(f, binary.LittleEndian, *op)
}

// Execute the end-block operation.
func (op *EndBlock) Execute(db state.StateDB, ctx *dict.DictionaryContext) time.Duration {
	start := time.Now()
	db.EndBlock(op.BlockNumber)
	return time.Since(start)
}

// Debug prints a debug message for the end-block operation.
func (op *EndBlock) Debug(ctx *dict.DictionaryContext) {
	fmt.Print(op.BlockNumber)
}
