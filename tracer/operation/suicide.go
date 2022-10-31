package operation

import (
	"encoding/binary"
	"fmt"
	"io"
	"time"

	"github.com/Fantom-foundation/Aida/tracer/dict"
	"github.com/Fantom-foundation/Aida/tracer/state"
)

// Suicide data structure
type Suicide struct {
	ContractIndex uint32 // encoded contract address
}

// GetId returns the suicide operation identifier.
func (op *Suicide) GetId() byte {
	return SuicideID
}

// NewSuicide creates a new suicide operation.
func NewSuicide(cIdx uint32) *Suicide {
	return &Suicide{ContractIndex: cIdx}
}

// ReadSuicide reads a suicide operation from a file.
func ReadSuicide(file io.Reader) (Operation, error) {
	data := new(Suicide)
	err := binary.Read(file, binary.LittleEndian, data)
	return data, err
}

// Write the suicide operation to a file.
func (op *Suicide) Write(f io.Writer) error {
	err := binary.Write(f, binary.LittleEndian, *op)
	return err
}

// Execute the suicide operation.
func (op *Suicide) Execute(db state.StateDB, ctx *dict.DictionaryContext) time.Duration {
	contract := ctx.DecodeContract(op.ContractIndex)
	start := time.Now()
	db.Suicide(contract)
	return time.Since(start)
}

// Debug prints a debug message for the suicide operation.
func (op *Suicide) Debug(ctx *dict.DictionaryContext) {
	fmt.Printf("\tcontract: %v\n", ctx.DecodeContract(op.ContractIndex))
}
