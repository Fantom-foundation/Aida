package operation

import (
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"math/big"
	"time"

	"github.com/Fantom-foundation/Aida/state"
	"github.com/ethereum/go-ethereum/common"

	"github.com/Fantom-foundation/Aida/tracer/context"
)

// SubBalance data structure
type SubBalance struct {
	Contract common.Address
	Amount   [16]byte // truncated amount to 16 bytes
}

// GetId returns the sub-balance operation identifier.
func (op *SubBalance) GetId() byte {
	return SubBalanceID
}

// NewSubBalance creates a new sub-balance operation.
func NewSubBalance(contract common.Address, amount *big.Int) *SubBalance {
	// check if amount requires more than 256 bits (16 bytes)
	if amount.BitLen() > 256 {
		log.Fatalf("Amount exceeds 256 bit")
	}
	ret := &SubBalance{Contract: contract}
	// copy amount to a 16-byte array with leading zeros
	amount.FillBytes(ret.Amount[:])
	return ret
}

// ReadSubBalance reads a sub-balance operation from a file.
func ReadSubBalance(file io.Reader) (Operation, error) {
	data := new(SubBalance)
	err := binary.Read(file, binary.LittleEndian, data)
	return data, err
}

// Write the sub-balance operation to a file.
func (op *SubBalance) Write(f io.Writer) error {
	err := binary.Write(f, binary.LittleEndian, *op)
	return err
}

// Execute the sub-balance operation.
func (op *SubBalance) Execute(db state.StateDB, ctx *context.Context) time.Duration {
	contract := ctx.DecodeContract(op.Contract)
	// construct bit.Int from a byte array
	amount := new(big.Int).SetBytes(op.Amount[:])
	start := time.Now()
	db.SubBalance(contract, amount)
	return time.Since(start)
}

// Debug prints a debug message for the sub-balance operation.
func (op *SubBalance) Debug(ctx *context.Context) {
	fmt.Print(op.Contract, new(big.Int).SetBytes(op.Amount[:]))
}
