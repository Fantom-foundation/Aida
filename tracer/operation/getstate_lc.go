package operation

import (
	"encoding/binary"
	"fmt"
	"io"
	"time"

	"github.com/Fantom-foundation/Aida/state"

	"github.com/Fantom-foundation/Aida/tracer/dictionary"
)

// The GetStateLc operation is a GetState operation
// whose contract address refers to previously
// recorded/replayed operations.
// (NB: Lc = last contract addresss used)

// GetStateLc data structure
type GetStateLc struct {
	StorageIndex uint32 // encoded storage address
}

// GetId returns the get-state-lc operation identifier.
func (op *GetStateLc) GetId() byte {
	return GetStateLcID
}

// GetStateLc creates a new get-state-lc operation.
func NewGetStateLc(sIdx uint32) *GetStateLc {
	return &GetStateLc{StorageIndex: sIdx}
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
func (op *GetStateLc) Execute(db state.StateDB, ctx *dict.DictionaryContext) time.Duration {
	contract := ctx.LastContractAddress()
	storage := ctx.DecodeStorage(op.StorageIndex)
	start := time.Now()
	db.GetState(contract, storage)
	return time.Since(start)
}

// Debug prints a debug message for the get-state-lc operation.
func (op *GetStateLc) Debug(ctx *dict.DictionaryContext) {
	contract := ctx.LastContractAddress()
	storage := ctx.DecodeStorage(op.StorageIndex)
	fmt.Print(contract, storage)
}
