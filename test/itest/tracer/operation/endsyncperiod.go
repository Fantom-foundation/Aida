package operation

import (
	"encoding/binary"
	"io"
	"time"

	"github.com/Fantom-foundation/rc-testing/test/itest/state"

	"github.com/Fantom-foundation/rc-testing/test/itest/tracer/context"
)

// End-sync-period operation data structure
type EndSyncPeriod struct {
}

// GetId returns the end-sync-period operation identifier.
func (op *EndSyncPeriod) GetId() byte {
	return EndSyncPeriodID
}

// NewEndSyncPeriod creates a new end-sync-period operation.
func NewEndSyncPeriod() *EndSyncPeriod {
	return &EndSyncPeriod{}
}

// ReadEndSyncPeriod reads an end-sync-period operation from file.
func ReadEndSyncPeriod(f io.Reader) (Operation, error) {
	return new(EndSyncPeriod), nil
}

// Write the end-sync-period operation to file.
func (op *EndSyncPeriod) Write(f io.Writer) error {
	err := binary.Write(f, binary.LittleEndian, *op)
	return err
}

// Execute the end-sync-period operation.
func (op *EndSyncPeriod) Execute(db state.StateDB, ctx *context.Replay) time.Duration {
	start := time.Now()
	db.EndSyncPeriod()
	return time.Since(start)
}

// Debug prints a debug message for the end-sync-period operation.
func (op *EndSyncPeriod) Debug(ctx *context.Context) {
}
