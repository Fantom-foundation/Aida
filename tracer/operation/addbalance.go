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

// AddBalance data structure
type AddBalance struct {
	ContractIndex uint32   // encoded contract address
	Amount        [16]byte // truncated amount to 16 bytes
}

// GetOpId returns the add-balance operation identifier.
func (op *AddBalance) GetOpId() byte {
	return AddBalanceID
}

// NewAddBalance creates a new add-balance operation.
func NewAddBalance(cIdx uint32, amount *big.Int) *AddBalance {
	// check if amount requires more than 256 bits (16 bytes)
	if amount.BitLen() > 256 {
		log.Fatalf("Amount exceeds 256 bit")
	}
	ret := &AddBalance{ContractIndex: cIdx}
	// copy amount to a 16-byte array with leading zeros
	amount.FillBytes(ret.Amount[:])
	return ret
}

// ReadAddBalance reads a add-balance operation from a file.
func ReadAddBalance(file *os.File) (Operation, error) {
	data := new(AddBalance)
	err := binary.Read(file, binary.LittleEndian, data)
	return data, err
}

// Write writes the add-balance operation to a file.
func (op *AddBalance) Write(f *os.File) error {
	err := binary.Write(f, binary.LittleEndian, *op)
	return err
}

// Execute executes the add-balance operation.
func (op *AddBalance) Execute(db state.StateDB, ctx *dict.DictionaryContext) time.Duration {
	contract := ctx.DecodeContract(op.ContractIndex)
	// construct bit.Int from a byte array
	amount := new(big.Int).SetBytes(op.Amount[:])
	start := time.Now()
	db.AddBalance(contract, amount)
	return time.Since(start)
}

// Debug prints a debug message for the add-balance operation.
func (op *AddBalance) Debug(ctx *dict.DictionaryContext) {
	fmt.Printf("\tcontract: %v\t amount: %v\n", ctx.DecodeContract(op.ContractIndex), new(big.Int).SetBytes(op.Amount[:]))
}
