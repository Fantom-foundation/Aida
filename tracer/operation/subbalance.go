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
	ContractIndex uint32 // encoded contract address
	Amount [16]byte
}

// GetOpId returns the sub-balance operation identifier.
func (op *SubBalance) GetOpId() byte {
	return SubBalanceID
}

// NewSubBalance creates a new sub-balance operation.
func NewSubBalance(cIdx uint32, amount *big.Int) *SubBalance {
	if amount.BitLen() > 256 {
		log.Fatalf("Amount exceeds 256 bit")
	}
	amountBytes := make([]byte, 16)
	amount.FillBytes(amountBytes)
	return &SubBalance{ContractIndex: cIdx, Amount: *(*[16]byte)(amountBytes)}
}

// ReadSubBalance reads a sub-balance operation from a file.
func ReadSubBalance(file *os.File) (Operation, error) {
	data := new(SubBalance)
	err := binary.Read(file, binary.LittleEndian, data)
	return data, err
}

// Write writes the sub-balance operation to a file.
func (op *SubBalance) Write(f *os.File) error {
	err := binary.Write(f, binary.LittleEndian, *op)
	return err
}

// Execute executes the sub-balance operation.
func (op *SubBalance) Execute(db state.StateDB, ctx *dict.DictionaryContext) time.Duration {
	// skip to avoid errors causing by negative balance when running on an empty db
	//contract := ctx.DecodeContract(op.ContractIndex)
	//amount := new(big.Int).SetBytes(op.Amount[:])
	//start := time.Now()
	//db.SubBalance(contract, amount)
	//return time.Since(start)
	return time.Duration(0)
}

// Debug prints a debug message for sub-balance.
func (op *SubBalance) Debug(ctx *dict.DictionaryContext) {
	fmt.Printf("\tcontract: %v\t sub amount: %v\n", ctx.DecodeContract(op.ContractIndex), new(big.Int).SetBytes(op.Amount[:]))
}
