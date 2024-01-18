package statedb

import (
	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/executor/extension"
	"github.com/Fantom-foundation/Aida/rpc"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

// MakeTemporaryArchivePrepper creates an extension for retrieving temporary archive before every txcontext.
// Archive is assigned to context.Archive. Archive is released after txcontext.
func MakeTemporaryArchivePrepper() executor.Extension[*rpc.RequestAndResults] {
	return &temporaryArchivePrepper{}
}

type temporaryArchivePrepper struct {
	extension.NilExtension[*rpc.RequestAndResults]
}

// PreTransaction creates temporary archive that is released after txcontext is executed.
func (r *temporaryArchivePrepper) PreTransaction(state executor.State[*rpc.RequestAndResults], ctx *executor.Context) error {
	block := findBlockNumber(state.Data)

	var err error
	ctx.Archive, err = ctx.State.GetArchiveState(block)
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

// findBlockNumber finds what block number request wants
func findBlockNumber(data *rpc.RequestAndResults) uint64 {
	l := len(data.Query.Params)
	var block uint64
	if data.Response != nil {
		block = data.Response.BlockID
	} else {
		block = data.Error.BlockID
	}
	if l < 2 {
		return block
	}

	str := data.Query.Params[l-1].(string)

	switch str {
	case "pending":
		// pending should be treated as latest
		fallthrough
	case "latest":
		return block
	case "earliest":
		return 0

	default:
		// botched params are not recorded, so this will  never panic
		return hexutil.MustDecodeUint64(str)
	}
}
