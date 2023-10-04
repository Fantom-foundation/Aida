package extension

import "github.com/Fantom-foundation/Aida/executor"

// NillExtension is an implementation of the executor.Extension interface
// ignoring all incoming events. It is mainly intended as a fall-back
// implementation when a no-op implementation is required, as well as an
// implementation that may be embedded in other extensions to avoid the
// need to implement all possible events.
type NilExtension[T any] struct{}

func (NilExtension[T]) PreRun(executor.State[T], *executor.Context) error          { return nil }
func (NilExtension[T]) PostRun(executor.State[T], *executor.Context, error) error  { return nil }
func (NilExtension[T]) PreBlock(executor.State[T], *executor.Context) error        { return nil }
func (NilExtension[T]) PostBlock(executor.State[T], *executor.Context) error       { return nil }
func (NilExtension[T]) PreTransaction(executor.State[T], *executor.Context) error  { return nil }
func (NilExtension[T]) PostTransaction(executor.State[T], *executor.Context) error { return nil }
