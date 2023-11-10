package profiler

import (
	"runtime"

	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/executor/extension"
)

// MakeThreadLocker creates an executor.Extension locks the executor thread
// to a single OS level thread for the duration of the execution. This can
// have significant performance impact on EVM or StateDB executions.
func MakeThreadLocker[T any]() executor.Extension[T] {
	return threadLocker[T]{}
}

type threadLocker[T any] struct {
	extension.NilExtension[T]
}

func (threadLocker[T]) PreRun(executor.State[T], *executor.Context) error {
	runtime.LockOSThread()
	return nil
}

func (threadLocker[T]) PostRun(executor.State[T], *executor.Context, error) error {
	runtime.UnlockOSThread()
	return nil
}

func (threadLocker[T]) PreBlock(executor.State[T], *executor.Context) error {
	runtime.LockOSThread()
	return nil
}

func (threadLocker[T]) PostBlock(executor.State[T], *executor.Context) error {
	runtime.UnlockOSThread()
	return nil
}

func (threadLocker[T]) PreTransaction(executor.State[T], *executor.Context) error {
	runtime.LockOSThread()
	return nil
}

func (threadLocker[T]) PostTransaction(executor.State[T], *executor.Context) error {
	runtime.UnlockOSThread()
	return nil
}
