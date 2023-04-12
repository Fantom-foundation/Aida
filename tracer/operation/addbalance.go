package operation

import (
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"

	"github.com/Fantom-foundation/Aida/state"
	"github.com/Fantom-foundation/Aida/tracer/context"
)

// AddBalance data structure
type AddBalance struct {
	Contract common.Address
	Amount   [16]byte // truncated amount to 16 bytes
}

// GetId returns the add-balance operation identifier.
func (op *AddBalance) GetId() byte {
	return AddBalanceID
}

// NewAddBalance creates a new add-balance operation.
func NewAddBalance(contract common.Address, amount *big.Int) *AddBalance {
	// check if amount requires more than 256 bits (16 bytes)
	if amount.BitLen() > 256 {
		log.Fatalf("Amount exceeds 256 bit")
	}
	ret := &AddBalance{Contract: contract}
	// copy amount to a 16-byte array with leading zeros
	amount.FillBytes(ret.Amount[:])
	return ret
}

// ReadAddBalance reads a add-balance operation from a file.
func ReadAddBalance(f io.Reader) (Operation, error) {
	data := new(AddBalance)
	err := binary.Read(f, binary.LittleEndian, data)
	return data, err
}

// Write writes the add-balance operation to a file.
func (op *AddBalance) Write(f io.Writer) error {
	err := binary.Write(f, binary.LittleEndian, *op)
	return err
}

// Execute executes the add-balance operation.
func (op *AddBalance) Execute(db state.StateDB, ctx *context.Replay) time.Duration {
	contract := ctx.DecodeContract(op.Contract)
	// construct bit.Int from a byte array
	amount := new(big.Int).SetBytes(op.Amount[:])
	start := time.Now()
	db.AddBalance(contract, amount)
	return time.Since(start)
}

// Debug prints a debug message for the add-balance operation.
func (op *AddBalance) Debug(ctx *context.Context) {
	fmt.Print(op.Contract, new(big.Int).SetBytes(op.Amount[:]))
}
