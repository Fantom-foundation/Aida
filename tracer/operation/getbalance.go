package operation

import (
	"encoding/binary"
	"fmt"
	"os"
	"time"

	"github.com/Fantom-foundation/Aida/tracer/dict"
	"github.com/Fantom-foundation/Aida/tracer/state"
)

// GetBalance data structure
type GetBalance struct {
	ContractIndex uint32
}

// Return the get-balance operation identifier.
func (op *GetBalance) GetOpId() byte {
	return GetBalanceID
}

// Create a new get-balance operation.
func NewGetBalance(cIdx uint32) *GetBalance {
	return &GetBalance{ContractIndex: cIdx}
}

// Read a get-balance operation from a file.
func ReadGetBalance(file *os.File) (Operation, error) {
	data := new(GetBalance)
	err := binary.Read(file, binary.LittleEndian, data)
	return data, err
}

// Write the get-balance operation.
func (op *GetBalance) Write(f *os.File) error {
	err := binary.Write(f, binary.LittleEndian, *op)
	return err
}

// Execute the get-balance operation.
func (op *GetBalance) Execute(db state.StateDB, ctx *dict.DictionaryContext) time.Duration {
	contract := ctx.DecodeContract(op.ContractIndex)
	start := time.Now()
	db.GetBalance(contract)
	return time.Since(start)
}

// Print a debug message for get-balance.
func (op *GetBalance) Debug(ctx *dict.DictionaryContext) {
	fmt.Printf("\tcontract: %v\n", ctx.DecodeContract(op.ContractIndex))
}
