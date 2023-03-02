package operation

import (
	"encoding/binary"
	"fmt"
	"io"
	"time"

	"github.com/Fantom-foundation/Aida/state"

	"github.com/Fantom-foundation/Aida/tracer/dictionary"
)

// GetCode data structure
type GetCode struct {
	ContractIndex uint32 // encoded contract address
}

// GetId returns the get-code operation identifier.
func (op *GetCode) GetId() byte {
	return GetCodeID
}

// NewGetCode creates a new get-code operation.
func NewGetCode(cIdx uint32) *GetCode {
	return &GetCode{ContractIndex: cIdx}
}

// ReadGetCode reads a get-code operation from a file.
func ReadGetCode(file io.Reader) (Operation, error) {
	data := new(GetCode)
	err := binary.Read(file, binary.LittleEndian, data)
	return data, err
}

// Write the get-code operation to a file.
func (op *GetCode) Write(f io.Writer) error {
	err := binary.Write(f, binary.LittleEndian, *op)
	return err
}

// Execute the get-code operation.
func (op *GetCode) Execute(db state.StateDB, ctx *dictionary.Context) time.Duration {
	contract := ctx.DecodeContract(op.ContractIndex)
	start := time.Now()
	db.GetCode(contract)
	return time.Since(start)
}

// Debug prints a debug message for the get-code operation.
func (op *GetCode) Debug(ctx *dictionary.Context) {
	fmt.Print(ctx.DecodeContract(op.ContractIndex))
}
