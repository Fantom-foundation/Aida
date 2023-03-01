package operation

import (
	"encoding/binary"
	"fmt"
	"io"
	"time"

	"github.com/Fantom-foundation/Aida/state"

	"github.com/Fantom-foundation/Aida/tracer/dictionary"
)

// Empty data structure
type Empty struct {
	ContractIndex uint32 // encoded contract address
}

// GetId returns the Empty operation identifier.
func (op *Empty) GetId() byte {
	return EmptyID
}

// NewEmpty creates a new Empty operation.
func NewEmpty(cIdx uint32) *Empty {
	return &Empty{ContractIndex: cIdx}
}

// ReadEmpty reads an Empty operation from a file.
func ReadEmpty(file io.Reader) (Operation, error) {
	data := new(Empty)
	err := binary.Read(file, binary.LittleEndian, data)
	return data, err
}

// Write the Empty operation to a file.
func (op *Empty) Write(f io.Writer) error {
	err := binary.Write(f, binary.LittleEndian, *op)
	return err
}

// Execute the Empty operation.
func (op *Empty) Execute(db state.StateDB, ctx *dict.DictionaryContext) time.Duration {
	contract := ctx.DecodeContract(op.ContractIndex)
	start := time.Now()
	db.Empty(contract)
	return time.Since(start)
}

// Debug prints a debug message for the Empty operation.
func (op *Empty) Debug(ctx *dict.DictionaryContext) {
	fmt.Print(ctx.DecodeContract(op.ContractIndex))
}
