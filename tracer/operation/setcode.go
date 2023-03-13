package operation

import (
	"encoding/binary"
	"fmt"
	"io"
	"time"

	"github.com/Fantom-foundation/Aida/state"
	"github.com/ethereum/go-ethereum/common"

	"github.com/Fantom-foundation/Aida/tracer/dictionary"
)

// SetCode data structure
type SetCode struct {
	Contract  common.Address
	CodeIndex uint32 // encoded bytecode
}

// GetId returns the set-code operation identifier.
func (op *SetCode) GetId() byte {
	return SetCodeID
}

// NewSetCode creates a new set-code operation.
func NewSetCode(contract common.Address, bcontract uint32) *SetCode {
	return &SetCode{Contract: contract, CodeIndex: bcontract}
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
func (op *SetCode) Execute(db state.StateDB, ctx *dictionary.Context) time.Duration {
	contract := ctx.DecodeContract(op.Contract)
	code := ctx.DecodeCode(op.CodeIndex)
	start := time.Now()
	db.SetCode(contract, code)
	return time.Since(start)
}

// Debug prints a debug message for the set-code operation.
func (op *SetCode) Debug(ctx *dictionary.Context) {
	fmt.Print(op.Contract, ctx.DecodeCode(op.CodeIndex))
}
