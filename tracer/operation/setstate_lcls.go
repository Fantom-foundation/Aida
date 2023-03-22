package operation

import (
	"encoding/binary"
	"fmt"
	"io"
	"time"

	"github.com/Fantom-foundation/Aida/state"
	"github.com/ethereum/go-ethereum/common"

	"github.com/Fantom-foundation/Aida/tracer/dictionary"
)

// The SetStateLcls operation is a SetState operation whose
// addresses refer to previously recorded/replayed addresses.
// (NB: Lc = last contract address, Ls = last storage address)

// SetStateLcls data structure
type SetStateLcls struct {
	Value common.Hash // encoded storage value
}

// GetId returns the set-state-lcls identifier.
func (op *SetStateLcls) GetId() byte {
	return SetStateLclsID
}

// SetStateLcls creates a new set-state-lcls operation.
func NewSetStateLcls(value common.Hash) *SetStateLcls {
	return &SetStateLcls{Value: value}
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
func (op *SetStateLcls) Execute(db state.StateDB, ctx *dictionary.Context) time.Duration {
	contract := ctx.PrevContract()
	storage := ctx.DecodeStorageCache(0)
	start := time.Now()
	db.SetState(contract, storage, op.Value)
	return time.Since(start)
}

// Debug prints a debug message for the set-state-lcls operation.
func (op *SetStateLcls) Debug(ctx *dictionary.Context) {
	contract := ctx.PrevContract()
	storage := ctx.ReadStorageCache(0)
	fmt.Print(contract, storage, op.Value)
}
