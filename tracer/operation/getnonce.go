package operation

import (
	"encoding/binary"
	"fmt"
	"os"
	"time"

	"github.com/Fantom-foundation/Aida/tracer/dict"
	"github.com/Fantom-foundation/Aida/tracer/state"
)

// GetNonce data structure
type GetNonce struct {
	ContractIndex uint32 // encoded contract address
}

// GetOpId returns the get-nonce operation identifier.
func (op *GetNonce) GetOpId() byte {
	return GetNonceID
}

// NewGetNonce creates a new get-nonce operation.
func NewGetNonce(cIdx uint32) *GetNonce {
	return &GetNonce{ContractIndex: cIdx}
}

// ReadGetNonce reads a get-nonce operation from a file.
func ReadGetNonce(file *os.File) (Operation, error) {
	data := new(GetNonce)
	err := binary.Read(file, binary.LittleEndian, data)
	return data, err
}

// Write the get-nonce operation to a file.
func (op *GetNonce) Write(f *os.File) error {
	err := binary.Write(f, binary.LittleEndian, *op)
	return err
}

// Execute the get-nonce operation.
func (op *GetNonce) Execute(db state.StateDB, ctx *dict.DictionaryContext) time.Duration {
	contract := ctx.DecodeContract(op.ContractIndex)
	start := time.Now()
	db.GetNonce(contract)
	return time.Since(start)
}

// Debug prints a debug message for get-nonce.
func (op *GetNonce) Debug(ctx *dict.DictionaryContext) {
	fmt.Printf("\tcontract: %v\n", ctx.DecodeContract(op.ContractIndex))
}
