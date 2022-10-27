package operation

import (
	"encoding/binary"
	"fmt"
	"os"
	"time"

	"github.com/Fantom-foundation/Aida/tracer/dict"
	"github.com/Fantom-foundation/Aida/tracer/state"
)

// CreateAccount data structure
type CreateAccount struct {
	ContractIndex uint32 // encoded contract address
}

// GetOpId returns the create-account operation identifier.
func (op *CreateAccount) GetOpId() byte {
	return CreateAccountID
}

// NewCreateAcccount creates a new create-account operation.
func NewCreateAccount(cIdx uint32) *CreateAccount {
	return &CreateAccount{ContractIndex: cIdx}
}

// ReadCreateAccount reads a create-account operation from a file.
func ReadCreateAccount(file *os.File) (Operation, error) {
	data := new(CreateAccount)
	err := binary.Read(file, binary.LittleEndian, data)
	return data, err
}

// Write the create-account operation to file.
func (op *CreateAccount) Write(f *os.File) error {
	err := binary.Write(f, binary.LittleEndian, *op)
	return err
}

// Execute the create-account operation.
func (op *CreateAccount) Execute(db state.StateDB, ctx *dict.DictionaryContext) time.Duration {
	contract := ctx.DecodeContract(op.ContractIndex)
	start := time.Now()
	db.CreateAccount(contract)
	return time.Since(start)
}

// Debug prints a debug message for the create-account operation.
func (op *CreateAccount) Debug(ctx *dict.DictionaryContext) {
	fmt.Printf("\tcontract: %v\n", ctx.DecodeContract(op.ContractIndex))
}
