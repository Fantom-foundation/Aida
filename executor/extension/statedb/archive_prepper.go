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
func (r *archivePrepper[T]) PreBlock(state executor.State[T], context *executor.Context) error {
	var err error
	context.Archive, err = context.State.GetArchiveState(uint64(state.Block) - 1)
	if err != nil {
		return err
	}

	return nil
}
