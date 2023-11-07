package executor

import (
	"fmt"

	"github.com/Fantom-foundation/Aida/rpc"
	"github.com/Fantom-foundation/Aida/utils"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/urfave/cli/v2"
)

func OpenRpcRecording(cfg *utils.Config, ctx *cli.Context) (Provider[*rpc.RequestAndResults], error) {
	iter, err := rpc.NewFileReader(ctx.Context, cfg.RpcRecordingFile)
	if err != nil {
		return nil, fmt.Errorf("cannot open rpc recording file; %v", err)
	}
	return openRpcRecording(iter, cfg, ctx), nil
}

func openRpcRecording(iter rpc.Iterator, cfg *utils.Config, ctxt *cli.Context) Provider[*rpc.RequestAndResults] {
	return rpcRequestProvider{
		ctxt:     ctxt,
		fileName: cfg.RpcRecordingFile,
		iter:     iter,
	}
}

type rpcRequestProvider struct {
	ctxt     *cli.Context
	fileName string
	iter     rpc.Iterator
}

func (r rpcRequestProvider) Run(from int, to int, consumer Consumer[*rpc.RequestAndResults]) error {
	var (
		block int
		ok    bool
	)

	for r.iter.Next() {
		if r.iter.Error() != nil {
			return fmt.Errorf("iterator returned error; %v", r.iter.Error())
		}

		req := r.iter.Value()

		if req == nil {
			return nil
		}

		// are we skipping requests?
		if req.Block < from {
			continue
		}

		if req.Block >= to {
			return nil
		}

		block, ok = findBlockNumber(req)
		if !ok {
			continue
		}

		if req.Query.MethodBase == "getBalance" && req.Error != nil {
			fmt.Sprintf("")
		}

		if err := consumer(TransactionInfo[*rpc.RequestAndResults]{block, 0, req}); err != nil {
			return err
		}
	}

	return nil
}

// findBlockNumber finds what block number request wants
func findBlockNumber(data *rpc.RequestAndResults) (int, bool) {
	if len(data.Query.Params) < 2 {
		return data.Block, true
	}

	str, ok := data.Query.Params[1].(string)
	if !ok {
		return data.Block, true
	}

	switch str {
	case "latest":
		return data.Block, true
	case "earliest":
		return 0, true
	case "pending":
		// pending block does not work
		return 0, false
	default:
		// botched params are not recorded, so this will  never panic
		block, err := hexutil.DecodeUint64(str)
		if err != nil {
			return 0, false
		}
		return int(block), true
	}
}

func (r rpcRequestProvider) Close() {
	r.iter.Close()
}
