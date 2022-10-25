package operation

import (
	"encoding/binary"
	"fmt"
	"os"
	"time"

	"github.com/Fantom-foundation/Aida/tracer/dict"
	"github.com/Fantom-foundation/Aida/tracer/state"
)

// SetStateLcls data structure for last contract address and last storage
// address.
type SetStateLcls struct {
	ValueIndex uint64 // encoded storage value
}

// Return the set-state-lcls identifier
func (op *SetStateLcls) GetOpId() byte {
	return SetStateLclsID
}

// Create a new set-state-lcls operation.
func NewSetStateLcls(vIdx uint64) *SetStateLcls {
	return &SetStateLcls{ValueIndex: vIdx}
}

// Read a set-state-lcls operation from file.
func ReadSetStateLcls(file *os.File) (Operation, error) {
	data := new(SetStateLcls)
	err := binary.Read(file, binary.LittleEndian, data)
	return data, err
}

// Write the set-state-lcls operation to file.
func (op *SetStateLcls) Write(f *os.File) error {
	err := binary.Write(f, binary.LittleEndian, *op)
	return err
}

// Execute the set-state-lcls operation.
func (op *SetStateLcls) Execute(db state.StateDB, ctx *dict.DictionaryContext) time.Duration {
	contract := ctx.LastContractAddress()
	storage := ctx.LookupStorage(0)
	value := ctx.DecodeValue(op.ValueIndex)
	start := time.Now()
	db.SetState(contract, storage, value)
	return time.Since(start)
}

// Print a debug message for set-state-lcls.
func (op *SetStateLcls) Debug(ctx *dict.DictionaryContext) {
	contract := ctx.LastContractAddress()
	storage := ctx.ReadStorage(0)
	value := ctx.DecodeValue(op.ValueIndex)
	fmt.Printf("\tcontract: %v\t storage: %v\t value: %v\n",
		contract,
		storage,
		value)
}
