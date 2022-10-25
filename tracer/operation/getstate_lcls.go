package operation

import (
	"fmt"
	"os"
	"time"

	"github.com/Fantom-foundation/Aida/tracer/dict"
	"github.com/Fantom-foundation/Aida/tracer/state"
)

// Get-state data structure for last contract and last storage address.
type GetStateLcls struct {
}

// Return the get-state-lcls operation identifier.
func (op *GetStateLcls) GetOpId() byte {
	return GetStateLclsID
}

// Create a new get-state-lcls operation.
func NewGetStateLcls() *GetStateLcls {
	return new(GetStateLcls)
}

// Read a get-state-lcls operation from a file.
func ReadGetStateLcls(file *os.File) (Operation, error) {
	return NewGetStateLcls(), nil
}

// Write the get-state-lcls operation to file.
func (op *GetStateLcls) Write(f *os.File) error {
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

// Print a debug message for get-state-lcls operation.
func (op *GetStateLcls) Debug(ctx *dict.DictionaryContext) {
	contract := ctx.LastContractAddress()
	storage := ctx.LookupStorage(0)
	fmt.Printf("\tcontract: %v\t storage: %v\n", contract, storage)
}
