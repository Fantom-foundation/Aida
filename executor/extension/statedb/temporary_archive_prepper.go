package statedb

import (
	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/executor/extension"
	"github.com/Fantom-foundation/Aida/rpc"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

// MakeTemporaryArchivePrepper creates an extension for retrieving temporary archive before every transaction.
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
	block, ok := findBlockNumber(state.Data)
	if !ok {
		ctx.Archive = nil
		return nil
	}

	ctx.Archive, err = ctx.State.GetArchiveState(uint64(block))
	if err != nil {
		return err
	}

	return nil
}

// PostTransaction releases temporary Archive.
func (r *temporaryArchivePrepper) PostTransaction(_ executor.State[*rpc.RequestAndResults], ctx *executor.Context) error {
	// Archive can be nil if invalid block number is passed
	if ctx.Archive == nil {
		return nil
	}
	ctx.Archive.Release()

	return nil
}

// findBlockNumber finds what block number request wants
func findBlockNumber(data *rpc.RequestAndResults) (int, bool) {
	l := len(data.Query.Params)
	if l < 2 {
		return data.Block, true
	}

	str, ok := data.Query.Params[l-1].(string)
	if !ok {
		return 0, false
	}

	switch str {
	case "pending":
		// pending does not work in opera, in this case the latest state is always returned
		fallthrough
	case "latest":
		return data.Block, true
	case "earliest":
		return 0, true

	default:
		// botched params are not recorded, so this will  never panic
		block := hexutil.MustDecodeUint64(str)
		return int(block), true
	}
}
