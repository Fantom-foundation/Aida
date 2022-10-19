package operation

import (
	"encoding/binary"
	"fmt"
	"os"

	"github.com/Fantom-foundation/Aida/tracer/dict"
	"github.com/Fantom-foundation/Aida/tracer/state"
)

// Get-state data structure with last contract address
type GetStateLc struct {
	StorageIndex uint32 // encoded storage address
}

// Return the get-state-lc operation identifier.
func (op *GetStateLc) GetOpId() byte {
	return GetStateLcID
}

// Create a new get-state-lc operation.
func NewGetStateLc(sIdx uint32) *GetStateLc {
	return &GetStateLc{StorageIndex: sIdx}
}

// Read a get-state-lc operation from a file.
func ReadGetStateLc(file *os.File) (Operation, error) {
	data := new(GetStateLc)
	err := binary.Read(file, binary.LittleEndian, data)
	return data, err
}

// Write the get-state-lc operation to file.
func (op *GetStateLc) Write(f *os.File) error {
	err := binary.Write(f, binary.LittleEndian, *op)
	return err
}

// Execute the get-state-lc operation.
func (op *GetStateLc) Execute(db state.StateDB, ctx *dict.DictionaryContext) {
	contract := ctx.LastContractAddress()
	storage := ctx.DecodeStorage(op.StorageIndex)
	db.GetState(contract, storage)
}

// Print a debug message.
func (op *GetStateLc) Debug(ctx *dict.DictionaryContext) {
	contract := ctx.LastContractAddress()
	storage := ctx.DecodeStorage(op.StorageIndex)
	fmt.Printf("\tcontract: %v\t storage: %v\n", contract, storage)
}
