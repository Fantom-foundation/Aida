package state_db_extensions

import (
	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/executor/extension"
)

// MakeArchivePrepper creates an extension for retrieving archive. Archive is assigned to context.Archive.
func MakeArchivePrepper() executor.Extension {
	return &archivePrepper{}
}

type archivePrepper struct {
	extension.NilExtension
}

// PreBlock sends needed archive to the processor.
func (r *archivePrepper) PreBlock(state executor.State, context *executor.Context) error {
	var err error
	context.Archive, err = context.State.GetArchiveState(uint64(state.Block) - 1)
	if err != nil {
		return err
	}

	return nil
}
