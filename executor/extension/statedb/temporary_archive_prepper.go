package statedb

import (
	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/executor/extension"
	"github.com/Fantom-foundation/Aida/rpc"
)

// MakeTemporaryArchivePrepper creates an extension for retrieving temporary archive before every txcontext.
// Archive is assigned to context.Archive. Archive is released after transaction.
func MakeTemporaryArchivePrepper() executor.Extension[*rpc.RequestAndResults] {
	return &temporaryArchivePrepper{}
}

type temporaryArchivePrepper struct {
	extension.NilExtension[*rpc.RequestAndResults]
}

// PreTransaction creates temporary archive that is released after transaction is executed.
func (r *temporaryArchivePrepper) PreTransaction(state executor.State[*rpc.RequestAndResults], ctx *executor.Context) error {
	var err error
	ctx.Archive, err = ctx.State.GetArchiveState(uint64(state.Data.RequestedBlock))
	if err != nil {
		return err
	}

	return nil
}

// PostTransaction releases temporary Archive.
func (r *temporaryArchivePrepper) PostTransaction(_ executor.State[*rpc.RequestAndResults], ctx *executor.Context) error {
	ctx.Archive.Release()

	return nil
}
