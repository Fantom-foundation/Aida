package statedb

import (
	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/executor/extension"
	"github.com/Fantom-foundation/Aida/executor/transaction/substate_transaction"
)

// MakeArchivePrepper creates an extension for retrieving archive. Archive is assigned to context.Archive.
func MakeArchivePrepper() executor.Extension[substate_transaction.SubstateData] {
	return &archivePrepper{}
}

type archivePrepper struct {
	extension.NilExtension[substate_transaction.SubstateData]
}

// PreBlock sends needed archive to the processor.
func (r *archivePrepper) PreBlock(state executor.State[substate_transaction.SubstateData], ctx *executor.Context) error {
	var err error
	ctx.Archive, err = ctx.State.GetArchiveState(uint64(state.Block) - 1)
	if err != nil {
		return err
	}

	return nil
}

// PostBlock releases the Archive StateDb
func (r *archivePrepper) PostBlock(_ executor.State[substate_transaction.SubstateData], ctx *executor.Context) error {
	ctx.Archive.Release()
	return nil
}
