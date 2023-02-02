package operation

import (
	"fmt"
	"github.com/Fantom-foundation/Aida/state"
	"io"
	"time"

	"github.com/Fantom-foundation/Aida/tracer/dict"
)

// GetCommittedStateLcls is a GetCommittedState operation
// whose contract/storage addresses refer to previously
// recorded/replayed operations.
// (NB: Lc = last contract address, ls = last storage
// address)

// GetCommittedStateLcls data structure
type GetCommittedStateLcls struct {
}

// GetId returns the get-committed-state-lcls operation identifier.
func (op *GetCommittedStateLcls) GetId() byte {
	return GetCommittedStateLclsID
}

// NewGetCommittedStateLcls creates a new get-committed-state-lcls operation.
func NewGetCommittedStateLcls() *GetCommittedStateLcls {
	return new(GetCommittedStateLcls)
}

// ReadGetCommittedStateLcls reads a get-committed-state-lcls operation from a file.
func ReadGetCommittedStateLcls(file io.Reader) (Operation, error) {
	return NewGetCommittedStateLcls(), nil
}

// Write the get-committed-state-lcls operation to file.
func (op *GetCommittedStateLcls) Write(f io.Writer) error {
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

// Debug prints a debug message for the get-committed-state-lcls operation.
func (op *GetCommittedStateLcls) Debug(ctx *dict.DictionaryContext) {
	contract := ctx.LastContractAddress()
	storage := ctx.ReadStorage(0)
	fmt.Print(contract, storage)
}
