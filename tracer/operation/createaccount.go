package operation

import (
	"encoding/binary"
	"fmt"
	"io"
	"time"

	"github.com/Fantom-foundation/Aida/state"
	"github.com/ethereum/go-ethereum/common"

	"github.com/Fantom-foundation/Aida/tracer/dictionary"
)

// CreateAccount data structure
type CreateAccount struct {
	Contract common.Address
}

// GetId returns the create-account operation identifier.
func (op *CreateAccount) GetId() byte {
	return CreateAccountID
}

// NewCreateAcccount creates a new create-account operation.
func NewCreateAccount(contract common.Address) *CreateAccount {
	return &CreateAccount{Contract: contract}
}

// ReadCreateAccount reads a create-account operation from a file.
func ReadCreateAccount(file io.Reader) (Operation, error) {
	data := new(CreateAccount)
	err := binary.Read(file, binary.LittleEndian, data)
	return data, err
}

// Write the create-account operation to file.
func (op *CreateAccount) Write(f io.Writer) error {
	err := binary.Write(f, binary.LittleEndian, *op)
	return err
}

// Execute the create-account operation.
func (op *CreateAccount) Execute(db state.StateDB, ctx *dictionary.Context) time.Duration {
	contract := ctx.DecodeContract(op.Contract)
	start := time.Now()
	db.CreateAccount(contract)
	return time.Since(start)
}

// Debug prints a debug message for the create-account operation.
func (op *CreateAccount) Debug(ctx *dictionary.Context) {
	fmt.Print(op.Contract)
}
