package operation

import (
	"encoding/binary"
	"fmt"
	"io"
	"time"

	"github.com/Fantom-foundation/Aida/tracer/dict"
	"github.com/Fantom-foundation/Aida/tracer/state"
)

// Commit data structure
type Commit struct {
	DeleteEmptyObjects bool
}

// GetId returns the commit operation identifier.
func (op *Commit) GetId() byte {
	return CommitID
}

// NewCommit creates a new commit operation.
func NewCommit(deleteEmptyObjects bool) *Commit {
	return &Commit{DeleteEmptyObjects: deleteEmptyObjects}
}

// ReadCommit reads a commit operation from a file.
func ReadCommit(file io.Reader) (Operation, error) {
	data := new(Commit)
	err := binary.Read(file, binary.LittleEndian, data)
	return data, err
}

// Write the commit operation to a file.
func (op *Commit) Write(f io.Writer) error {
	err := binary.Write(f, binary.LittleEndian, *op)
	return err
}

// Execute the commit operation.
func (op *Commit) Execute(db state.StateDB, ctx *dict.DictionaryContext) time.Duration {
	start := time.Now()
	db.Commit(op.DeleteEmptyObjects)
	return time.Since(start)
}

// Debug prints a debug message for the commit operation.
func (op *Commit) Debug(ctx *dict.DictionaryContext) {
	fmt.Print(op.DeleteEmptyObjects)
}
