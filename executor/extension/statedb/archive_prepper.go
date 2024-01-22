package statedb

import (
	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/executor/extension"
)

// MakeArchivePrepper creates an extension for retrieving archive. Archive is assigned to context.Archive.
func MakeArchivePrepper[T any]() executor.Extension[T] {
	return &archivePrepper[T]{}
}

type archivePrepper[T any] struct {
	extension.NilExtension[T]
}

// PreBlock sends needed archive to the processor.
func (r *archivePrepper[T]) PreBlock(state executor.State[T], ctx *executor.Context) error {
	var err error
	ctx.Archive, err = ctx.State.GetArchiveState(uint64(state.Block) - 1)
	if err != nil {
		return err
	}

	return nil
}

// PostBlock releases the Archive StateDb
func (r *archivePrepper[T]) PostBlock(_ executor.State[T], ctx *executor.Context) error {
	ctx.Archive.Release()
	return nil
}
