package operation

import (
	"fmt"
	"io"
	"time"

	"github.com/Fantom-foundation/Aida/state"
	"github.com/Fantom-foundation/Aida/tracer/dictionary"
)

// The GetStateLcls operation is a GetState operation
// whose addresses refer to previous recorded/replayed
// operations.
// (NB: Lc = last contract addreess, ls = last storage
// address)

// GetStateLcls data structure
type GetStateLcls struct {
}

// GetId returns the get-state-lcls operation identifier.
func (op *GetStateLcls) GetId() byte {
	return GetStateLclsID
}

// NewGetStateLcls creates a new get-state-lcls operation.
func NewGetStateLcls() *GetStateLcls {
	return new(GetStateLcls)
}

// ReadGetStateLcls reads a get-state-lcls operation from a file.
func ReadGetStateLcls(file io.Reader) (Operation, error) {
	return NewGetStateLcls(), nil
}

// Write the get-state-lcls operation to file.
func (op *GetStateLcls) Write(f io.Writer) error {
	return nil
}

// Execute the get-state-lcls operation.
func (op *GetStateLcls) Execute(db state.StateDB, ctx *dictionary.Context) time.Duration {
	contract := ctx.PrevContract()
	storage := ctx.DecodeStorageCache(0)
	start := time.Now()
	db.GetState(contract, storage)
	return time.Since(start)
}

// Debug prints a debug message for the get-state-lcls operation.
func (op *GetStateLcls) Debug(ctx *dictionary.Context) {
	contract := ctx.PrevContract()
	storage := ctx.ReadStorageCache(0)
	fmt.Print(contract, storage)
}
