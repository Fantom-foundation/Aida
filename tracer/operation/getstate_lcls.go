package operation

import (
	"fmt"
	"io"
	"time"

	"github.com/Fantom-foundation/Aida/tracer/dict"
	"github.com/Fantom-foundation/Aida/tracer/state"
)

// The GetStateLcls operation is a GetState operation
// whose addresses refer to previous recorded/replayed
// operations.
// (NB: Lc = last contract addreess, ls = last storage
// address)

// GetStateLcls data structure
type GetStateLcls struct {
}

// GetOpId returns the get-state-lcls operation identifier.
func (op *GetStateLcls) GetOpId() byte {
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
func (op *GetStateLcls) Execute(db state.StateDB, ctx *dict.DictionaryContext) time.Duration {
	contract := ctx.LastContractAddress()
	storage := ctx.LookupStorage(0)
	start := time.Now()
	db.GetState(contract, storage)
	return time.Since(start)
}

// Debug prints a debug message for the get-state-lcls operation.
func (op *GetStateLcls) Debug(ctx *dict.DictionaryContext) {
	contract := ctx.LastContractAddress()
	storage := ctx.ReadStorage(0)
	fmt.Printf("\tcontract: %v\t storage: %v\n", contract, storage)
}
