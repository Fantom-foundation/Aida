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
func (r *archivePrepper) PreBlock(state executor.State[*substate.Substate], ctx *executor.Context) error {
	var err error
	ctx.Archive, err = ctx.State.GetArchiveState(uint64(state.Block) - 1)
	if err != nil {
		return err
	}

	return nil
}

// PreTransaction starts the transaction.
func (r *archivePrepper) PreTransaction(state executor.State[*substate.Substate], ctx *executor.Context) error {
	ctx.Archive.BeginTransaction(uint32(state.Transaction))
	return nil
}

// PostTransaction ends the transaction.
func (r *archivePrepper) PostTransaction(_ executor.State[*substate.Substate], ctx *executor.Context) error {
	ctx.Archive.EndTransaction()
	return nil
}

// PostBlock releases the Archive StateDb
func (r *archivePrepper) PostBlock(_ executor.State[*substate.Substate], ctx *executor.Context) error {
	ctx.Archive.Release()
	return nil
}
