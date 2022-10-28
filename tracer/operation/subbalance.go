package operation

import (
	"encoding/binary"
	"fmt"
	"log"
	"math/big"
	"os"
	"time"

	"github.com/Fantom-foundation/Aida/tracer/dict"
	"github.com/Fantom-foundation/Aida/tracer/state"
)

// SubBalance data structure
type SubBalance struct {
	ContractIndex uint32   // encoded contract address
	Amount        [16]byte // truncated amount to 16 bytes
}

// GetOpId returns the sub-balance operation identifier.
func (op *SubBalance) GetOpId() byte {
	return SubBalanceID
}

// NewSubBalance creates a new sub-balance operation.
func NewSubBalance(cIdx uint32, amount *big.Int) *SubBalance {
	// check if amount requires more than 256 bits (16 bytes)
	if amount.BitLen() > 256 {
		log.Fatalf("Amount exceeds 256 bit")
	}
	ret := &SubBalance{ContractIndex: cIdx}
	// copy amount to a 16-byte array with leading zeros
	amount.FillBytes(ret.Amount[:])
	return ret
}

// ReadSubBalance reads a sub-balance operation from a file.
func ReadSubBalance(file *os.File) (Operation, error) {
	data := new(SubBalance)
	err := binary.Read(file, binary.LittleEndian, data)
	return data, err
}

// Write the sub-balance operation to a file.
func (op *SubBalance) Write(f *os.File) error {
	err := binary.Write(f, binary.LittleEndian, *op)
	return err
}

// Execute the sub-balance operation.
func (op *SubBalance) Execute(db state.StateDB, ctx *dict.DictionaryContext) time.Duration {
	contract := ctx.DecodeContract(op.ContractIndex)
	// construct bit.Int from a byte array
	amount := new(big.Int).SetBytes(op.Amount[:])
	start := time.Now()
	db.SubBalance(contract, amount)
	return time.Since(start)
}

// Debug prints a debug message for the sub-balance operation.
func (op *SubBalance) Debug(ctx *dict.DictionaryContext) {
	fmt.Printf("\tcontract: %v\t amount: %v\n", ctx.DecodeContract(op.ContractIndex), new(big.Int).SetBytes(op.Amount[:]))
}
