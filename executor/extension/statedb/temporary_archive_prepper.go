package statedb

import (
	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/executor/extension"
)

// MakeTemporaryArchivePrepper creates an extension for retrieving temporary archive before every transaction.
// Archive is assigned to context.Archive. Archive is released after transaction.
func MakeTemporaryArchivePrepper[T any]() executor.Extension[T] {
	return &temporaryArchivePrepper[T]{}
}

type temporaryArchivePrepper[T any] struct {
	extension.NilExtension[T]
}

// PreTransaction creates temporary archive that is released after transaction is executed.
func (r *temporaryArchivePrepper[T]) PreTransaction(state executor.State[T], ctx *executor.Context) error {
	var err error
	ctx.Archive, err = ctx.State.GetArchiveState(uint64(state.Block))
	if err != nil {
		return err
	}

	return nil
}

// PostTransaction releases temporary Archive.
func (r *temporaryArchivePrepper[T]) PostTransaction(_ executor.State[T], ctx *executor.Context) error {
	ctx.Archive.Release()

	return nil
}
