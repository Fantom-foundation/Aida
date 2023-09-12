package extension

import "github.com/Fantom-foundation/Aida/executor"

// NillExtension is an implementation of the executor.Extension interface
// ignoring all incoming events. It is mainly intended as a fall-back
// implementation when a no-op implementation is required, as well as an
// implementation that may be embedded in other extensions to avoid the
// need to implement all possible events.
type NilExtension struct{}

func (NilExtension) PreRun(executor.State, *executor.Context) error          { return nil }
func (NilExtension) PostRun(executor.State, *executor.Context, error) error  { return nil }
func (NilExtension) PreBlock(executor.State, *executor.Context) error        { return nil }
func (NilExtension) PostBlock(executor.State, *executor.Context) error       { return nil }
func (NilExtension) PreTransaction(executor.State, *executor.Context) error  { return nil }
func (NilExtension) PostTransaction(executor.State, *executor.Context) error { return nil }
