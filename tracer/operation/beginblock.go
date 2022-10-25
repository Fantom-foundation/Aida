package operation

import (
	"encoding/binary"
	"fmt"
	"os"
	"time"

	"github.com/Fantom-foundation/Aida/tracer/dict"
	"github.com/Fantom-foundation/Aida/tracer/state"
)

// Begin-block operation data structure
type BeginBlock struct {
	BlockNumber uint64 // block number
}

// Return the begin-block operation identifier.
func (op *BeginBlock) GetOpId() byte {
	return BeginBlockID
}

// Create a new begin-block operation.
func NewBeginBlock(bbNum uint64) *BeginBlock {
	return &BeginBlock{BlockNumber: bbNum}
}

// Read a begin-block operation from file.
func ReadBeginBlock(file *os.File) (Operation, error) {
	data := new(BeginBlock)
	err := binary.Read(file, binary.LittleEndian, data)
	return data, err
}

// Write the begin-block operation to file.
func (op *BeginBlock) Write(f *os.File) error {
	err := binary.Write(f, binary.LittleEndian, *op)
	return err
}

// Execute the begin-block operation.
func (op *BeginBlock) Execute(db state.StateDB, ctx *dict.DictionaryContext) time.Duration {
	ctx.ClearIndexCaches()
	return time.Duration(0)
}

// Print a debug message for begin-block.
func (op *BeginBlock) Debug(ctx *dict.DictionaryContext) {
	fmt.Printf("\tblock number: %v\n", op.BlockNumber)
}
