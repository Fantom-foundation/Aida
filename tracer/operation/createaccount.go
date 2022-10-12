package operation

import (
	"encoding/binary"
	"fmt"
	"os"

	"github.com/Fantom-foundation/aida/tracer/dict"
	"github.com/Fantom-foundation/substate-cli/state"
)

// Create-account data structure
type CreateAccount struct {
	ContractIndex uint32 // encoded contract address
}

// Return the create-account operation identifier.
func (op *CreateAccount) GetOpId() byte {
	return CreateAccountID
}

// Create a new create account operation.
func NewCreateAccount(cIdx uint32) *CreateAccount {
	return &CreateAccount{ContractIndex: cIdx}
}

// Read a create-account operation from a file.
func ReadCreateAccount(file *os.File) (Operation, error) {
	data := new(CreateAccount)
	err := binary.Read(file, binary.LittleEndian, data)
	return data, err
}

// Write the create account operation to file.
func (op *CreateAccount) Write(f *os.File) error {
	err := binary.Write(f, binary.LittleEndian, *op)
	return err
}

// Execute the create account operation.
func (op *CreateAccount) Execute(db state.StateDB, ctx *dict.DictionaryContext) {
	contract := ctx.DecodeContract(op.ContractIndex)
	db.CreateAccount(contract)
}

// Print a debug message for snapshot operation.
func (op *CreateAccount) Debug(ctx *dict.DictionaryContext) {
	fmt.Printf("\tcontract: %v\n", ctx.DecodeContract(op.ContractIndex))
}
