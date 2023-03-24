package operation

import (
	"fmt"
	"io"
	"time"

	"github.com/Fantom-foundation/Aida/state"

	"github.com/Fantom-foundation/Aida/tracer/context"
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
func ReadGetCodeHashLc(f io.Reader) (Operation, error) {
	return NewGetCodeHashLc(), nil
}

// Write the get-code-hash-lc operation to a file.
func (op *GetCodeHashLc) Write(f io.Writer) error {
	return nil
}

// Execute the get-code-hash-lc operation.
func (op *GetCodeHashLc) Execute(db state.StateDB, ctx *context.Context) time.Duration {
	contract := ctx.PrevContract()
	start := time.Now()
	db.GetCodeHash(contract)
	return time.Since(start)
}

// Debug prints a debug message for the get-code-hash-lc operation.
func (op *GetCodeHashLc) Debug(ctx *context.Context) {
	contract := ctx.PrevContract()
	fmt.Print(contract)
}
