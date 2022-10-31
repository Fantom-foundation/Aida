package operation

import (
	"encoding/binary"
	"fmt"
	"io"
	"time"

	"github.com/Fantom-foundation/Aida/tracer/dict"
	"github.com/Fantom-foundation/Aida/tracer/state"
)

// GetCodeSize data structure
type GetCodeSize struct {
	ContractIndex uint32 // encoded contract address
}

// GetCodeSize returns the get-code-size operation identifier.
func (op *GetCodeSize) GetOpId() byte {
	return GetCodeSizeID
}

// NewGetCodeSize creates a new get-code-size operation.
func NewGetCodeSize(cIdx uint32) *GetCodeSize {
	return &GetCodeSize{ContractIndex: cIdx}
}

// ReadGetCodeSize reads a get-code-size operation from a file.
func ReadGetCodeSize(file io.Reader) (Operation, error) {
	data := new(GetCodeSize)
	err := binary.Read(file, binary.LittleEndian, data)
	return data, err
}

// Write the get-code-size operation to a file.
func (op *GetCodeSize) Write(f io.Writer) error {
	err := binary.Write(f, binary.LittleEndian, *op)
	return err
}

// Execute the get-code-size operation.
func (op *GetCodeSize) Execute(db state.StateDB, ctx *dict.DictionaryContext) time.Duration {
	contract := ctx.DecodeContract(op.ContractIndex)
	start := time.Now()
	db.GetCodeSize(contract)
	return time.Since(start)
}

// Debug prints a debug message for get-code-size.
func (op *GetCodeSize) Debug(ctx *dict.DictionaryContext) {
	fmt.Printf("\tcontract: %v\n", ctx.DecodeContract(op.ContractIndex))
}
