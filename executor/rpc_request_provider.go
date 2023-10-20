package executor

import (
	"fmt"

	"github.com/Fantom-foundation/Aida/rpc_iterator"
	"github.com/Fantom-foundation/Aida/utils"
	"github.com/urfave/cli/v2"
)

func OpenRpcRecording(cfg *utils.Config, ctxt *cli.Context) (Provider[*rpc_iterator.RequestWithResponse], error) {
	iter, err := rpc_iterator.NewFileReader(ctxt.Context, cfg.RPCRecordingFile)
	if err != nil {
		return nil, fmt.Errorf("cannot open rpc recording file; %v", err)
	}
	return rpcRequestProvider{
		ctxt:     ctxt,
		fileName: cfg.RPCRecordingFile,
		iter:     iter,
	}, nil
}

type rpcRequestProvider struct {
	ctxt     *cli.Context
	fileName string
	iter     *rpc_iterator.FileReader
}

func (r rpcRequestProvider) Run(from int, to int, consumer Consumer[*rpc_iterator.RequestWithResponse]) error {
	var blockNumber int

	defer r.iter.Release()
	for r.iter.Next() {
		if r.iter.Error() != nil {
			return fmt.Errorf("iterator returned error; %v", r.iter.Error())
		}

		req := r.iter.Value()

		if req == nil {
			return nil
		}

		if req.Response != nil {
			blockNumber = int(req.Response.BlockID)
		} else {
			blockNumber = int(req.Error.BlockID)
		}

		// are we skipping requests?
		if blockNumber < from {
			continue
		}

		if blockNumber >= to {
			return nil
		}

		if err := consumer(TransactionInfo[*rpc_iterator.RequestWithResponse]{blockNumber, 0, req}); err != nil {
			return err
		}
	}

	return nil
}

func (r rpcRequestProvider) Close() {
	r.iter.Close()
}
