package operation

import (
	"encoding/binary"
	"fmt"
	"io"
	"time"

	"github.com/Fantom-foundation/Aida/state"

	"github.com/Fantom-foundation/Aida/tracer/dictionary"
)

// GetBalance data structure
type GetBalance struct {
	ContractIndex uint32
}

// GetId returns the get-balance operation identifier.
func (op *GetBalance) GetId() byte {
	return GetBalanceID
}

// NewGetBalance creates a new get-balance operation.
func NewGetBalance(cIdx uint32) *GetBalance {
	return &GetBalance{ContractIndex: cIdx}
}

// ReadGetBalance reads a get-balance operation from a file.
func ReadGetBalance(file io.Reader) (Operation, error) {
	data := new(GetBalance)
	err := binary.Read(file, binary.LittleEndian, data)
	return data, err
}

// Write the get-balance operation.
func (op *GetBalance) Write(f io.Writer) error {
	err := binary.Write(f, binary.LittleEndian, *op)
	return err
}

// Execute the get-balance operation.
func (op *GetBalance) Execute(db state.StateDB, ctx *dictionary.DictionaryContext) time.Duration {
	contract := ctx.DecodeContract(op.ContractIndex)
	start := time.Now()
	db.GetBalance(contract)
	return time.Since(start)
}

// Debug prints a debug message for the get-balance operation.
func (op *GetBalance) Debug(ctx *dictionary.DictionaryContext) {
	fmt.Print(ctx.DecodeContract(op.ContractIndex))
}
