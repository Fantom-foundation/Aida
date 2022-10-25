package operation

import (
	"fmt"
	"os"
	"time"

	"github.com/Fantom-foundation/Aida/tracer/dict"
	"github.com/Fantom-foundation/Aida/tracer/state"
)

// Get-committed-state data structure for last contract and last storage address.
type GetCommittedStateLcls struct {
}

// Return the get-committed-state-lcls operation identifier.
func (op *GetCommittedStateLcls) GetOpId() byte {
	return GetCommittedStateLclsID
}

// Create a new get-committed-state-lcls operation.
func NewGetCommittedStateLcls() *GetCommittedStateLcls {
	return new(GetCommittedStateLcls)
}

// Read a get-committed-state-lcls operation from a file.
func ReadGetCommittedStateLcls(file *os.File) (Operation, error) {
	return NewGetCommittedStateLcls(), nil
}

// Write the get-committed-state-lcls operation to file.
func (op *GetCommittedStateLcls) Write(f *os.File) error {
	return nil
}

// Execute the get-committed-state-lcls operation.
func (op *GetCommittedStateLcls) Execute(db state.StateDB, ctx *dict.DictionaryContext) time.Duration {
	contract := ctx.LastContractAddress()
	storage := ctx.LookupStorage(0)
	start := time.Now()
	db.GetCommittedState(contract, storage)
	return time.Since(start)
}

// Print a debug message for get-committed-state-lcls operation.
func (op *GetCommittedStateLcls) Debug(ctx *dict.DictionaryContext) {
	contract := ctx.LastContractAddress()
	storage := ctx.ReadStorage(0)
	fmt.Printf("\tcontract: %v\t storage: %v\n", contract, storage)
}
