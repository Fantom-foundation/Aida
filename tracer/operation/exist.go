package operation

import (
	"encoding/binary"
	"fmt"
	"io"
	"time"

	"github.com/Fantom-foundation/Aida/state"
	"github.com/ethereum/go-ethereum/common"

	"github.com/Fantom-foundation/Aida/tracer/context"
)

// Exist data structure
type Exist struct {
	Contract common.Address
}

// GetId returns the exist operation identifier.
func (op *Exist) GetId() byte {
	return ExistID
}

// NewExist creates a new exist operation.
func NewExist(contract common.Address) *Exist {
	return &Exist{Contract: contract}
}

// ReadExist reads an exist operation from a file.
func ReadExist(f io.Reader) (Operation, error) {
	data := new(Exist)
	err := binary.Read(f, binary.LittleEndian, data)
	return data, err
}

// Write the exist operation to a file.
func (op *Exist) Write(f io.Writer) error {
	err := binary.Write(f, binary.LittleEndian, *op)
	return err
}

// Execute the exist operation.
func (op *Exist) Execute(db state.StateDB, ctx *context.Replay) time.Duration {
	contract := ctx.DecodeContract(op.Contract)
	start := time.Now()
	db.Exist(contract)
	return time.Since(start)
}

// Debug prints a debug message for the exist operation.
func (op *Exist) Debug(ctx *context.Context) {
	fmt.Print(ctx.DecodeContract(op.Contract))
}
