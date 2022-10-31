package operation

import (
	"fmt"
	"os"
	"time"

	"github.com/Fantom-foundation/Aida/tracer/dict"
	"github.com/Fantom-foundation/Aida/tracer/state"
)

// GetCodeHashLc is a GetCodeHash operations whose
// contract address refers to previously recorded/
// replayed operations.
// (NB: Lc = last contract address)

// GetCodeHashLc data structure
type GetCodeHashLc struct {
}

// GetId returns the get-code-hash-lc operation identifier.
func (op *GetCodeHashLc) GetId() byte {
	return GetCodeHashLcID
}

// NewGetCodeHashLc creates a new get-code-hash-lc operation.
func NewGetCodeHashLc() *GetCodeHashLc {
	return &GetCodeHashLc{}
}

// ReadGetCodeHashLc reads a get-code-hash-lc operation from a file.
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

// Debug prints a debug message for the get-code-hash-lc operation.
func (op *GetCodeHashLc) Debug(ctx *dict.DictionaryContext) {
	contract := ctx.LastContractAddress()
	fmt.Printf("\tcontract: %v\n", contract)
}
