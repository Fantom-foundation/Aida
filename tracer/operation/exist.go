package operation

import (
	"encoding/binary"
	"fmt"
	"io"
	"time"

	"github.com/Fantom-foundation/Aida/tracer/dict"
	"github.com/Fantom-foundation/Aida/tracer/state"
)

// Exist data structure
type Exist struct {
	ContractIndex uint32 // encoded contract address
}

// GetOpId returns the exist operation identifier.
func (op *Exist) GetOpId() byte {
	return ExistID
}

// NewExist creates a new exist operation.
func NewExist(cIdx uint32) *Exist {
	return &Exist{ContractIndex: cIdx}
}

// ReadExist reads an exist operation from a file.
func ReadExist(file io.Reader) (Operation, error) {
	data := new(Exist)
	err := binary.Read(file, binary.LittleEndian, data)
	return data, err
}

// Write the exist operation to a file.
func (op *Exist) Write(f io.Writer) error {
	err := binary.Write(f, binary.LittleEndian, *op)
	return err
}

// Execute the exist operation.
func (op *Exist) Execute(db state.StateDB, ctx *dict.DictionaryContext) time.Duration {
	contract := ctx.DecodeContract(op.ContractIndex)
	start := time.Now()
	db.Exist(contract)
	return time.Since(start)
}

// Debug prints a debug message for the exist operation.
func (op *Exist) Debug(ctx *dict.DictionaryContext) {
	fmt.Printf("\tcontract: %v\n", ctx.DecodeContract(op.ContractIndex))
}
