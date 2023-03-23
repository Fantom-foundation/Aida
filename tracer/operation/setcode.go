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

// SetCode data structure
type SetCode struct {
	Contract common.Address
	Bytecode []byte // encoded bytecode
}

// GetId returns the set-code operation identifier.
func (op *SetCode) GetId() byte {
	return SetCodeID
}

// NewSetCode creates a new set-code operation.
func NewSetCode(contract common.Address, bytecode []byte) *SetCode {
	return &SetCode{Contract: contract, Bytecode: bytecode}
}

// ReadSetCode reads a set-code operation from a file.
func ReadSetCode(file io.Reader) (Operation, error) {
	data := new(SetCode)
	if err := binary.Read(file, binary.LittleEndian, &data.Contract); err != nil {
		return nil, fmt.Errorf("Cannot read contract address. Error: %v", err)
	}
	var length uint32
	if err := binary.Read(file, binary.LittleEndian, &length); err != nil {
		return nil, fmt.Errorf("Cannot read byte-code length. Error: %v", err)
	}
	data.Bytecode = make([]byte, length)
	if err := binary.Read(file, binary.LittleEndian, data.Bytecode); err != nil {
		return nil, fmt.Errorf("Cannot read byte-code. Error: %v", err)
	}
	return data, nil
}

// Write the set-code operation to a file.
func (op *SetCode) Write(f io.Writer) error {
	if err := binary.Write(f, binary.LittleEndian, op.Contract); err != nil {
		return fmt.Errorf("Cannot write contract address. Error: %v", err)
	}
	var length uint32 = uint32(len(op.Bytecode))
	if err := binary.Write(f, binary.LittleEndian, &length); err != nil {
		return fmt.Errorf("Cannot read byte-code length. Error: %v", err)
	}
	if err := binary.Write(f, binary.LittleEndian, op.Bytecode); err != nil {
		return fmt.Errorf("Cannot write byte-code. Error: %v", err)
	}
	return nil
}

// Execute the set-code operation.
func (op *SetCode) Execute(db state.StateDB, ctx *context.Context) time.Duration {
	contract := ctx.DecodeContract(op.Contract)
	start := time.Now()
	db.SetCode(contract, op.Bytecode)
	return time.Since(start)
}

// Debug prints a debug message for the set-code operation.
func (op *SetCode) Debug(ctx *context.Context) {
	fmt.Print(op.Contract, op.Bytecode)
}
