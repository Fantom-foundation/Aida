package operation

import (
	"encoding/binary"
	"fmt"
	"io"
	"time"

	"github.com/Fantom-foundation/Aida/state"

	"github.com/Fantom-foundation/Aida/tracer/dictionary"
)

// GetNonce data structure
type GetNonce struct {
	ContractIndex uint32 // encoded contract address
}

// GetId returns the get-nonce operation identifier.
func (op *GetNonce) GetId() byte {
	return GetNonceID
}

// NewGetNonce creates a new get-nonce operation.
func NewGetNonce(cIdx uint32) *GetNonce {
	return &GetNonce{ContractIndex: cIdx}
}

// ReadGetNonce reads a get-nonce operation from a file.
func ReadGetNonce(file io.Reader) (Operation, error) {
	data := new(GetNonce)
	err := binary.Read(file, binary.LittleEndian, data)
	return data, err
}

// Write the get-nonce operation to a file.
func (op *GetNonce) Write(f io.Writer) error {
	err := binary.Write(f, binary.LittleEndian, *op)
	return err
}

// Execute the get-nonce operation.
func (op *GetNonce) Execute(db state.StateDB, ctx *dictionary.Context) time.Duration {
	contract := ctx.DecodeContract(op.ContractIndex)
	start := time.Now()
	db.GetNonce(contract)
	return time.Since(start)
}

// Debug prints a debug message for the get-nonce operation.
func (op *GetNonce) Debug(ctx *dictionary.Context) {
	fmt.Print(ctx.DecodeContract(op.ContractIndex))
}
