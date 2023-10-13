package statedb

import (
	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/executor/extension"
	substate "github.com/Fantom-foundation/Substate"
)

// MakeArchivePrepper creates an extension for retrieving archive. Archive is assigned to context.Archive.
func MakeArchivePrepper() executor.Extension[*substate.Substate] {
	return &archivePrepper{}
}

type archivePrepper struct {
	extension.NilExtension[*substate.Substate]
}

// PreBlock sends needed archive to the processor.
func (r *archivePrepper) PreBlock(state executor.State[*substate.Substate], context *executor.Context) error {
	var err error
	context.Archive, err = context.State.GetArchiveState(uint64(state.Block) - 1)
	if err != nil {
		return err
	}

	return nil
}
