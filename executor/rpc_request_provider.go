package executor

import (
	"fmt"

	"github.com/Fantom-foundation/Aida/rpc"
	"github.com/Fantom-foundation/Aida/utils"
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

		if err := consumer(TransactionInfo[*rpc.RequestAndResults]{req.Block, 0, req}); err != nil {
			return err
		}
	}

	return nil
}

func (r rpcRequestProvider) Close() {
	r.iter.Close()
}
