package operation

import (
	"encoding/binary"
	"fmt"
	"os"
	"time"

	"github.com/Fantom-foundation/Aida/tracer/dict"
	"github.com/Fantom-foundation/Aida/tracer/state"
)

// SetNonce data structure
type SetNonce struct {
	ContractIndex uint32 // encoded contract address
	Nonce         uint64 // nonce
}

// GetOpId returns the set-nonce operation identifier.
func (op *SetNonce) GetOpId() byte {
	return SetNonceID
}

// NewSetNonce creates a new set-nonce operation.
func NewSetNonce(cIdx uint32, nonce uint64) *SetNonce {
	return &SetNonce{ContractIndex: cIdx, Nonce: nonce}
}

// ReadSetNonce reads a set-nonce operation from a file.
func ReadSetNonce(file *os.File) (Operation, error) {
	data := new(SetNonce)
	err := binary.Read(file, binary.LittleEndian, data)
	return data, err
}

// Write the set-nonce operation to a file.
func (op *SetNonce) Write(f *os.File) error {
	err := binary.Write(f, binary.LittleEndian, *op)
	return err
}

// Execute the set-nonce operation.
func (op *SetNonce) Execute(db state.StateDB, ctx *dict.DictionaryContext) time.Duration {
	contract := ctx.DecodeContract(op.ContractIndex)
	start := time.Now()
	db.SetNonce(contract, op.Nonce)
	return time.Since(start)
}

// Debug prints a debug message for set-nonce.
func (op *SetNonce) Debug(ctx *dict.DictionaryContext) {
	fmt.Printf("\tcontract: %v nonce: %v\n", ctx.DecodeContract(op.ContractIndex), op.Nonce)
}
