package extension

import "github.com/Fantom-foundation/Aida/executor"

// MakeArchiveGetter creates an extension for retrieving archive. Archive is assigned to context.Archive.
func MakeArchiveGetter() executor.Extension {
	return &archiveGetter{}
}

type archiveGetter struct {
	NilExtension
}

// PreBlock sends needed archive to the processor.
func (r *archiveGetter) PreBlock(state executor.State, context *executor.Context) error {
	var err error
	context.Archive, err = context.State.GetArchiveState(uint64(state.Block) - 1)
	if err != nil {
		return err
	}

	return nil
}
