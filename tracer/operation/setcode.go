package operation

import (
	"encoding/binary"
	"fmt"
	"io"
	"time"

	"github.com/Fantom-foundation/Aida/tracer/dict"
	"github.com/Fantom-foundation/Aida/tracer/state"
)

// SetCode data structure
type SetCode struct {
	ContractIndex uint32 // encoded contract address
	CodeIndex     uint32 // encoded bytecode
}

// GetId returns the set-code operation identifier.
func (op *SetCode) GetId() byte {
	return SetCodeID
}

// NewSetCode creates a new set-code operation.
func NewSetCode(cIdx uint32, bcIdx uint32) *SetCode {
	return &SetCode{ContractIndex: cIdx, CodeIndex: bcIdx}
}

// ReadSetCode reads a set-code operation from a file.
func ReadSetCode(file io.Reader) (Operation, error) {
	data := new(SetCode)
	err := binary.Read(file, binary.LittleEndian, data)
	return data, err
}

// Write the set-code operation to a file.
func (op *SetCode) Write(f io.Writer) error {
	err := binary.Write(f, binary.LittleEndian, *op)
	return err
}

// Execute the set-code operation.
func (op *SetCode) Execute(db state.StateDB, ctx *dict.DictionaryContext) time.Duration {
	contract := ctx.DecodeContract(op.ContractIndex)
	code := ctx.DecodeCode(op.CodeIndex)
	start := time.Now()
	db.SetCode(contract, code)
	return time.Since(start)
}

// Debug prints a debug message for the set-code operation.
func (op *SetCode) Debug(ctx *dict.DictionaryContext) {
	fmt.Sprintf("\tcontract: %v code: %x\n",
		ctx.DecodeContract(op.ContractIndex),
		ctx.DecodeCode(op.CodeIndex))
}
