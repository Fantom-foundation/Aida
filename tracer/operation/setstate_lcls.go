package operation

import (
	"encoding/binary"
	"fmt"
	"io"
	"time"

	"github.com/Fantom-foundation/Aida/tracer/dict"
	"github.com/Fantom-foundation/Aida/tracer/state"
)

// The SetStateLcls operation is a SetState operation whose
// addresses refer to previously recorded/replayed addresses.
// (NB: Lc = last contract address, Ls = last storage address)

// SetStateLcls data structure
type SetStateLcls struct {
	ValueIndex uint64 // encoded storage value
}

// GetId returns the set-state-lcls identifier.
func (op *SetStateLcls) GetId() byte {
	return SetStateLclsID
}

// SetStateLcls creates a new set-state-lcls operation.
func NewSetStateLcls(vIdx uint64) *SetStateLcls {
	return &SetStateLcls{ValueIndex: vIdx}
}

// ReadSetStateLcls reads a set-state-lcls operation from file.
func ReadSetStateLcls(file io.Reader) (Operation, error) {
	data := new(SetStateLcls)
	err := binary.Read(file, binary.LittleEndian, data)
	return data, err
}

// Write the set-state-lcls operation to file.
func (op *SetStateLcls) Write(f io.Writer) error {
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

// Debug prints a debug message for the set-state-lcls operation.
func (op *SetStateLcls) Debug(ctx *dict.DictionaryContext) {
	contract := ctx.LastContractAddress()
	storage := ctx.ReadStorage(0)
	value := ctx.DecodeValue(op.ValueIndex)
	fmt.Printf("\tcontract: %v\t storage: %v\t value: %v\n",
		contract,
		storage,
		value)
}
