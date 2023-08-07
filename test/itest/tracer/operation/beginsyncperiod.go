package operation

import (
	"encoding/binary"
	"fmt"
	"io"
	"time"

	"github.com/Fantom-foundation/rc-testing/test/itest/state"

	"github.com/Fantom-foundation/rc-testing/test/itest/tracer/context"
)

// BeginSyncPeriod data structure
type BeginSyncPeriod struct {
	SyncPeriodNumber uint64
}

// GetId returns the begin-sync-period operation identifier.
func (op *BeginSyncPeriod) GetId() byte {
	return BeginSyncPeriodID
}

// NewBeginSyncPeriod creates a new begin-sync-period operation.
func NewBeginSyncPeriod(number uint64) *BeginSyncPeriod {
	return &BeginSyncPeriod{number}
}

// ReadBeginSyncPeriod reads a begin-sync-period operation from file.
func ReadBeginSyncPeriod(f io.Reader) (Operation, error) {
	data := new(BeginSyncPeriod)
	err := binary.Read(f, binary.LittleEndian, data)
	return data, err
}

// Write the begin-sync-period operation to file.
func (op *BeginSyncPeriod) Write(f io.Writer) error {
	return binary.Write(f, binary.LittleEndian, *op)
}

// Execute the begin-sync-period operation.
func (op *BeginSyncPeriod) Execute(db state.StateDB, ctx *context.Replay) time.Duration {
	start := time.Now()
	db.BeginSyncPeriod(op.SyncPeriodNumber)
	return time.Since(start)
}

// Debug prints a debug message for the begin-sync-period operation.
func (op *BeginSyncPeriod) Debug(ctx *context.Context) {
	fmt.Print(op.SyncPeriodNumber)
}
