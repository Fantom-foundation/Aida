package extension

import "github.com/Fantom-foundation/Aida/executor"

// MakeArchiveGetter creates an extension for retrieving archive. Archive is assigned to context.Archive.
func MakeArchiveGetter[T any]() executor.Extension[T] {
	return &archiveGetter[T]{}
}

type archiveGetter[T any] struct {
	NilExtension[T]
}

// PreBlock sends needed archive to the processor.
func (r *archiveGetter[T]) PreBlock(state executor.State[T], context *executor.Context) error {
	var err error
	context.Archive, err = context.State.GetArchiveState(uint64(state.Block) - 1)
	if err != nil {
		return err
	}

	return nil
}
