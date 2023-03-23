package operation

import (
	"encoding/binary"
	"fmt"
	"io"
	"time"

	"github.com/Fantom-foundation/Aida/state"
	"github.com/ethereum/go-ethereum/common"

	"github.com/Fantom-foundation/Aida/tracer/context"
)

// The GetStateLc operation is a GetState operation
// whose contract address refers to previously
// recorded/replayed operations.
// (NB: Lc = last contract addresss used)

// GetStateLc data structure
type GetStateLc struct {
	Key common.Hash
}

// GetId returns the get-state-lc operation identifier.
func (op *GetStateLc) GetId() byte {
	return GetStateLcID
}

// GetStateLc creates a new get-state-lc operation.
func NewGetStateLc(key common.Hash) *GetStateLc {
	return &GetStateLc{Key: key}
}

// ReadGetStateLc reads a get-state-lc operation from a file.
func ReadGetStateLc(file io.Reader) (Operation, error) {
	data := new(GetStateLc)
	err := binary.Read(file, binary.LittleEndian, data)
	return data, err
}

// Write the get-state-lc operation to file.
func (op *GetStateLc) Write(f io.Writer) error {
	err := binary.Write(f, binary.LittleEndian, *op)
	return err
}

// Execute the get-state-lc operation.
func (op *GetStateLc) Execute(db state.StateDB, ctx *context.Context) time.Duration {
	contract := ctx.PrevContract()
	storage := ctx.DecodeKey(op.Key)
	start := time.Now()
	db.GetState(contract, storage)
	return time.Since(start)
}

// Debug prints a debug message for the get-state-lc operation.
func (op *GetStateLc) Debug(ctx *context.Context) {
	contract := ctx.PrevContract()
	storage := ctx.DecodeKey(op.Key)
	fmt.Print(contract, storage)
}
