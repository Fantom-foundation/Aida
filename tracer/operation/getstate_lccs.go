package operation

import (
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"time"

	"github.com/Fantom-foundation/Aida/state"

	"github.com/Fantom-foundation/Aida/tracer/dictionary"
)

// The GetStateLccs operation is a GetState operation whose
// addresses refer to previously recorded/replayed operations.
// (NB: Lc = last contract address, cs = cached storage
// address referring to a position in an indexed cache
// for storage addresses.)

// GetStateLccs  data structure
type GetStateLccs struct {
	StoragePosition uint8 // position in storage index-cache
}

// GetId returns the get-state-lccs operation identifier.
func (op *GetStateLccs) GetId() byte {
	return GetStateLccsID
}

// NewGetStateLccs creates a new get-state-lccs operation.
func NewGetStateLccs(sPos int) *GetStateLccs {
	if sPos < 0 || sPos > 255 {
		log.Fatalf("Position out of range")
	}
	return &GetStateLccs{StoragePosition: uint8(sPos)}
}

// ReadGetStateLccs reads a get-state-lccs operation from a file.
func ReadGetStateLccs(file io.Reader) (Operation, error) {
	data := new(GetStateLccs)
	err := binary.Read(file, binary.LittleEndian, data)
	return data, err
}

// Write the get-state-lccs operation to file.
func (op *GetStateLccs) Write(f io.Writer) error {
	err := binary.Write(f, binary.LittleEndian, *op)
	return err
}

// Execute the get-state-lccs operation.
func (op *GetStateLccs) Execute(db state.StateDB, ctx *dict.DictionaryContext) time.Duration {
	contract := ctx.LastContractAddress()
	storage := ctx.LookupStorage(int(op.StoragePosition))
	start := time.Now()
	db.GetState(contract, storage)
	return time.Since(start)
}

// Debug prints a debug message for the get-state-lccs operation.
func (op *GetStateLccs) Debug(ctx *dict.DictionaryContext) {
	contract := ctx.LastContractAddress()
	storage := ctx.ReadStorage(int(op.StoragePosition))
	fmt.Print(contract, storage)
}
