package operation

import (
	"fmt"
	"os"

	"github.com/Fantom-foundation/Aida/tracer/dict"
	"github.com/Fantom-foundation/Aida/tracer/state"
)

// Get-state data structure
type LastGetState struct {
}

// Return the last-get-state operation identifier.
func (op *LastGetState) GetOpId() byte {
	return LastGetStateID
}

// Create a new last-get-state operation.
func NewLastGetState() *LastGetState {
	return new(LastGetState)
}

// Read a last-get-state operation from a file.
func ReadLastGetState(file *os.File) (Operation, error) {
	return NewLastGetState(), nil
}

// Write the last-get-state operation to file.
func (op *LastGetState) Write(f *os.File) error {
	return nil
}

// Execute the last-get-state operation.
func (op *LastGetState) Execute(db state.StateDB, ctx *dict.DictionaryContext) {
	contract := ctx.LastContractAddress()
	storage := ctx.LookupStorage(0)
	db.GetState(contract, storage)
}

// Print a debug message for last-get-state operation.
func (op *LastGetState) Debug(ctx *dict.DictionaryContext) {
	contract := ctx.LastContractAddress()
	storage := ctx.LookupStorage(0)
	fmt.Printf("\tcontract: %v\t storage: %v\n", contract, storage)
}
