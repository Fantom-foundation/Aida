package operation

import (
	"encoding/binary"
	"fmt"
	"log"
	"os"

	"github.com/Fantom-foundation/Aida/tracer/dict"
	"github.com/Fantom-foundation/Aida/tracer/state"
)

// Get-state data structure for using last contract address
// and cached storage address (lccs)
type GetStateLccs struct {
	StoragePosition uint8 // position in storage cache
}

// Return the get-state-lccs operation identifier.
func (op *GetStateLccs) GetOpId() byte {
	return GetStateLccsID
}

// Create a new get-state-lccs operation.
func NewGetStateLccs(sPos int) *GetStateLccs {
	if sPos < 0 || sPos > 255 {
		log.Fatalf("Position out of range")
	}
	return &GetStateLccs{StoragePosition: uint8(sPos)}
}

// Read a get-state-lccs operation from a file.
func ReadGetStateLccs(file *os.File) (Operation, error) {
	data := new(GetStateLccs)
	err := binary.Read(file, binary.LittleEndian, data)
	return data, err
}

// Write the get-state-lccs operation to file.
func (op *GetStateLccs) Write(f *os.File) error {
	err := binary.Write(f, binary.LittleEndian, *op)
	return err
}

// Execute the get-state-lccs operation.
func (op *GetStateLccs) Execute(db state.StateDB, ctx *dict.DictionaryContext) {
	contract := ctx.LastContractAddress()
	storage := ctx.LookupStorage(int(op.StoragePosition))
	db.GetState(contract, storage)
}

// Print a debug message.
func (op *GetStateLccs) Debug(ctx *dict.DictionaryContext) {
	contract := ctx.LastContractAddress()
	storage := ctx.ReadStorage(int(op.StoragePosition))
	fmt.Printf("\tcontract: %v\t storage: %v\n", contract, storage)
}
