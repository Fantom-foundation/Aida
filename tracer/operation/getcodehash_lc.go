package operation

import (
	"fmt"
	"os"
	"time"

	"github.com/Fantom-foundation/Aida/tracer/dict"
	"github.com/Fantom-foundation/Aida/tracer/state"
)

// Get-code-hash data structure with last contract address
type GetCodeHashLc struct {
}

// Return the get-code-hash-lc operation identifier.
func (op *GetCodeHashLc) GetOpId() byte {
	return GetCodeHashLcID
}

// Create a new get-code-hash-lc operation.
func NewGetCodeHashLc() *GetCodeHashLc {
	return &GetCodeHashLc{}
}

// Read a get-code-hash-lc operation from a file.
func ReadGetCodeHashLc(file *os.File) (Operation, error) {
	return NewGetCodeHashLc(), nil
}

// Write the get-code-hash-lc operation to a file.
func (op *GetCodeHashLc) Write(f *os.File) error {
	return nil
}

// Execute the get-code-hash-lc operation.
func (op *GetCodeHashLc) Execute(db state.StateDB, ctx *dict.DictionaryContext) time.Duration {
	contract := ctx.LastContractAddress()
	start := time.Now()
	db.GetCodeHash(contract)
	return time.Since(start)
}

// Print a debug message for get-code-hash-lc.
func (op *GetCodeHashLc) Debug(ctx *dict.DictionaryContext) {
	contract := ctx.LastContractAddress()
	fmt.Printf("\tcontract: %v\n", contract)
}
