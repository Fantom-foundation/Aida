package operation

import (
	"encoding/binary"
	"fmt"
	"log"
	"os"

	"github.com/Fantom-foundation/aida/tracer/dict"
	"github.com/Fantom-foundation/substate-cli/state"
)

////////////////////////////////////////////////////////////
// Begin Block Operation (Pseudo Operation)
////////////////////////////////////////////////////////////

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
func (op *BeginBlock) writeOperation(f *os.File) {
	if err := binary.Write(f, binary.LittleEndian, *op); err != nil {
		log.Fatalf("Failed to write binary data: %v", err)
	}
}

// Execute the begin-block operation.
func (op *BeginBlock) Execute(db state.StateDB, ctx *dict.DictionaryContext) {
}

// Print a debug message for begin-block.
func (op *BeginBlock) Debug(ctx *dict.DictionaryContext) {
	fmt.Printf("\tblock number: %v\n", op.BlockNumber)
}
