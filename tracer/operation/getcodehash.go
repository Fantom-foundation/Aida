package operation

import (
	"encoding/binary"
	"fmt"
	"github.com/Fantom-foundation/Aida/state"
	"io"
	"time"

	"github.com/Fantom-foundation/Aida/tracer/dict"
)

// GetCodeHash data structure
type GetCodeHash struct {
	ContractIndex uint32 // encoded contract address
}

// GetId returns the get-code-hash operation identifier.
func (op *GetCodeHash) GetId() byte {
	return GetCodeHashID
}

// NewGetCodeHash creates a new get-code-hash operation.
func NewGetCodeHash(cIdx uint32) *GetCodeHash {
	return &GetCodeHash{ContractIndex: cIdx}
}

// ReadGetHash reads a get-code-hash operation from a file.
func ReadGetCodeHash(file io.Reader) (Operation, error) {
	data := new(GetCodeHash)
	err := binary.Read(file, binary.LittleEndian, data)
	return data, err
}

// Write the get-code-hash operation to a file.
func (op *GetCodeHash) Write(f io.Writer) error {
	err := binary.Write(f, binary.LittleEndian, *op)
	return err
}

// Execute the get-code-hash operation.
func (op *GetCodeHash) Execute(db state.StateDB, ctx *dict.DictionaryContext) time.Duration {
	contract := ctx.DecodeContract(op.ContractIndex)
	start := time.Now()
	db.GetCodeHash(contract)
	return time.Since(start)
}

// Debug prints a debug message for the get-code-hash operation.
func (op *GetCodeHash) Debug(ctx *dict.DictionaryContext) {
	fmt.Print(ctx.DecodeContract(op.ContractIndex))
}
