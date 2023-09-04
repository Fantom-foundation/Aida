package extension

import "github.com/Fantom-foundation/Aida/executor"

// NillExtension is an implementation of the executor.Extension interface
// ignoring all incoming events. It is mainly intended as a fall-back
// implementation when a no-op implementation is required, as well as an
// implementation that may be embedded in other extensions to avoid the
// need to implement all possible events.
type NilExtension struct{}

func (NilExtension) PreRun(executor.State) error          { return nil }
func (NilExtension) PostRun(executor.State, error) error  { return nil }
func (NilExtension) PreBlock(executor.State) error        { return nil }
func (NilExtension) PostBlock(executor.State) error       { return nil }
func (NilExtension) PreTransaction(executor.State) error  { return nil }
func (NilExtension) PostTransaction(executor.State) error { return nil }
