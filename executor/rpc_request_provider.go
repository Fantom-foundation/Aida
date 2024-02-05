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
	var blockNumber int
	var number int

	fmt.Println("from")
	fmt.Println(from)
	fmt.Println("to")
	fmt.Println(to)

	for r.iter.Next() {
		if r.iter.Error() != nil {
			return fmt.Errorf("iterator returned error; %v", r.iter.Error())
		}

		req := r.iter.Value()

		if req == nil {
			return nil
		}

		// get logs is not yet implemented, skip these for now
		if req.Query.MethodBase == "getLogs" {
			continue
		}

		if req.Response != nil {
			blockNumber = int(req.Response.BlockID)
		} else {
			blockNumber = int(req.Error.BlockID)
		}

		blockNumber = 10

		// are we skipping requests?
		if blockNumber < from {
			fmt.Println("skip")
			continue
		}

		if blockNumber > to {
			fmt.Println("return nil")
			fmt.Println(blockNumber)
			return nil
		}

		if err := consumer(TransactionInfo[*rpc.RequestAndResults]{blockNumber, 0, req}); err != nil {
			fmt.Println("err")
			fmt.Println(err)
			return err
		}
		number++
	}

	fmt.Println("number")
	fmt.Println(number)

	return nil
}

func (r rpcRequestProvider) Close() {
	r.iter.Close()
}
